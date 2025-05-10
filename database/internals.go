package database

import (
	"errors"
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Internals struct {
	// 基础信息
	GUID      uuid.UUID         `gorm:"primaryKey" json:"guid"`
	Timestamp uint64            `json:"timestamp"`
	Status    constant.TxStatus `json:"status" gorm:"column:status"`

	// 区块信息
	BlockHash   common.Hash              `gorm:"column:block_hash;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int                 `gorm:"serializer:u256;column:block_number" json:"block_number"`
	TxHash      common.Hash              `gorm:"column:hash;serializer:bytes" json:"hash"`
	TxType      constant.TransactionType `json:"tx_type" gorm:"column:tx_type"`

	// 交易基础信息
	FromAddress common.Address `json:"from_address" gorm:"serializer:bytes;column:from_address"`
	ToAddress   common.Address `json:"to_address" gorm:"serializer:bytes;column:to_address"`
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

type InternalsView interface {
	QueryInternalsById(requestId string, guid string) (*Internals, error)
	// todo
}

type InternalsDB interface {
	InternalsView

	StoreInternal(string, *Internals) error
	UpdateInternalById(requestId string, id string, signedTx string, status constant.TxStatus) error

	// todo
}

type internalsDB struct {
	gorm *gorm.DB
}

func NewInternalsDB(db *gorm.DB) InternalsDB {
	return &internalsDB{gorm: db}
}

/*存储内部交易*/
func (db *internalsDB) StoreInternal(requestId string, internals *Internals) error {
	return db.gorm.Table("internals_" + requestId).Create(internals).Error
}

/*查询内部交易*/
func (db *internalsDB) QueryInternalsById(requestId string, guid string) (*Internals, error) {
	var internalsEntity Internals
	result := db.gorm.Table("internals_"+requestId).
		Where("guid = ?", guid).
		Take(&internalsEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &internalsEntity, nil
}

/*更新内部交易（归集、热冷互转）*/
func (db *internalsDB) UpdateInternalById(requestId string, id string, signedTx string, status constant.TxStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table("internals_"+requestId).
		Where("guid = ?", id).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
