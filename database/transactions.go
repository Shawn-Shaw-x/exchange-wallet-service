package database

import (
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

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
	// todo
}

type TransactionsDB interface {
	TransactionsView
	/*todo*/
}

type transactionsDB struct {
	gorm *gorm.DB
}

func NewTransactionsDB(db *gorm.DB) TransactionsDB {
	return &transactionsDB{db}
}
