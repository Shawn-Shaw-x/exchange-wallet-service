package dynamic

import (
	"exchange-wallet-service/database"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateTableFromTemplate(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	requestId := "test123"
	tables := []string{"address", "balances", "transactions", "deposits", "withdraws", "internals", "tokens"}

	for _, table := range tables {
		expectedSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s_%s (LIKE %s INCLUDING ALL)", table, requestId, table)
		mock.ExpectExec(expectedSQL).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	CreateTableFromTemplate(requestId, &database.DB{Gorm: gormDB})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}

}
