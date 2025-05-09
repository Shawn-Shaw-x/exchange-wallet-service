package config

import (
	"exchange-wallet-service/flags"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const (
	defaultConfirmations        = 64
	defaultSynchronizerInterval = 5000
	defaultWorkerInterval       = 500
	defaultBlocksStep           = 500
)

type Config struct {
	Migrations     string
	ChainNode      ChainNodeConfig
	MasterDB       DBConfig
	SlaveDB        DBConfig
	SlaveDbEnable  bool
	ApiCacheEnable bool
	CacheConfig    CacheConfig
	RpcServer      ServerConfig
	MetricsServer  ServerConfig
	ChainsUnionRpc string
}

type ChainNodeConfig struct {
	ChainId              uint64
	ChainName            string
	RpcUrl               string
	StartingHeight       uint
	Confirmations        uint
	SynchronizerInterval time.Duration
	WorkerInterval       time.Duration
	BlocksStep           uint64
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

type CacheConfig struct {
	ListSize         int
	DetailSize       int
	ListExpireTime   time.Duration
	DetailExpireTime time.Duration
}

type ServerConfig struct {
	Host string
	Port int
}

func LoadConfig(cliCtx *cli.Context) (Config, error) {
	var cfg Config
	cfg = NewConfig(cliCtx)

	if cfg.ChainNode.Confirmations == 0 {
		cfg.ChainNode.Confirmations = defaultConfirmations
	}

	if cfg.ChainNode.SynchronizerInterval == 0 {
		cfg.ChainNode.SynchronizerInterval = defaultSynchronizerInterval
	}

	if cfg.ChainNode.WorkerInterval == 0 {
		cfg.ChainNode.WorkerInterval = defaultWorkerInterval
	}

	if cfg.ChainNode.BlocksStep == 0 {
		cfg.ChainNode.BlocksStep = defaultBlocksStep
	}

	log.Info("loaded chain config", "config", cfg.ChainNode)
	return cfg, nil
}

func NewConfig(ctx *cli.Context) Config {
	return Config{
		Migrations:     ctx.String(flags.MigrationsFlag.Name),
		ChainsUnionRpc: ctx.String(flags.ChainsUnionRpcFlag.Name),
		ChainNode: ChainNodeConfig{
			ChainId:              ctx.Uint64(flags.ChainIdFlag.Name),
			ChainName:            ctx.String(flags.ChainNameFlag.Name),
			RpcUrl:               ctx.String(flags.RpcUrlFlag.Name),
			StartingHeight:       ctx.Uint(flags.StartingHeightFlag.Name),
			Confirmations:        ctx.Uint(flags.ConfirmationsFlag.Name),
			SynchronizerInterval: ctx.Duration(flags.SynchronizerIntervalFlag.Name),
			WorkerInterval:       ctx.Duration(flags.WorkerIntervalFlag.Name),
			BlocksStep:           ctx.Uint64(flags.BlocksStepFlag.Name),
		},
		MasterDB: DBConfig{
			Host:     ctx.String(flags.MasterDbHostFlag.Name),
			Port:     ctx.Int(flags.MasterDbPortFlag.Name),
			Name:     ctx.String(flags.MasterDbNameFlag.Name),
			User:     ctx.String(flags.MasterDbUserFlag.Name),
			Password: ctx.String(flags.MasterDbPasswordFlag.Name),
		},
		SlaveDB: DBConfig{
			Host:     ctx.String(flags.SlaveDbHostFlag.Name),
			Port:     ctx.Int(flags.SlaveDbPortFlag.Name),
			Name:     ctx.String(flags.SlaveDbNameFlag.Name),
			User:     ctx.String(flags.SlaveDbUserFlag.Name),
			Password: ctx.String(flags.SlaveDbPasswordFlag.Name),
		},
		SlaveDbEnable:  ctx.Bool(flags.SlaveDbEnableFlag.Name),
		ApiCacheEnable: ctx.Bool(flags.ApiCacheEnableFlag.Name),
		CacheConfig: CacheConfig{
			ListSize:         ctx.Int(flags.ApiCacheListSizeFlag.Name),
			DetailSize:       ctx.Int(flags.ApiCacheDetailSizeFlag.Name),
			ListExpireTime:   ctx.Duration(flags.ApiCacheListExpireTimeFlag.Name),
			DetailExpireTime: ctx.Duration(flags.ApiCacheDetailExpireTimeFlag.Name),
		},
		RpcServer: ServerConfig{
			Host: ctx.String(flags.RpcHostFlag.Name),
			Port: ctx.Int(flags.RpcPortFlag.Name),
		},
		MetricsServer: ServerConfig{
			Host: ctx.String(flags.MetricsHostFlag.Name),
			Port: ctx.Int(flags.MetricsPortFlag.Name),
		},
	}
}
