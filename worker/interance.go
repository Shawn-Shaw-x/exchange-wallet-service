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

type WorkerInterance struct {
	BaseSynchronizer *BaseSynchronizer

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

/*新建所有定时任务*/
func NewAllWorker(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*WorkerInterance, error) {
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

	/* 0. 处理同步器 worker*/
	synchronizer, err := NewSynchronizer(cfg, db, rpcClient, shutdown)
	if err != nil {
		log.Error("failed to create synchronizer", "err", err)
		return nil, err
	}
	/* todo 1. 充值处理任务*/
	/* todo 2. 提现处理任务*/
	/* todo 3. 内部交易处理任务*/
	/*todo 4. 回滚任务*/

	out := &WorkerInterance{
		BaseSynchronizer: synchronizer,
		shutdown:         shutdown,
	}
	return out, nil
}

func (w *WorkerInterance) Start(ctx context.Context) error {
	/* 1. 启动同步器*/
	err := w.BaseSynchronizer.Start()
	if err != nil {
		log.Error("failed to start base-synchronizer", "err", err)
		return err
	}
	/*todo 2. 启动充值任务*/
	/*todo 3. 启动提现任务*/
	/*todo 4. 启动内部交易任务*/
	/*todo 6. 启动回滚任务*/
	/*todo 7. 启动通知任务*/
	return nil
}

func (w *WorkerInterance) Stop(ctx context.Context) error {
	/*todo 1. 停止同步器*/
	err := w.BaseSynchronizer.Stop()
	if err != nil {
		log.Error("failed to stop base-synchronizer", "err", err)
		return err
	}
	/*todo 2. 停止充值任务*/
	/*todo 3. 停止提现任务*/
	/*todo 4. 停止内部交易任务*/
	/*todo 6. 停止回滚任务*/
	/*todo 7. 停止通知任务*/
	return nil
}

func (w *WorkerInterance) Stopped() bool {
	return w.stopped.Load()
}
