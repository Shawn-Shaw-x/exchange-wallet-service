package worker

import (
	"context"
	"errors"
	"exchange-wallet-service/common/retry"
	"exchange-wallet-service/common/tasks"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	"exchange-wallet-service/database/constant"
	"exchange-wallet-service/rpcclient"
	"exchange-wallet-service/rpcclient/chainsunion"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"math/big"
	"time"
)

/*
交易发现器，负责处理交易 channel 中
推送过来的交易，
处理充值、提现、归集、内部交易的链上发现、入库
*/
type Finder struct {
	/*同步器*/
	BaseSynchronizer *BaseSynchronizer

	/*确认位*/
	confirms uint8
	/*最新区块*/
	latestHeader   rpcclient.BlockHeader
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	/*协程管理*/
	tasks tasks.Group
}

/*新建交易发现器*/
func NewFinder(synchronizer *BaseSynchronizer, cfg config.Config, shutdown context.CancelCauseFunc) (*Finder, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Finder{
		BaseSynchronizer: synchronizer,
		confirms:         uint8(cfg.ChainNode.Confirmations),
		resourceCtx:      resCtx,
		resourceCancel:   resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("fail to execute finder tasks: %w", err))
		}},
	}, nil
}

/*启动交易发现器（消费者）*/
func (f *Finder) Start() error {
	/*协程异步处理任务*/
	f.tasks.Go(func() error {
		log.Info("handle deposit task start")

		for {
			select {
			case <-f.resourceCtx.Done():
				log.Info("handle deposit task done")
				return nil
			case batch := <-f.BaseSynchronizer.businessChannels:
				log.Info("deposit business channel", "batch length", len(batch))

				/* 实现所有交易处理*/
				if err := f.handleBatch(batch); err != nil {
					log.Info("failed to handle batch, stopping L2 Synchronizer:", "err", err)
					return fmt.Errorf("failed to handle batch, stopping L2 Synchronizer: %w", err)
				}
			}
		}

		return nil
	})
	return nil
}

/*停止发现器*/
func (f *Finder) Stop() error {
	var result error
	f.resourceCancel()
	log.Info("stop finder......")
	if err := f.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await finder %w", err))
		return result
	}
	log.Info("stop finder success")
	return nil
}

