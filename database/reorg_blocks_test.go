package database

import (
	"math/big"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNewReorgBlocksDB_WithSQLMock(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	block := ReorgBlocks{
		Hash:       common.HexToHash("0xaaa"),
		ParentHash: common.HexToHash("0xbbb"),
		Number:     big.NewInt(101),
		Timestamp:  23456789,
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "reorg_blocks"`).
		WithArgs(
			block.Hash[:],
			block.ParentHash[:],
			block.Number.String(),
			block.Timestamp,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	dbImpl := NewReorgBlocksDB(db)
	//err = dbImpl.StoreReorgBlocks([]ReorgBlocks{block})
	//if err != nil {
	//	t.Errorf("StoreReorgBlocks failed: %v", err)
	//}
	if dbImpl == nil {
		t.Fatalf("failed to create database")
	}
}
