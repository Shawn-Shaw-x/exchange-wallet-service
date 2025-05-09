package dynamic

import (
	"exchange-wallet-service/database"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

/*批量创建表*/
func CreateTableFromTemplate(requestId string, db *database.DB) {
	err := db.Gorm.Transaction(func(tx *gorm.DB) error {
		createTable("addresses", fmt.Sprintf("addresses_%s", requestId), db.Gorm)
		createTable("balances", fmt.Sprintf("balances_%s", requestId), db.Gorm)
		createTable("transactions", fmt.Sprintf("transactions_%s", requestId), db.Gorm)
		createTable("deposits", fmt.Sprintf("deposits_%s", requestId), db.Gorm)
		createTable("withdraws", fmt.Sprintf("withdraws_%s", requestId), db.Gorm)
		createTable("internals", fmt.Sprintf("internals_%s", requestId), db.Gorm)
		createTable("tokens", fmt.Sprintf("tokens_%s", requestId), db.Gorm)
		return nil
	})
	if err != nil {
		log.Error("failed to create dynamic table", "requestId", requestId, "err", err)
	}
}

/*动态创建表*/
func createTable(baseTable, realTableName string, gorm *gorm.DB) {
	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (LIKE %s INCLUDING ALL)`, realTableName, baseTable)
	err := gorm.Exec(sql).Error
	if err != nil {
		log.Error("create table from base table fail")
	}
}
