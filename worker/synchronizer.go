package worker

import (
	"context"
	"errors"
	"exchange-wallet-service/common/clock"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	"exchange-wallet-service/database/constant"
	"exchange-wallet-service/rpcclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"time"
)

/*同步器*/
type BaseSynchronizer struct {
	loopInterval     time.Duration
	headerBufferSize uint64
	/*核心管道，存放一批次的交易，map 中的 key 为业务方 id*/
	businessChannels chan map[string]*BatchTransactions

	rpcClient *rpcclient.ChainsUnionRpcClient
	/*批量扫块工具*/
	blockBatch *rpcclient.BatchBlock
	database   *database.DB

	/*本次扫描的区块头切片*/
	headers []rpcclient.BlockHeader
	/*发生回滚的区块*/
	fallbackBlockHeader *rpcclient.BlockHeader
	worker              *clock.LoopFn
	isFallback          bool
}

/*单个交易*/
type Transaction struct {
	BusinessId     string
	BlockNumber    *big.Int
	FromAddress    string
	ToAddress      string
	Hash           string
	TokenAddress   string
	ContractWallet string
	TxType         constant.TransactionType
}

/*一批交易*/
type BatchTransactions struct {
	BlockHeight  uint64
	BatchId      string
	Transactions []*Transaction
}

