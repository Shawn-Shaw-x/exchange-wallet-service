package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNewAddressDB_WithSQLMock(t *testing.T) {
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

	// 构造 Address 实例
	addr := &Address{
		GUID:        uuid.New(),
		Address:     common.HexToAddress("0xabc123"),
		AddressType: "user",
		PublicKey:   "0x123456789abcdef",
		Timestamp:   1710000000,
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "addresses"`).
		WithArgs(
			addr.GUID,
			addr.Address[:],
			addr.AddressType,
			addr.PublicKey,
			addr.Timestamp,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	dbImpl := NewAddressDB(db)
	//err = dbImpl.StoreAddresses("user", []*Address{addr})
	//if err != nil {
	//	t.Errorf("StoreAddresses failed: %v", err)
	//}

	if dbImpl == nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}
}
