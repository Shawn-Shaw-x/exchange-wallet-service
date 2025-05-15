package worker

import (
	"context"
	"errors"
	"exchange-wallet-service/common/bigint"
	"exchange-wallet-service/common/retry"
	"exchange-wallet-service/common/tasks"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	"exchange-wallet-service/rpcclient"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"time"
)

/*回滚任务*/
type Fallback struct {
	BaseSynchronizer *BaseSynchronizer
	database         *database.DB
	rpcClient        *rpcclient.ChainsUnionRpcClient
	resourceCtx      context.Context
	resourceCancel   context.CancelFunc
	tasks            tasks.Group
	ticker           *time.Ticker
	confirmations    uint64
}

/*新建回滚任务*/
func NewFallback(cfg *config.Config, db *database.DB, rpcClient *rpcclient.ChainsUnionRpcClient, syncer *BaseSynchronizer, shutdown context.CancelCauseFunc) (*Fallback, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Fallback{
		BaseSynchronizer: syncer,
		database:         db,
		rpcClient:        rpcClient,
		resourceCtx:      resCtx,
		resourceCancel:   resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in fallback: %w", err))
		}},
		ticker:        time.NewTicker(time.Second * 3),
		confirmations: uint64(cfg.ChainNode.Confirmations),
	}, nil
}

/*关闭*/
func (fb *Fallback) Stop() error {
	var result error
	fb.resourceCancel()
	fb.ticker.Stop()
	log.Info("stop fallback......")
	if err := fb.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await fallback %w", err))
		return result
	}
	log.Info("stop fallback success")
	return nil
}

/*启动*/
func (fb *Fallback) Start() error {
	log.Info("start fallback.........")
	fb.tasks.Go(func() error {
		for {
			select {
			case <-fb.ticker.C:
				if fb.BaseSynchronizer.isFallback {
					log.Info("fallback task", "synchronizer fallback handle", fb.BaseSynchronizer.fallbackBlockHeader.Number)
					if err := fb.onFallback(fb.BaseSynchronizer.fallbackBlockHeader); err != nil {
						log.Error("failed to notify fallback", "err", err)
					}
					dbLatestBlockHeader, err := fb.database.Blocks.LatestBlocks()
					if err != nil {
						log.Error("query latest block fail", "err", err)
					}
					/*传入新的 dbLatestBlockHeader，重新启动扫块*/
					fb.BaseSynchronizer.blockBatch = rpcclient.NewBatchBlock(fb.rpcClient, dbLatestBlockHeader, big.NewInt(int64(fb.confirmations)))
					/*处理完回滚，取消回滚状态*/
					fb.BaseSynchronizer.isFallback = false
					fb.BaseSynchronizer.fallbackBlockHeader = nil
				}
			case <-fb.resourceCtx.Done():
				log.Info("stop fallback.........")
				return nil
			}
		}
	})
	return nil
}

