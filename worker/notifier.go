package worker

import (
	"context"
	"errors"
	"exchange-wallet-service/common/retry"
	"exchange-wallet-service/common/tasks"
	"exchange-wallet-service/database"
	"exchange-wallet-service/database/constant"
	"exchange-wallet-service/httpclient"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"sync/atomic"
	"time"
)

/*通知任务*/
type Notifier struct {
	db *database.DB
	/*项目方切片*/
	businessIds []string
	/*每个项目方给一个专用 httpClient 去通知 */
	notifier       map[string]*httpclient.NotifyClient
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

/*新建通知器*/
func NewNotifier(db *database.DB, shutdown context.CancelCauseFunc) (*Notifier, error) {
	businessList, err := db.Business.QueryBusinessList()
	if err != nil {
		log.Error("query business list error", err)
		return nil, err
	}
	var businessIds []string
	notifierClients := make(map[string]*httpclient.NotifyClient)

	/*创建多客户端*/
	for _, business := range businessList {
		log.Info("handle business id in creating notifier client for each business id", "id", business.BusinessUid)
		businessIds = append(businessIds, business.BusinessUid)
		client, err := httpclient.NewNotifyClient(business.NotifyUrl)
		if err != nil {
			log.Error("create notifier client error", err)
			return nil, err
		}
		notifierClients[business.BusinessUid] = client
	}

	resCtx, resCancel := context.WithCancel(context.Background())

	return &Notifier{
		db:             db,
		notifier:       notifierClients,
		businessIds:    businessIds,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in internals: %w", err))
		}},
		ticker: time.NewTicker(5 * time.Second),
	}, nil
}

/*启动通知任务*/
func (nf *Notifier) Start() error {
	log.Info("start notifier worker...")
	nf.tasks.Go(func() error {
		for {
			select {
			case <-nf.ticker.C:
				var txn []Transaction
				/*每个项目方去查询相应业务表，发出通知*/
				for _, businessId := range nf.businessIds {
					log.Info("start notifier business", "business", businessId, "txn", txn)

					/*查出应通知的充值交易*/
					needNotifyDeposits, err := nf.db.Deposits.QueryNotifyDeposits(businessId)
					if err != nil {
						log.Error("Query notify deposits fail", "err", err)
						return err
					}
					/*查出应通知的提现*/
					needNotifyWithdraws, err := nf.db.Withdraws.QueryNotifyWithdraws(businessId)
					if err != nil {
						log.Error("Query notify withdraw fail", "err", err)
						return err
					}
					/*查出应通知的内部交易*/
					needNotifyInternals, err := nf.db.Internals.QueryNotifyInternal(businessId)
					if err != nil {
						log.Error("Query notify internal fail", "err", err)
						return err
					}
					// BeforeRequest: 更新通知前状态
					err = nf.BeforeAfterNotify(businessId, true, false, needNotifyDeposits, needNotifyWithdraws, needNotifyInternals)
					if err != nil {
						log.Error("Before notify update status  fail", "err", err)
						return err
					}
					/*构建通知请求体*/
					notifyRequest, err := nf.BuildNotifyTransaction(needNotifyDeposits, needNotifyWithdraws, needNotifyInternals)

					/*发送通知*/
					notify, err := nf.notifier[businessId].BusinessNotify(notifyRequest)
					if err != nil {
						log.Error("notify business platform fail", "err")
						return err
					}

					// AfterRequest：更新通知后状态
					err = nf.BeforeAfterNotify(businessId, false, notify, needNotifyDeposits, needNotifyWithdraws, needNotifyInternals)
					if err != nil {
						log.Error("After notify update status fail", "err", err)
						return err
					}
				}
			case <-nf.resourceCtx.Done():
				log.Info("notifier worker shutting down")
				return nil
			}
		}
	})
	return nil
}

