package database

import (
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Withdraws struct {
	// 基础信息
	GUID      uuid.UUID         `gorm:"primaryKey" json:"guid"`
	Timestamp uint64            `json:"timestamp"`
	Status    constant.TxStatus `json:"status" gorm:"column:status"`

	// 区块信息
	BlockHash   common.Hash              `gorm:"column:block_hash;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int                 `gorm:"serializer:u256;column:block_number" json:"block_number"`
	TxHash      common.Hash              `gorm:"column:hash;serializer:bytes" json:"hash"`
	TxType      constant.TransactionType `gorm:"column:tx_type" json:"tx_type"`

	// 交易基础信息
	FromAddress common.Address `gorm:"serializer:bytes;column:from_address" json:"from_address"`
	ToAddress   common.Address `gorm:"serializer:bytes;column:to_address" json:"to_address"`
	Amount      *big.Int       `gorm:"serializer:u256;column:amount" json:"amount"`

	// Gas 费用
	GasLimit             uint64 `json:"gas_limit"`
	MaxFeePerGas         string `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas"`

	// Token 相关信息
	TokenType    constant.TokenType `json:"token_type" gorm:"column:token_type"` // ETH, ERC20, ERC721, ERC1155
	TokenAddress common.Address     `json:"token_address" gorm:"serializer:bytes;column:token_address"`
	TokenId      string             `json:"token_id" gorm:"column:token_id"`     // ERC721/ERC1155 的 token ID
	TokenMeta    string             `json:"token_meta" gorm:"column:token_meta"` // Token 元数据

	// 交易签名
	TxSignHex string `json:"tx_sign_hex" gorm:"column:tx_sign_hex"`
}

type WithdrawsView interface {
	// todo
}

type WithdrawDB interface {
	WithdrawsView

	// todo
}

type withdrawsDB struct {
	gorm *gorm.DB
}

func NewWithdrawsDB(db *gorm.DB) WithdrawDB {
	return &withdrawsDB{gorm: db}
}
