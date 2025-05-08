package main

import (
	"context"
	"exchange-wallet-service/common/cliapp"
	"exchange-wallet-service/config"
	flags2 "exchange-wallet-service/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"time"
)

const (
	POLLING_INTERVAL     = 1 * time.Second
	MAX_RPC_MESSAGE_SIZE = 1024 * 1024 * 300
)

func NewCli(GitCommit string, GitData string) *cli.App {
	flags := flags2.Flags
	return &cli.App{
		Version:              params.VersionWithCommit(GitCommit, GitData),
		Description:          "An exchange wallet scanner services with rpc and rest api server",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "version",
				Description: "Show project version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
			{
				Name:        "rpc",
				Flags:       flags,
				Description: "Run rpc services",
				Action:      cliapp.LifecycleCmd(runRpc),
			},
		},
	}
}

/*运行 gRPC 服务*/
func runRpc(ctx *cli.Context, shutdown context.CancelFunc) (cliapp.Lifecycle, error) {
	log.Info("starting rpc service...")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "error", err)
		return nil, err
	}
	grpcServer := &config.WalletBusinessConfig{
		GrpcHostName: cfg.RpcServer.Host,
		GrpcPort:     cfg.RpcServer.Port,
	}
	/*todo  1.数据库*/
	var db database.DB = nil
	/* todo 2.chains-union-rpc 连接客户端*/

	/*todo grpc 服务启动*/
	return nil, nil
}
