package database

import (
	"context"
	"exchange-wallet-service/common/retry"
	"exchange-wallet-service/config"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "exchange-wallet-service/database/utils/serializers" // 自动注册序列器
)

// DB 封装了 GORM 的数据库连接以及后续可能扩展的其他表接口。
type DB struct {
	Gorm         *gorm.DB
	Business     BusinessDB
	Blocks       BlocksDB
	ReorgBlocks  ReorgBlocksDB
	Address      AddressDB
	Balances     BalancesDB
	Deposits     DepositsDB
	Withdraws    WithdrawDB
	Internals    InternalsDB
	Transactions TransactionsDB
	Tokens       TokensDB
}

// Close 关闭底层数据库连接。
func (db *DB) Close() error {
	sql, err := db.Gorm.DB()
	if err != nil {
		return err
	}
	return sql.Close()
}

// ExecuteSQLMigration 遍历 migrationsFolder 目录下的所有 SQL 文件并依次执行，用于初始化或迁移数据库结构。
func (db *DB) ExecuteSQLMigration(migrationsFolder string) error {
	err := filepath.Walk(migrationsFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to walk %s", path))
		}
		if info.IsDir() {
			return nil
		}
		fileContent, readErr := ioutil.ReadFile(path)
		if readErr != nil {
			return errors.Wrap(readErr, fmt.Sprintf("failed to read SQL file %s", path))
		}
		execErr := db.Gorm.Exec(string(fileContent)).Error
		if execErr != nil {
			return errors.Wrap(execErr, fmt.Sprintf("failed to execute SQL file %s", path))
		}
		return nil
	})
	return err
}

// NewDB 根据配置创建一个新的数据库连接，并封装成 DB 结构体。
// 支持使用重试策略处理初始化连接失败的情况。
func NewDB(ctx context.Context, dbConfig config.DBConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Name)
	if dbConfig.Port != 0 {
		dsn += fmt.Sprintf(" port=%d", dbConfig.Port)
	}
	if dbConfig.User != "" {
		dsn += fmt.Sprintf(" user=%s", dbConfig.User)
	}
	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)
	gormConfig := gorm.Config{
		SkipDefaultTransaction: true,
		CreateBatchSize:        3_000,
		Logger:                 newLogger,
	}

	/*重试策略*/
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	gormDbBox, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gormDb, err := gorm.Open(postgres.Open(dsn), &gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		return gormDb, nil
	})

	if err != nil {
		return nil, err
	}

	db := &DB{
		Gorm:         gormDbBox,
		Business:     NewBusinessDB(gormDbBox),
		Blocks:       NewBlocksDB(gormDbBox),
		ReorgBlocks:  NewReorgBlocksDB(gormDbBox),
		Address:      NewAddressDB(gormDbBox),
		Balances:     NewBalancesDB(gormDbBox),
		Deposits:     NewDepositsDB(gormDbBox),
		Withdraws:    NewWithdrawsDB(gormDbBox),
		Internals:    NewInternalsDB(gormDbBox),
		Transactions: NewTransactionsDB(gormDbBox),
		Tokens:       NewTokensDB(gormDbBox),
	}
	return db, nil
}
