package worker

import (
	"context"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	"exchange-wallet-service/rpcclient"
	"exchange-wallet-service/rpcclient/chainsunion"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync/atomic"
)

/*所有定时任务入口*/
type WorkerEntry struct {
	BaseSynchronizer *BaseSynchronizer

	Finder *Finder

	Withdraw *Withdraw

	Internal *Internal

	Fallback *Fallback

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

/*新建所有定时任务*/
func NewAllWorker(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*WorkerEntry, error) {
	db, err := database.NewDB(ctx, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to master database", "err", err)
		return nil, err
	}
	conn, err := grpc.NewClient(cfg.ChainsUnionRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to connect to chains interance", "err", err)
		return nil, err
	}
	client := chainsunion.NewChainsUnionServiceClient(conn)
	rpcClient, err := rpcclient.NewChainsUnionRpcClient(context.Background(), client, "Ethereum")
	if err != nil {
		log.Error("failed to connect to chains interance", "err", err)
		return nil, err
	}

	/* 1. 新建区块同步器（生成者）*/
	synchronizer, err := NewSynchronizer(cfg, db, rpcClient, shutdown)
	if err != nil {
		log.Error("failed to create synchronizer", "err", err)
		return nil, err
	}
	/*2. 新建交易发现器（消费者）*/
	finder, err := NewFinder(synchronizer, *cfg, shutdown)
	if err != nil {
		log.Error("failed to create finder", "err", err)
		return nil, err
	}
	/*  3. 提现处理任务*/
	withdraw, err := NewWithdraw(cfg, db, rpcClient, shutdown)
	if err != nil {
		log.Error("failed to create new withdraw", "err", err)
		return nil, err
	}
	/*  4. 内部交易处理任务*/
	internal, err := NewInternal(cfg, db, rpcClient, shutdown)
	if err != nil {
		log.Error("failed to create internal", "err", err)
		return nil, err
	}
	/* 5. 回滚处理任务*/
	fallback, err := NewFallback(cfg, db, rpcClient, synchronizer, shutdown)
	if err != nil {
		log.Error("failed to create fallback", "err", err)
		return nil, err
	}
	/*todo 6. 通知处理任务*/

	out := &WorkerEntry{
		BaseSynchronizer: synchronizer,
		Finder:           finder,
		Withdraw:         withdraw,
		Internal:         internal,
		Fallback:         fallback,
		shutdown:         shutdown,
	}
	return out, nil
}

/*启动所有任务*/
func (w *WorkerEntry) Start(ctx context.Context) error {
	/* 1. 启动同步器*/
	err := w.BaseSynchronizer.Start()
	if err != nil {
		log.Error("failed to start base-synchronizer", "err", err)
		return err
	}
	/* 2. 启动交易发现器*/
	err = w.Finder.Start()
	if err != nil {
		log.Error("failed to start finder", "err", err)
		return err
	}
	/* 3. 启动提现处理任务*/
	err = w.Withdraw.Start()
	if err != nil {
		log.Error("failed to start withdraw", "err", err)
		return err
	}

	/* 4. 启动内部交易处理任务*/
	err = w.Internal.Start()
	if err != nil {
		log.Error("failed to start internal", "err", err)
		return err
	}
	/* 6. 启动回滚处理任务*/
	err = w.Fallback.Start()
	if err != nil {
		log.Error("failed to start fallback", "err", err)
		return err
	}

	/*todo 7. 启动通知处理任务*/
	return nil
}

func (w *WorkerEntry) Stop(ctx context.Context) error {
	/* 1. 停止同步器*/
	err := w.BaseSynchronizer.Stop()
	if err != nil {
		log.Error("failed to stop base-synchronizer", "err", err)
		return err
	}
	/* 2. 停止交易发现器*/
	err = w.Finder.Stop()
	if err != nil {
		log.Error("failed to stop finder", "err", err)
		return err
	}
	/* 3. 停止提现任务*/
	err = w.Withdraw.Stop()
	if err != nil {
		log.Error("failed to stop withdraw", "err", err)
		return err
	}
	/* 4. 停止内部交易任务*/
	err = w.Internal.Stop()
	if err != nil {
		log.Error("failed to stop internal", "err", err)
		return err
	}
	/* 6. 停止回滚任务*/
	err = w.Fallback.Stop()
	if err != nil {
		log.Error("failed to stop fallback", "err", err)
		return err
	}
	/*todo 7. 停止通知任务*/
	return nil
}

func (w *WorkerEntry) Stopped() bool {
	return w.stopped.Load()
}