/*
处理所有推送过来交易（一批次，所有有关项目方的都在这个 map 中）
充值：库中原来没有，入库、更新余额。库中的充值更新确认位
提现：库中原来有记录（项目方提交的），更新状态为已发现
归集：库中原来有记录（项目方提交的），更新状态为已发现
热转冷、冷转热：库中原来有记录（项目方提交的），更新状态为已发现
交易流水：入库 transaction 表
*/
func (f *Finder) handleBatch(batch map[string]*BatchTransactions) error {
	/*查出项目方列表*/
	businessList, err := f.BaseSynchronizer.database.Business.QueryBusinessList()
	if err != nil {
		log.Error("failed to query business list", "err", err)
		return err
	}
	if businessList == nil || len(businessList) <= 0 {
		err := fmt.Errorf("failed to query business list")
		return err
	}

	/*项目方分别处理*/
	for _, business := range businessList {
		_, exists := batch[business.BusinessUid]
		if !exists {
			/*不存在则跳过*/
			continue
		}
		/*存库用*/
		var (
			/*流水表*/
			transactionFlowList []*database.Transactions
			/*充值表*/
			depositList []*database.Deposits
			/*提现表*/
			withdrawList []*database.Withdraws
			/*内部表*/
			internals []*database.Internals
			/*余额表*/
			balances []*database.TokenBalance
		)
		log.Info("handle business flow", "businessId", business.BusinessUid, "chainLatestBlock", batch[business.BusinessUid].BlockHeight, "txn", len(batch[business.BusinessUid].Transactions))
		for _, tx := range batch[business.BusinessUid].Transactions {
			/*每笔交易分别处理*/
			log.Info("Request transaction from chain account", "txHash", tx.Hash, "fromAddress", tx.FromAddress)
			txItem, err := f.BaseSynchronizer.rpcClient.GetTransactionByHash(tx.Hash)
			if err != nil {
				log.Info("failed to get transaction by hash", "hash", tx.Hash, "err", err)
				return err
			}
			if txItem == nil {
				err := fmt.Errorf("GetTransactionByHash txItem is nil: TxHash = %s", tx.Hash)
				return err
			}
			amountBigInt, _ := new(big.Int).SetString(txItem.Value, 10)
			log.Info("transaction amount", "amountBigInt", amountBigInt, "FromAddress", tx.FromAddress, "toAddress", tx.ToAddress, "TokenAddress", tx.TokenAddress, "txType", tx.TxType)

			/*代币余额，ETH 主币余额*/
			balances = append(
				balances,
				&database.TokenBalance{
					FromAddress:  common.HexToAddress(tx.FromAddress),
					ToAddress:    common.HexToAddress(tx.ToAddress),
					TokenAddress: common.HexToAddress(tx.TokenAddress),
					Balance:      amountBigInt,
					TxType:       tx.TxType,
				},
			)

			log.Info("get transaction success", "txHash", txItem.Hash)
			transactionFlow, err := f.BuildTransaction(tx, txItem)
			if err != nil {
				log.Info("handle  transaction fail", "err", err)
				return err
			}
			/*放入交易流水列表，等待入库*/
			transactionFlowList = append(transactionFlowList, transactionFlow)

			switch tx.TxType {
			/*充值*/
			case constant.TxTypeDeposit:
				depositItem, _ := f.HandleDeposit(tx, txItem)
				depositList = append(depositList, depositItem)
				break
			/*提现*/
			case constant.TxTypeWithdraw:
				withdrawItem, _ := f.HandleWithdraw(tx, txItem)
				withdrawList = append(withdrawList, withdrawItem)
				break
			/*内部（归集、转冷、转热）*/
			case constant.TxTypeCollection, constant.TxTypeCold2Hot, constant.TxTypeHot2Cold:
				internelItem, _ := f.HandleInternalTx(tx, txItem)
				internals = append(internals, internelItem)
				break
			default:
				break
			}
		}
		/*数据库重试策略*/
		retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
		/*重试*/
		if _, err := retry.Do[interface{}](f.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
			/*事务*/
			if err := f.BaseSynchronizer.database.Transaction(func(tx *database.DB) error {
				/* 1. 充值业务处理*/
				if len(depositList) > 0 {
					log.Info("Store deposit transaction success", "totalTx", len(depositList))
					if err := tx.Deposits.StoreDeposits(business.BusinessUid, depositList); err != nil {
						return err
					}
				}
				/* 2. 充值确认位处理*/
				if err := tx.Deposits.UpdateDepositsConfirms(business.BusinessUid, batch[business.BusinessUid].BlockHeight, uint64(f.confirms)); err != nil {
					log.Info("Handle confims fail", "totalTx", "err", err)
					return err
				}
				/* 3. 余额处理*/
				if len(balances) > 0 {
					log.Info("Handle balances transaction success", "totalTx", len(balances))
					if err := tx.Balances.UpdateOrCreate(business.BusinessUid, balances); err != nil {
						return err
					}
				}
				/* 4. 提现状态处理*/
				/*todo 还需保存区块号、区块 hash，不然回滚会有 bug*/
				if len(withdrawList) > 0 {
					if err := tx.Withdraws.UpdateWithdrawStatusByTxHash(business.BusinessUid, constant.TxStatusWalletDone, withdrawList); err != nil {
						return err
					}
				}

				/* 5. 内部交易状态处理*/
				/*todo 还需保存区块号、区块 hash，不然回滚会有 bug*/
				if len(internals) > 0 {
					if err := tx.Internals.UpdateInternalStatusByTxHash(business.BusinessUid, constant.TxStatusWalletDone, internals); err != nil {
						return err
					}
				}

				/* 6. 交易流水表入库*/
				if len(transactionFlowList) > 0 {
					if err := tx.Transactions.StoreTransactions(business.BusinessUid, transactionFlowList, uint64(len(transactionFlowList))); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				log.Error("unable to persist batch", "err", err)
				return nil, err
			}
			return nil, nil
		}); err != nil {
			return err
		}

	}
	return nil

}

/*构建交易流水记录*/
func (f *Finder) BuildTransaction(tx *Transaction, txMsg *chainsunion.TxMessage) (*database.Transactions, error) {
	txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Value, 10)
	transationTx := &database.Transactions{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		Hash:         common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		Fee:          txFee,
		Status:       constant.TxStatusSuccess, /* 充值扫到交易后则为成功*/
		Amount:       txAmount,
		TxType:       tx.TxType,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return transationTx, nil
}

/*充值记录构建*/
func (f *Finder) HandleDeposit(tx *Transaction, txMsg *chainsunion.TxMessage) (*database.Deposits, error) {
	//txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Value, 10)
	depositTx := &database.Deposits{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		TxHash:       common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		MaxFeePerGas: txMsg.Fee,
		Amount:       txAmount,
		Status:       constant.TxStatusSuccess, /* 充值扫到交易后则为成功*/
		Timestamp:    uint64(time.Now().Unix()),
	}
	return depositTx, nil
}

func (f *Finder) HandleWithdraw(tx *Transaction, txMsg *chainsunion.TxMessage) (*database.Withdraws, error) {
	//txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Value, 10)
	withdrawTx := &database.Withdraws{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		TxHash:       common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		MaxFeePerGas: txMsg.Fee,
		Amount:       txAmount,
		Status:       constant.TxStatusBroadcasted, /*扫到交易后则为已广播*/
		Timestamp:    uint64(time.Now().Unix()),
	}
	return withdrawTx, nil
}

func (f *Finder) HandleInternalTx(tx *Transaction, txMsg *chainsunion.TxMessage) (*database.Internals, error) {
	//txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Value, 10)
	internalTx := &database.Internals{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		TxHash:       common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		MaxFeePerGas: txMsg.Fee,
		Amount:       txAmount,
		Status:       constant.TxStatusBroadcasted, /*扫到交易后则为已广播*/
		Timestamp:    uint64(time.Now().Unix()),
	}
	return internalTx, nil
}
