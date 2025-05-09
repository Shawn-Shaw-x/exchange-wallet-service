package services

import (
	"context"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	exchange_wallet_go "exchange-wallet-service/protobuf/exchange-wallet-go"
	"exchange-wallet-service/rpcclient"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"net"
	"runtime/debug"
	"sync/atomic"
)

const MaxRecvMessageSize = 1024 * 1024 * 300

type WalletBusinessService struct {
	WalletBusinessConfig *config.WalletBusinessConfig
	chainUnionClient     *rpcclient.ChainsUnionRpcClient
	db                   *database.DB
	stopped              atomic.Bool
}

/*新建本地 rpc 服务*/
func NewWalletBusinessService(config *config.WalletBusinessConfig, db *database.DB, rpcClient *rpcclient.ChainsUnionRpcClient) (*WalletBusinessService, error) {
	log.Info("new WalletBusinessService success", "config", config, "db", db)
	return &WalletBusinessService{
		WalletBusinessConfig: config,
		chainUnionClient:     rpcClient,
		db:                   db,
	}, nil
}

/*cli.app的的生命周期管理管理，会自动启动 start */
func (w *WalletBusinessService) Start(ctx context.Context) error {
	go func(w *WalletBusinessService) {
		addr := fmt.Sprintf("%s:%d", w.WalletBusinessConfig.GrpcHostName, w.WalletBusinessConfig.GrpcPort)
		log.Info("starting grpc server", "host", addr, "port", w.WalletBusinessConfig.GrpcPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("failed to listen", "err", err)
		}
		gs := grpc.NewServer(
			grpc.MaxRecvMsgSize(MaxRecvMessageSize),
			grpc.ChainUnaryInterceptor(
				WrapPanicInterceptor,
			),
		)
		reflection.Register(gs)
		exchange_wallet_go.RegisterWalletBusinessServicesServer(gs, w)
		log.Info("starting chainsunion grpc server", "host", addr)
		if err := gs.Serve(listener); err != nil {
			log.Error("failed to serve", "err", err)
		}
	}(w)
	return nil
}

func (w *WalletBusinessService) Stop(ctx context.Context) error {
	w.stopped.Store(true)
	return nil
}

func (w *WalletBusinessService) Stopped() bool {
	return w.stopped.Load()
}

/*panic拦截器*/
func WrapPanicInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	log.Info("wrapped interceptor", "method", info.FullMethod, "req", req, "info", info)
	/*错误处理,防止出错全部程序崩溃*/
	defer func() {
		if e := recover(); e != nil {
			log.Error("panic error", "msg", e)
			log.Debug("panic", "stack", string(debug.Stack()))
			err = status.Errorf(codes.Internal, "panic: %v", e)
		}
	}()
	resp, err = handler(ctx, req)
	log.Debug("wrapped interceptor", "resp", resp, "err", err)
	return resp, err
}
