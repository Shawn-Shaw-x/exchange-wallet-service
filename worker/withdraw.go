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

/*提现定时任务协程*/
type Withdraw struct {
	rpcClient      *rpcclient.ChainsUnionRpcClient
	db             *database.DB
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

/*启动定时任务发送提现记录*/
func (w *Withdraw) Start() error {
	log.Info("starting withdraw....")
	w.tasks.Go(func() error {
		for {
			select {
			case <-w.ticker.C:
				/*定时发送提现交易*/
				businessList, err := w.db.Business.QueryBusinessList()
				if err != nil {
					log.Error("failed to query business list", "err", err)
					continue
				}
				for _, business := range businessList {
					/*每个项目方处理已签名但未发出的交易*/
					unSendTransactionList, err := w.db.Withdraws.UnSendWithdrawsList(business.BusinessUid)
					if err != nil {
						log.Error("failed to unsend transaction", "err", err)
						continue
					}
					if unSendTransactionList == nil || len(unSendTransactionList) == 0 {
						log.Error("no withdraw transaction found", "businessId", business.BusinessUid)
						continue
					}

					var balanceList []*database.Balances

					for _, unSendTransaction := range unSendTransactionList {
						/*每一笔提现交易发出去*/
						txHash, err := w.rpcClient.SendTx(unSendTransaction.TxSignHex)
						if err != nil {
							log.Error("failed to send transaction", "err", err)
							continue
						} else {
							/*成功更新余额*/
							balanceItem := &database.Balances{
								TokenAddress: unSendTransaction.TokenAddress,
								Address:      unSendTransaction.FromAddress,
								/*发出提现，balance-，lockBalance+，*/
								LockBalance: unSendTransaction.Amount,
							}
							balanceList = append(balanceList, balanceItem)
							unSendTransaction.TxHash = common.HexToHash(txHash)
							/*已广播，未确认*/
							unSendTransaction.Status = constant.TxStatusBroadcasted
						}
					}

					retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
					/*数据库重试*/
					if _, err := retry.Do[interface{}](w.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
						/*事务*/
						if err := w.db.Transaction(func(tx *database.DB) error {
							/*更新余额表*/
							if len(balanceList) > 0 {
								log.Info("update withdraw balance transaction", "totalTx", len(balanceList))
								if err := tx.Balances.UpdateBalanceListByTwoAddress(business.BusinessUid, balanceList); err != nil {
									log.Error("failed to update withdraw balance transaction", "err", err)
									return err
								}
							}

							/*更新提现表*/
							if len(unSendTransactionList) > 0 {
								err = tx.Withdraws.UpdateWithdrawListById(business.BusinessUid, unSendTransactionList)
								if err != nil {
									log.Error("update withdraw status fail", "err", err)
									return err
								}
							}
							return nil
						}); err != nil {
							return err, nil
						}
						return nil, nil
					}); err != nil {
						return err
					}
				}
			case <-w.resourceCtx.Done():
				/*提现任务终止*/
				log.Info("stopping withdraw in worker")
				return nil
			}
		}
	})
	return nil
}

/*停止提现任务*/
func (w *Withdraw) Stop() error {
	var result error
	w.resourceCancel()
	w.ticker.Stop()
	log.Info("stop withdraw......")
	if err := w.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await withdraw %w", err))
		return result
	}
	log.Info("stop withdraw success")
	return nil
}

/*新建提现处理任务*/
func NewWithdraw(cfg *config.Config, db *database.DB, rpcClient *rpcclient.ChainsUnionRpcClient, shutdown context.CancelCauseFunc) (*Withdraw, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Withdraw{
		rpcClient:      rpcClient,
		db:             db,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critial error in withdraw: %w", err))
		}},
		ticker: time.NewTicker(cfg.ChainNode.WorkerInterval),
	}, nil
}