/*通知之前：更新通知前状态*/
func (nf *Notifier) BeforeAfterNotify(businessId string, isBefore bool, notifySuccess bool, deposits []*database.Deposits, withdraws []*database.Withdraws, internals []*database.Internals) error {
	var depositsNotifyStatus constant.TxStatus
	var withdrawNotifyStatus constant.TxStatus
	var internalNotifyStatus constant.TxStatus
	if isBefore {
		depositsNotifyStatus = constant.TxStatusNotified
		withdrawNotifyStatus = constant.TxStatusNotified
		internalNotifyStatus = constant.TxStatusNotified
	} else {
		if notifySuccess {
			depositsNotifyStatus = constant.TxStatusSuccess
			withdrawNotifyStatus = constant.TxStatusSuccess
			internalNotifyStatus = constant.TxStatusSuccess
		} else {
			depositsNotifyStatus = constant.TxStatusWalletDone
			withdrawNotifyStatus = constant.TxStatusWalletDone
			internalNotifyStatus = constant.TxStatusWalletDone
		}
	}
	// 过滤状态为 0 的交易
	var updateStutusDepositTxn []*database.Deposits
	for _, deposit := range deposits {
		if deposit.Status != constant.TxStatusCreateUnsigned {
			updateStutusDepositTxn = append(updateStutusDepositTxn, deposit)
		}
	}
	/*更新通知前状态（待通知）*/
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](nf.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
		if err := nf.db.Transaction(func(tx *database.DB) error {
			if len(deposits) > 0 {
				if err := tx.Deposits.UpdateDepositsStatusByTxHash(businessId, depositsNotifyStatus, updateStutusDepositTxn); err != nil {
					return err
				}
			}
			if len(withdraws) > 0 {
				if err := tx.Withdraws.UpdateWithdrawStatusByTxHash(businessId, withdrawNotifyStatus, withdraws); err != nil {
					return err
				}
			}

			if len(internals) > 0 {
				if err := tx.Internals.UpdateInternalStatusByTxHash(businessId, internalNotifyStatus, internals); err != nil {
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
	return nil
}

/*构建充值、提现、内部交易的通知请求*/
func (nf *Notifier) BuildNotifyTransaction(deposits []*database.Deposits, withdraws []*database.Withdraws, internals []*database.Internals) (*httpclient.NotifyRequest, error) {
	var notifyTransactions []*httpclient.Transaction
	for _, deposit := range deposits {
		txItem := &httpclient.Transaction{
			BlockHash:    deposit.BlockHash.String(),
			BlockNumber:  deposit.BlockNumber.Uint64(),
			Hash:         deposit.TxHash.String(),
			FromAddress:  deposit.FromAddress.String(),
			ToAddress:    deposit.ToAddress.String(),
			Value:        deposit.Amount.String(),
			Fee:          deposit.MaxFeePerGas,
			TxType:       deposit.TxType,
			Confirms:     deposit.Confirms,
			TokenAddress: deposit.TokenAddress.String(),
			TokenId:      deposit.TokenId,
			TokenMeta:    deposit.TokenMeta,
		}
		notifyTransactions = append(notifyTransactions, txItem)
	}

	for _, withdraw := range withdraws {
		txItem := &httpclient.Transaction{
			BlockHash:    withdraw.BlockHash.String(),
			BlockNumber:  withdraw.BlockNumber.Uint64(),
			Hash:         withdraw.TxHash.String(),
			FromAddress:  withdraw.FromAddress.String(),
			ToAddress:    withdraw.ToAddress.String(),
			Value:        withdraw.Amount.String(),
			Fee:          withdraw.MaxFeePerGas,
			TxType:       withdraw.TxType,
			Confirms:     0,
			TokenAddress: withdraw.TokenAddress.String(),
			TokenId:      withdraw.TokenId,
			TokenMeta:    withdraw.TokenMeta,
		}
		notifyTransactions = append(notifyTransactions, txItem)
	}

	for _, internal := range internals {
		txItem := &httpclient.Transaction{
			BlockHash:    internal.BlockHash.String(),
			BlockNumber:  internal.BlockNumber.Uint64(),
			Hash:         internal.TxHash.String(),
			FromAddress:  internal.FromAddress.String(),
			ToAddress:    internal.ToAddress.String(),
			Value:        internal.Amount.String(),
			Fee:          internal.MaxFeePerGas,
			TxType:       internal.TxType,
			Confirms:     0,
			TokenAddress: internal.TokenAddress.String(),
			TokenId:      internal.TokenId,
			TokenMeta:    internal.TokenMeta,
		}
		notifyTransactions = append(notifyTransactions, txItem)
	}
	notifyReq := &httpclient.NotifyRequest{
		Txn: notifyTransactions,
	}
	return notifyReq, nil
}

func (nf *Notifier) Stop() error {
	var result error
	nf.resourceCancel()
	nf.ticker.Stop()
	if err := nf.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await notify %w", err))
		return result
	}
	log.Info("stop notify success")
	return nil
}
