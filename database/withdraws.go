package database

import (
	"errors"
	"exchange-wallet-service/database/constant"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
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
	QueryWithdrawsById(requestId string, guid string) (*Withdraws, error)
	// todo
}

type WithdrawDB interface {
	WithdrawsView

	StoreWithdraw(requestId string, withdraw *Withdraws) error
	UpdateWithdrawById(requestId string, guid string, signedTx string, status constant.TxStatus) error

	// todo
}

type withdrawsDB struct {
	gorm *gorm.DB
}

func NewWithdrawsDB(db *gorm.DB) WithdrawDB {
	return &withdrawsDB{gorm: db}
}

/*存储提现交易*/
func (db *withdrawsDB) StoreWithdraw(requestId string, withdraw *Withdraws) error {
	return db.gorm.Table("withdraws_" + requestId).Create(&withdraw).Error
}

/*查询提现交易*/
func (db *withdrawsDB) QueryWithdrawsById(requestId string, guid string) (*Withdraws, error) {
	var withdrawsEntity Withdraws
	result := db.gorm.Table("withdraws_"+requestId).Where("guid = ?", guid).Take(&withdrawsEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &withdrawsEntity, nil
}

/*更新提现*/
func (db *withdrawsDB) UpdateWithdrawById(requestId string, guid string, signedTx string, status constant.TxStatus) error {
	tableName := fmt.Sprintf("withdraws_%s", requestId)

	/*检查提现是否存在*/
	if err := db.CheckWithdrawExistsById(tableName, guid); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": status,
	}
	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	// 3. 执行更新
	if err := db.gorm.Table(tableName).
		Where("guid = ?", guid).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("update withdraw failed: %w", err)
	}

	// 4. 记录日志
	log.Info("Update withdraw success",
		"requestId", requestId,
		"guid", guid,
		"status", status,
		"updates", updates,
	)
	return nil
}

/*检查提现是否存在*/
func (db *withdrawsDB) CheckWithdrawExistsById(tableName string, id string) error {
	var exist bool
	err := db.gorm.Table(tableName).
		Where("guid = ?", id).
		Select("1").
		Find(&exist).Error

	if err != nil {
		return fmt.Errorf("check withdraw exist failed: %w", err)
	}

	if !exist {
		return fmt.Errorf("withdraw not found: %s", id)
	}

	return nil
}
