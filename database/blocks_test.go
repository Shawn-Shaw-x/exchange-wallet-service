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

func TestNewBlocksDB_WithSQLMock(t *testing.T) {
	// 创建 sqlmock 数据库连接和模拟器
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	// 创建 GORM 实例，使用 sqlmock 的连接
	dialector := postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true, // 禁用复杂协议使 sqlmock 更好支持
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	// 构造待插入数据
	block := Blocks{
		Hash:       common.HexToHash("0xabc"),
		ParentHash: common.HexToHash("0xdef"),
		Number:     big.NewInt(100),
		Timestamp:  12345678,
	}

	// 模拟预期的 SQL 执行
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "blocks"`).
		WithArgs(
			block.Hash[:], // common.Hash 是 [32]byte -> bytes
			block.ParentHash[:],
			block.Number.String(), // u256 serializer 会转为字符串
			block.Timestamp,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// 实例化 DB 实现并调用
	blockDB := NewBlocksDB(db)

	if blockDB == nil {
		panic("failed to create block")
	}

}