/*回滚区块表、充值、提现、内部、流水、余额表处理*/
func (fb *Fallback) onFallback(fallbackBlockHeader *rpcclient.BlockHeader) error {
	reorgBlockHeaders, chainBlocks, entryBlockHeader, err := fb.findFallbackEntry(fallbackBlockHeader)
	if err != nil {
		log.Error("failed to find fallback entry", "err", err)
		return err
	}

	businessList, err := fb.database.Business.QueryBusinessList()
	if err != nil {
		log.Error("failed to query business list", "err", err)
		return err
	}

	var fallbackBalances []*database.TokenBalance
	for _, business := range businessList {
		log.Info("handle business", "businessUid", business.BusinessUid)
		/*范围内的交易记录*/
		transactionList, err := fb.database.Transactions.QueryFallBackTransactions(business.BusinessUid, entryBlockHeader.Number, fallbackBlockHeader.Number)
		if err != nil {
			log.Error("failed to query fallback transactions", "err", err)
			return err
		}
		for _, transaction := range transactionList {
			fallbackBalances = append(fallbackBalances, &database.TokenBalance{
				FromAddress:  transaction.FromAddress,
				ToAddress:    transaction.ToAddress,
				TokenAddress: transaction.TokenAddress,
				Balance:      transaction.Amount,
				TxType:       transaction.TxType,
			})
		}
	}

	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](fb.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
		if err := fb.database.Transaction(func(tx *database.DB) error {
			if len(reorgBlockHeaders) > 0 {
				/*被回滚的区块备份*/
				if err := tx.ReorgBlocks.StoreReorgBlocks(reorgBlockHeaders); err != nil {
					log.Error("failed to store reorg blocks", "err", err)
					return err
				}
				log.Info("store reorg block success", "totalTx", len(reorgBlockHeaders))
			}

			if len(chainBlocks) > 0 {
				if err := tx.Blocks.DeleteBlocksByNumber(chainBlocks); err != nil {
					return err
				}
				log.Info("delete block success", "totalTx", len(chainBlocks))
			}
			/*存在回滚块，标记其中交易（根据交易通知业务层去让其做逆向交易）*/
			if fallbackBlockHeader.Number.Cmp(entryBlockHeader.Number) > 0 {
				for _, business := range businessList {
					/*充值回滚*/
					if err := tx.Deposits.HandleFallBackDeposits(business.BusinessUid, entryBlockHeader.Number, fallbackBlockHeader.Number); err != nil {
						log.Error("failed to handle fallback deposits", "err", err)
						return err
					}
					/*提现回滚*/
					if err := tx.Withdraws.HandleFallBackWithdraw(business.BusinessUid, entryBlockHeader.Number, fallbackBlockHeader.Number); err != nil {
						log.Error("failed to handle fallback withdraws", "err", err)
						return err
					}

					/*内部交易回滚*/
					if err := tx.Internals.HandleFallBackInternals(business.BusinessUid, entryBlockHeader.Number, fallbackBlockHeader.Number); err != nil {
						log.Error("failed to handle fallback internals", "err", err)
						return err
					}
					/*流水表回滚*/
					if err := tx.Transactions.HandleFallBackTransactions(business.BusinessUid, entryBlockHeader.Number, fallbackBlockHeader.Number); err != nil {
						log.Error("failed to handle fallback transactions", "err", err)
						return err
					}
					/*余额回滚*/
					if err := tx.Balances.UpdateFallBackBalance(business.BusinessUid, fallbackBalances); err != nil {
						log.Error("failed to update fallback balance", "err", err)
						return err
					}
				}
			}
			return nil
		}); err != nil {
			log.Error("unable to persist fallback batch", "err", err)
			return nil, err
		}
		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}

/*找到回滚初始块并返回*/
func (fb *Fallback) findFallbackEntry(fallbackBlockHeader *rpcclient.BlockHeader) ([]database.ReorgBlocks, []database.Blocks, *rpcclient.BlockHeader, error) {
	var reorgBlockHeaders []database.ReorgBlocks
	var chainBlocks []database.Blocks

	lastBlockHeader := fallbackBlockHeader

	/*寻找到回滚的分叉点*/
	for {
		/*往回查找*/
		lastBlockNumber := new(big.Int).Sub(lastBlockHeader.Number, bigint.One)
		log.Info("start get block header info...", "last block number", lastBlockNumber)

		/*链上这个块*/
		chainBlockHeader, err := fb.rpcClient.GetBlockHeader(lastBlockNumber)
		if err != nil {
			log.Warn("failed to get block header info from chain", "err", err)
			return nil, nil, nil, fmt.Errorf("failed to get block header info from chain: %w", err)
		}
		/*数据库中*/
		dbBlockHeader, err := fb.database.Blocks.QueryBlocksByNumber(lastBlockNumber)
		if err != nil {
			log.Warn("failed to get block header info from database", "err", err)
			return nil, nil, nil, fmt.Errorf("failed to get block header info from database: %w", err)
		}
		log.Info("query blocks from database success", "last block number", lastBlockNumber)
		/*需要删除的*/
		chainBlocks = append(chainBlocks, database.Blocks{
			Hash:       dbBlockHeader.Hash,
			ParentHash: dbBlockHeader.ParentHash,
			Number:     dbBlockHeader.Number,
			Timestamp:  dbBlockHeader.Timestamp,
		})
		/*需要备份的*/
		reorgBlockHeaders = append(reorgBlockHeaders, database.ReorgBlocks{
			Hash:       dbBlockHeader.Hash,
			ParentHash: dbBlockHeader.ParentHash,
			Number:     dbBlockHeader.Number,
			Timestamp:  dbBlockHeader.Timestamp,
		})
		log.Info("lastBlockHeader chainBlockHeader", "lastBlockParentHash", lastBlockHeader.ParentHash, "lastBlockNumber", lastBlockHeader.Number, "chainBlockHash", chainBlockHeader.Hash, "chainBlockHeaderNumber", chainBlockHeader.Number)

		/*已找到分叉点，正常终止*/
		if lastBlockHeader.ParentHash == chainBlockHeader.Hash {
			lastBlockHeader = chainBlockHeader
			return reorgBlockHeaders, chainBlocks, chainBlockHeader, nil
		}
		/*往前移动*/
		lastBlockHeader = chainBlockHeader
	}
}
