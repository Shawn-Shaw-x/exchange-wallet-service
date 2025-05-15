package dynamic

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

type createTableDB struct {
	gorm *gorm.DB
}

type CreateTableDB interface {
	CreateTableFromTemplate(tableName string)
}

func NewCreateTableDB(db *gorm.DB) CreateTableDB {
	return &createTableDB{
		gorm: db,
	}
}

/*批量创建表*/
func (c *createTableDB) CreateTableFromTemplate(requestId string) {
	err := c.gorm.Transaction(func(tx *gorm.DB) error {
		c.createTable(tx, "addresses", fmt.Sprintf("addresses_%s", requestId))
		c.createTable(tx, "balances", fmt.Sprintf("balances_%s", requestId))
		c.createTable(tx, "transactions", fmt.Sprintf("transactions_%s", requestId))
		c.createTable(tx, "deposits", fmt.Sprintf("deposits_%s", requestId))
		c.createTable(tx, "withdraws", fmt.Sprintf("withdraws_%s", requestId))
		c.createTable(tx, "internals", fmt.Sprintf("internals_%s", requestId))
		c.createTable(tx, "tokens", fmt.Sprintf("tokens_%s", requestId))
		return nil
	})
	if err != nil {
		log.Error("failed to create dynamic table", "requestId", requestId, "err", err)
	}
}

/*动态创建表*/
func (c *createTableDB) createTable(tx *gorm.DB, baseTable, realTableName string) error {
	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (LIKE %s INCLUDING ALL)`, realTableName, baseTable)
	err := tx.Exec(sql).Error
	if err != nil {
		log.Error("create table from base table fail")
		return nil
	}
	return nil
}
