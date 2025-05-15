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
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"time"
)

/*
内部交易：
归集、
热转冷、
冷转热
*/
type Internal struct {
	rpcClient      *rpcclient.ChainsUnionRpcClient
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

/*新建内部交易处理器*/
func NewInternal(cfg *config.Config, db *database.DB, rpcClient *rpcclient.ChainsUnionRpcClient, shutdown context.CancelCauseFunc) (*Internal, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Internal{
		rpcClient:      rpcClient,
		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("internal crit error: %w", err))
		}},
		ticker: time.NewTicker(cfg.ChainNode.WorkerInterval),
	}, nil
}

/*
启动内部交易处理任务
处理归集、热转冷、冷转热
交易的发送到链上，更新库、余额
*/
func (in *Internal) Start() error {
	log.Info("starting internal worker.......")
	in.tasks.Go(func() error {
		for {
			select {
			case <-in.ticker.C:
				log.Info("starting internal worker...")
				businessList, err := in.db.Business.QueryBusinessList()
				if err != nil {
					log.Error("failed to query business list", "err", err)
					continue
				}
				for _, business := range businessList {
					/*分项目方处理*/
					unSendTransactionList, err := in.db.Internals.UnSendInternalsList(business.BusinessUid)
					if err != nil {
						log.Error("failed to query unsend internals list", "err", err)
						continue
					}
					if unSendTransactionList == nil || len(unSendTransactionList) <= 0 {
						log.Error("failed to query unsend internals list", "err", err)
						continue
					}

					var balanceList []*database.Balances

					for _, unSendTransaction := range unSendTransactionList {
						/*分单笔交易发送*/
						txHash, err := in.rpcClient.SendTx(unSendTransaction.TxSignHex)
						if err != nil {
							log.Error("failed to send internal transaction", "err", err)
							continue
						} else {
							/*发送成功, 处理from 地址余额*/
							balanceItem := &database.Balances{
								TokenAddress: unSendTransaction.TokenAddress,
								Address:      unSendTransaction.FromAddress,
								LockBalance:  unSendTransaction.Amount,
							}
							/*todo 缺少 to 地址的余额处理？*/

							balanceList = append(balanceList, balanceItem)

							unSendTransaction.TxHash = common.HexToHash(txHash)
							unSendTransaction.Status = constant.TxStatusBroadcasted
						}
					}
					retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
					if _, err := retry.Do[interface{}](in.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
						if err := in.db.Transaction(func(tx *database.DB) error {
							/*处理内部交易余额*/
							if len(balanceList) > 0 {
								log.Info("Update address balance", "totalTx", len(balanceList))
								if err := tx.Balances.UpdateBalanceListByTwoAddress(business.BusinessUid, balanceList); err != nil {
									log.Error("Update address balance fail", "err", err)
									return err
								}

							}
							/*保存内部交易状态*/
							if len(unSendTransactionList) > 0 {
								err = tx.Internals.UpdateInternalListById(business.BusinessUid, unSendTransactionList)
								if err != nil {
									log.Error("update internals status fail", "err", err)
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

			case <-in.resourceCtx.Done():
				log.Info("worker is shutting down")
				return nil
			}
		}
	})
	return nil
}

func (in *Internal) Stop() error {
	var result error
	in.resourceCancel()
	in.ticker.Stop()
	log.Info("internal task stopping...")
	if err := in.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await internal %w", err))
		return result
	}

	log.Info("internal task stopped")
	return nil
}
