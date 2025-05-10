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
	QueryDepositsById(requestId string, guid string) (*Deposits, error)
	UpdateDepositById(requestId string, guid string, signedTx string, status constant.TxStatus) error

	// todo
}

type depositsDB struct {
	gorm *gorm.DB
}

/*充值存储*/
func (db *depositsDB) StoreDeposits(requestId string, depositList []*Deposits) error {
	if len(depositList) == 0 {
		return nil
	}
	result := db.gorm.Table("deposits_"+requestId).CreateInBatches(depositList, len(depositList))
	if result.Error != nil {
		log.Error("create deposits batch failed", "requestId", requestId, "error", result.Error)
		return result.Error
	}
	return nil
}

/*根据 id 查询充值交易*/
func (db *depositsDB) QueryDepositsById(requestId string, guid string) (*Deposits, error) {
	var deposit Deposits
	result := db.gorm.Table("deposits_"+requestId).
		Where("guid = ?", guid).
		Take(&deposit)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if no record is found
		}
		return nil, result.Error
	}

	return &deposit, nil
}

/*更新充值状态*/
func (db *depositsDB) UpdateDepositById(requestId string, guid string, signedTx string, status constant.TxStatus) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var deposit Deposits
		result := tx.Table("deposits_"+requestId).
			Where("guid = ?", guid).
			Take(&deposit)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("deposit not found for GUID: %s", guid)
			}
			return result.Error
		}

		deposit.Status = status
		deposit.TxSignHex = signedTx

		if err := tx.Table("deposits_" + requestId).Save(&deposit).Error; err != nil {
			return fmt.Errorf("failed to update deposit for GUID: %s, error: %w", guid, err)
		}

		return nil
	})
}

func NewDepositsDB(db *gorm.DB) DepositsDB {
	return &depositsDB{gorm: db}
}
