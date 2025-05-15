package database

import (
	"errors"
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

/*交易流水表*/
type Transactions struct {
	GUID         uuid.UUID                `gorm:"primaryKey" json:"guid"`
	BlockHash    common.Hash              `gorm:"column:block_hash;serializer:bytes"  db:"block_hash" json:"block_hash"`
	BlockNumber  *big.Int                 `gorm:"serializer:u256;column:block_number" db:"block_number" json:"BlockNumber" form:"block_number"`
	Hash         common.Hash              `gorm:"column:hash;serializer:bytes"  db:"hash" json:"hash"`
	FromAddress  common.Address           `json:"from_address" gorm:"serializer:bytes"`
	ToAddress    common.Address           `json:"to_address" gorm:"serializer:bytes"`
	TokenAddress common.Address           `json:"token_address" gorm:"serializer:bytes"`
	TokenId      string                   `json:"token_id" gorm:"column:token_id"`
	TokenMeta    string                   `json:"token_meta" gorm:"column:token_meta"`
	Fee          *big.Int                 `gorm:"serializer:u256;column:fee" db:"fee" json:"Fee" form:"fee"`
	Amount       *big.Int                 `gorm:"serializer:u256;column:amount" db:"amount" json:"Amount" form:"amount"`
	Status       constant.TxStatus        `gorm:"type:varchar(10);not null" json:"status"`
	TxType       constant.TransactionType `json:"tx_type" gorm:"column:tx_type"`
	Timestamp    uint64
}

type TransactionsView interface {
	QueryFallBackTransactions(requestId string, startBlock, EndBlock *big.Int) ([]*Transactions, error)
	// todo
}

type TransactionsDB interface {
	TransactionsView

	StoreTransactions(string, []*Transactions, uint64) error
	HandleFallBackTransactions(requestId string, startBlock, EndBlock *big.Int) error
	/*todo*/
}

type transactionsDB struct {
	gorm *gorm.DB
}

func NewTransactionsDB(db *gorm.DB) TransactionsDB {
	return &transactionsDB{db}
}

/*存储交易记录*/
func (db *transactionsDB) StoreTransactions(requestId string, transactionsList []*Transactions, transactionsLength uint64) error {
	result := db.gorm.Table("transactions_"+requestId).CreateInBatches(transactionsList, int(transactionsLength))
	return result.Error
}

/*回滚范围内的交易记录*/
func (db *transactionsDB) QueryFallBackTransactions(requestId string, startBlock, EndBlock *big.Int) ([]*Transactions, error) {
	log.Info("Query fallback transactions", "startBlock", startBlock.String(), "EndBlock", EndBlock.String())
	var fallbackTransactions []*Transactions
	result := db.gorm.Table("transactions_"+requestId).Where("block_number >= ? and block_number <= ?", startBlock.Uint64(), EndBlock.Uint64()).Find(&fallbackTransactions)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return fallbackTransactions, nil
}

/*回滚交易流水表*/
func (db *transactionsDB) HandleFallBackTransactions(requestId string, startBlock, EndBlock *big.Int) error {
	for indexBlock := startBlock.Uint64(); indexBlock <= EndBlock.Uint64(); indexBlock++ {
		var transactionSingle = Transactions{}
		result := db.gorm.Table("transactions_"+requestId).Where("block_number=?", indexBlock).Take(&transactionSingle)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			return result.Error
		}
		log.Info("Handle fallBack transactions", "txStatusFallBack", constant.TxStatusFallback)
		transactionSingle.Status = constant.TxStatusFallback
		err := db.gorm.Table("transactions_" + requestId).Save(&transactionSingle).Error
		if err != nil {
			return err
		}
	}
	return nil
}
