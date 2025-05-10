package services

import (
	"context"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	exchange_wallet_go "exchange-wallet-service/protobuf/exchange-wallet-go"
	"exchange-wallet-service/rpcclient"
	"exchange-wallet-service/rpcclient/chainsunion"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"math/big"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
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

/*调用 chainunion 获取 fee*/
func (w *WalletBusinessService) getFeeInfo(ctx context.Context, address string) (*FeeInfo, error) {
	accountFeeReq := &chainsunion.FeeRequest{
		Chain:   ChainName,
		Network: Network,
		RawTx:   "",
		Address: address,
	}
	feeResponse, err := w.chainUnionClient.ChainsRpcClient.GetFee(ctx, accountFeeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee info: %w", err)
	}
	return ParseFastFee(feeResponse.FastFee)
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

// FeeInfo 结构体用于存储解析后的费用信息
type FeeInfo struct {
	GasPrice       *big.Int // 基础 gas 价格
	GasTipCap      *big.Int // 小费上限
	Multiplier     int64    // 倍数
	MultipliedTip  *big.Int // 小费 * 倍数
	MaxPriorityFee *big.Int // 小费 * 倍数 * 2 (最大上限)
}

// ParseFastFee 解析 FastFee 字符串并计算相关费用
func ParseFastFee(fastFee string) (*FeeInfo, error) {
	// 1. 按 "|" 分割字符串
	parts := strings.Split(fastFee, "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid fast fee format: %s", fastFee)
	}

	// 2. 解析 GasPrice (baseFee)
	gasPrice := new(big.Int)
	if _, ok := gasPrice.SetString(parts[0], 10); !ok {
		return nil, fmt.Errorf("invalid gas price: %s", parts[0])
	}

	// 3. 解析 GasTipCap
	gasTipCap := new(big.Int)
	if _, ok := gasTipCap.SetString(parts[1], 10); !ok {
		return nil, fmt.Errorf("invalid gas tip cap: %s", parts[1])
	}

	// 4. 解析倍数（去掉 "*" 前缀）
	multiplierStr := strings.TrimPrefix(parts[2], "*")
	multiplier, err := strconv.ParseInt(multiplierStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid multiplier: %s", parts[2])
	}

	// 5. 计算 MultipliedTip (小费 * 倍数)
	multipliedTip := new(big.Int).Mul(
		gasTipCap,
		big.NewInt(multiplier),
	)
	// 设置最小小费阈值 (1 Gwei)
	//minTipCap := big.NewInt(int64(Min1Gwei))
	//if multipliedTip.Cmp(minTipCap) < 0 {
	//	multipliedTip = minTipCap
	//}

	// 6. 计算 MaxPriorityFee (baseFee + 小费*倍数*2)
	maxPriorityFee := new(big.Int).Mul(
		multipliedTip,
		big.NewInt(2),
	)
	// 加上 baseFee
	maxPriorityFee.Add(maxPriorityFee, gasPrice)

	return &FeeInfo{
		GasPrice:       gasPrice,
		GasTipCap:      gasTipCap,
		Multiplier:     multiplier,
		MultipliedTip:  multipliedTip,
		MaxPriorityFee: maxPriorityFee,
	}, nil
}
