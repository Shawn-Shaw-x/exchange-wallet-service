package database

import (
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Deposits struct {
	GUID      uuid.UUID         `gorm:"primaryKey;type:varchar(36)" json:"guid"`
	Timestamp uint64            `gorm:"not null;check:timestamp > 0" json:"timestamp"`
	Status    constant.TxStatus `gorm:"type:varchar(10);not null" json:"status"`
	Confirms  uint8             `gorm:"not null;default:0" json:"confirms"`

	BlockHash   common.Hash              `gorm:"type:varchar;not null;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int                 `gorm:"not null;check:block_number > 0;serializer:u256" json:"block_number"`
	TxHash      common.Hash              `gorm:"column:hash;type:varchar;not null;serializer:bytes" json:"hash"`
	TxType      constant.TransactionType `gorm:"type:varchar;not null" json:"tx_type"`

	FromAddress common.Address `gorm:"type:varchar;not null;serializer:bytes" json:"from_address"`
	ToAddress   common.Address `gorm:"type:varchar;not null;serializer:bytes" json:"to_address"`
	Amount      *big.Int       `gorm:"not null;serializer:u256" json:"amount"`

	GasLimit             uint64 `gorm:"not null" json:"gas_limit"`
	MaxFeePerGas         string `gorm:"type:varchar;not null" json:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `gorm:"type:varchar;not null" json:"max_priority_fee_per_gas"`

	TokenType    constant.TokenType `gorm:"type:varchar;not null" json:"token_type"`
	TokenAddress common.Address     `gorm:"type:varchar;not null;serializer:bytes" json:"token_address"`
	TokenId      string             `gorm:"type:varchar;not null" json:"token_id"`
	TokenMeta    string             `gorm:"type:varchar;not null" json:"token_meta"`

	TxSignHex string `gorm:"type:varchar;not null" json:"tx_sign_hex"`
}

type DepositsView interface {
	// todo
}

type DepositsDB interface {
	DepositsView

	StoreDeposits(string, []*Deposits) error
	// todo
}

type depositsDB struct {
	gorm *gorm.DB
}

func (db *depositsDB) StoreDeposits(s string, deposits []*Deposits) error {
	//TODO implement me
	panic("implement me")
}

func NewDepositsDB(db *gorm.DB) DepositsDB {
	return &depositsDB{gorm: db}
}
