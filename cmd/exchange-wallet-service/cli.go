package main

import (
	"context"
	"exchange-wallet-service/common/cliapp"
	"exchange-wallet-service/common/opio"
	"exchange-wallet-service/config"
	"exchange-wallet-service/database"
	flags2 "exchange-wallet-service/flags"
	"exchange-wallet-service/rpcclient"
	"exchange-wallet-service/rpcclient/chainsunion"
	"exchange-wallet-service/services"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
				Name:        "migrate",
				Flags:       flags,
				Description: "Run database migrations",
				Action:      runMigrations,
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

/*数据库迁移命令，创建基础表*/
func runMigrations(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("starting migrations")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			log.Error("failed to close database connection", "err", err)
		}
	}(db)
	return db.ExecuteSQLMigration(cfg.Migrations)
}

/*运行 gRPC 服务*/
func runRpc(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("starting rpc service...")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "error", err)
		return nil, err
	}
	grpcServerConfig := &config.WalletBusinessConfig{
		GrpcHostName: cfg.RpcServer.Host,
		GrpcPort:     cfg.RpcServer.Port,
	}
	/*  1.数据库*/
	db, err := database.NewDB(context.Background(), cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect database", "err", err)
		return nil, err
	}
	log.Info("successfully connected to database")
	/* 2. 新建 chains-union-rpc client*/
	log.Info("creating chains-union-rpc client")
	conn, err := grpc.NewClient(cfg.ChainsUnionRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Connect to da retriever fail", "err", err)
		return nil, err
	}
	client := chainsunion.NewChainsUnionServiceClient(conn)
	rpcClient, err := rpcclient.NewChainsUnionRpcClient(context.Background(), client, "Ethereum")
	if err != nil {
		log.Error(" new chains-union-rpc fail", "err", err)
		return nil, err
	}
	log.Info("successfully connected to chains-union-rpc client", "chains-union-rpc client", &rpcClient)

	/*3. grpc 服务启动 */
	return services.NewWalletBusinessService(grpcServerConfig, db, rpcClient)
}