/*新建同步器*/
func NewSynchronizer(cfg *config.Config, db *database.DB, rpcClient *rpcclient.ChainsUnionRpcClient, shutdown context.CancelCauseFunc) (*BaseSynchronizer, error) {
	/*获取数据库中最新区块*/
	dbLatestBlockHeader, err := db.Blocks.LatestBlocks()
	if err != nil {
		log.Error("get latest block from database fail")
		return nil, err
	}
	var fromHeader *rpcclient.BlockHeader

	if dbLatestBlockHeader != nil {
		/*库中有最新区块，获取库中最新区块*/
		log.Info("sync bock", "number", dbLatestBlockHeader.Number, "hash", dbLatestBlockHeader.Hash)
		fromHeader = dbLatestBlockHeader
	} else if cfg.ChainNode.StartingHeight > 0 {
		/*库中没区块，但是配置了开始区块，获取配置区块*/
		chainLatestBlockHeader, err := rpcClient.GetBlockHeader(big.NewInt(int64(cfg.ChainNode.StartingHeight)))
		if err != nil {
			log.Error("get chain latest block header fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	} else {
		/*库中没区块，且配置文件中没配置，获取最新区块*/
		chainLatestBlockHeader, err := rpcClient.GetBlockHeader(nil)
		if err != nil {
			log.Error("get chain latest block header fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	}

	/*构造同步器*/
	baseSynchronizer := &BaseSynchronizer{
		loopInterval:        cfg.ChainNode.SynchronizerInterval,
		headerBufferSize:    cfg.ChainNode.BlocksStep,
		businessChannels:    make(chan map[string]*BatchTransactions),
		rpcClient:           rpcClient,
		blockBatch:          rpcclient.NewBatchBlock(rpcClient, fromHeader, big.NewInt(int64(cfg.ChainNode.Confirmations))),
		database:            db,
		isFallback:          false,
		fallbackBlockHeader: nil,
	}
	return baseSynchronizer, nil
}

/*启动同步器*/
func (syncer *BaseSynchronizer) Start() error {
	if syncer.worker != nil {
		return errors.New("already started")
	}
	/*定时任务*/
	syncer.worker = clock.NewLoopFn(clock.SystemClock, syncer.tick, func() error {
		log.Info("shutting down synchronizer produce...")
		close(syncer.businessChannels)
		return nil
	}, syncer.loopInterval)
	return nil
}

/*停止同步器*/
func (syncer *BaseSynchronizer) Stop() error {
	if syncer.worker != nil {
		return nil
	}
	return syncer.worker.Close()
}

/*同步任务*/
func (syncer *BaseSynchronizer) tick(_ context.Context) {
	/*本次任务还在处理，跳过获取，直接处理区块*/
	if len(syncer.headers) > 0 {
		log.Info("retrying previous batch")
	} else {
		/*  调用 NextHeaders 获取新的 headers 批次*/
		newHeaders, fallBackHeader, isReorg, err := syncer.blockBatch.NextHeaders(syncer.headerBufferSize)
		if err != nil {
			/*链重组(发生回滚)或者 fallback 回滚标记*/
			if isReorg && errors.Is(err, rpcclient.ErrBlockFallBack) {
				/*非回滚状态（第一次发生回滚状态），则标记回滚状态和回头区块*/
				if !syncer.isFallback {
					log.Warn("found block fallback, start fallback task")
					syncer.isFallback = true
					syncer.fallbackBlockHeader = fallBackHeader
				} else {
					log.Warn("the block fallback, fallback task handling it now")
				}
			} else {
				log.Error("error querying for headers", "err", err)
			}
		} else if len(newHeaders) == 0 {
			/*正常，但未扫到区块*/
			log.Warn("no new headers. syncer at head?")
		} else {
			/*没出错，正常处理这一批次 headers */
			syncer.headers = newHeaders
			log.Info("find new block headers success", "headers size", len(syncer.headers))
		}
	}

	/*处理这一批次区块*/
	err := syncer.processBatch(syncer.headers)
	/*成功则清空 headers，进入到下一轮*/
	if err == nil {
		syncer.headers = nil
	}
}

/*
批处理区块，
根据区块头获取区块内的交易，
按项目方 id 进行分类，打上标记，放入 BusinessChannel 中
*/
func (syncer *BaseSynchronizer) processBatch(headers []rpcclient.BlockHeader) error {
	/*无数据，无须处理*/
	if len(headers) == 0 {
		return nil
	}
	/*项目方的 map*/
	businessTxsMap := make(map[string]*BatchTransactions)
	/*存库用*/
	blockHeaders := make([]database.Blocks, len(headers))

	/*按区块处理*/
	for i, header := range headers {
		log.Info("sync block data", "height", headers[i].Number)
		blockHeaders[i] = database.Blocks{
			Hash:       header.Hash,
			ParentHash: header.ParentHash,
			Number:     header.Number,
			Timestamp:  header.Timestamp,
		}
		/*获取此块交易*/
		txList, err := syncer.rpcClient.GetBlockInfo(header.Number)
		if err != nil {
			log.Error("get block info fail", "err", err)
			return err
		}
		/*数据库中查询项目方列表*/
		businessList, err := syncer.database.Business.QueryBusinessList()
		if err != nil {
			log.Error("get business list fail", "err", err)
			return err
		}

		/*根据项目方进行分类处理*/
		for _, business := range businessList {
			/*某个项目方的这批次的交易*/
			var businessTransactions []*Transaction
			/*每个项目方，遍历这个交易中的全量交易*/
			for _, tx := range txList {
				toAddress := common.HexToAddress(tx.To)
				fromAddress := common.HexToAddress(tx.From)
				/*库中是否存在 to 地址和 to 地址类型*/
				existToAddress, toAddressType := syncer.database.Address.AddressExist(business.BusinessUid, &toAddress)
				/*库中是否存在 from 地址和 from 地址类型*/
				existFromAddress, FromAddressType := syncer.database.Address.AddressExist(business.BusinessUid, &fromAddress)

				/*都不存在，与本项目方无关，跳过*/
				if !existToAddress && !existFromAddress {
					continue
				}
				log.Info("================ found transaction", "txHash", tx.Hash, "from", fromAddress, "to", toAddress, "fromAddressType", FromAddressType, "toAddressType", toAddressType)

				/*组装交易*/
				txItem := &Transaction{
					BusinessId:     business.BusinessUid,
					BlockNumber:    headers[i].Number,
					FromAddress:    tx.From,
					ToAddress:      tx.To,
					Hash:           tx.Hash,
					TokenAddress:   tx.TokenAddress,
					ContractWallet: tx.ContractWallet,
					TxType:         constant.TxTypeUnKnow, // 暂设未知
				}

				/*
				* 充值：from 地址为外部地址，to 地址为用户地址
				* 提现：from 地址为热钱包地址，to 地址为外部地址
				* 归集：from 地址为用户地址，to 地址为热钱包地址（默认热钱包地址为归集地址）
				* 热转冷：from 地址为热钱包地址，to 地址为冷钱包地址
				* 冷转热：from 地址为冷钱包地址，to 地址为热钱包地址
				 */

				/* 1.充值*/
				if !existFromAddress && (existToAddress && toAddressType == constant.AddressTypeUser) {
					log.Info("Found deposit transaction", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = constant.TxTypeDeposit
				}
				/* 2.提现*/
				if (existFromAddress && FromAddressType == constant.AddressTypeHot) && !existToAddress {
					log.Info("Found withdraw transaction", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = constant.TxTypeWithdraw
				}
				/* 3.归集*/
				if (existFromAddress && FromAddressType == constant.AddressTypeUser) && (existToAddress && toAddressType == constant.AddressTypeHot) {
					log.Info("Found collection transaction", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = constant.TxTypeCollection
				}
				/* 4.热转冷*/
				if (existFromAddress && FromAddressType == constant.AddressTypeHot) && (existToAddress && toAddressType == constant.AddressTypeCold) {
					log.Info("Found hot2cold transaction", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = constant.TxTypeHot2Cold
				}
				/* 5.冷转热*/
				if (existFromAddress && FromAddressType == constant.AddressTypeCold) && (existToAddress && toAddressType == constant.AddressTypeHot) {
					log.Info("Found cold2hot transaction", "txHash", tx.Hash, "from", fromAddress, "to", toAddress)
					txItem.TxType = constant.TxTypeCold2Hot
				} else {
					/*都不命中不处理*/
					continue
				}

				/*项目方的交易列表*/
				businessTransactions = append(businessTransactions, txItem)
			}
			if len(businessTransactions) > 0 {
				if businessTxsMap[business.BusinessUid] == nil {
					/*项目方不存在 map， 直接放入*/
					businessTxsMap[business.BusinessUid] = &BatchTransactions{
						BlockHeight:  header.Number.Uint64(),
						Transactions: businessTransactions,
					}
				} else {
					/*项目方已存在 map，追加到 map 中的特定项目方的 transactions 中*/
					businessTxsMap[business.BusinessUid].BlockHeight = header.Number.Uint64()
					businessTxsMap[business.BusinessUid].Transactions = append(businessTxsMap[business.BusinessUid].Transactions, businessTransactions...)
				}
			}
		}
	}
	/*将区块存储到表中*/
	if len(blockHeaders) > 0 {
		log.Info("Store block headers success", "totalBlockHeader size", len(blockHeaders))
		if err := syncer.database.Blocks.StoreBlocks(blockHeaders); err != nil {
			return err
		}
	}
	log.Info("business tx channel", "businessTxChannel", businessTxsMap, "map length", len(businessTxsMap))
	if len(businessTxsMap) > 0 {
		/*交易放入管道中，供后续充值、提现、归集、转冷、转热等处理*/
		syncer.businessChannels <- businessTxsMap
	}
	return nil
}
