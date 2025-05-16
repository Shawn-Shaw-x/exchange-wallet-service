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
	UnSendWithdrawsList(requestId string) ([]*Withdraws, error)
	QueryNotifyWithdraws(requestId string) ([]*Withdraws, error)

	// todo
}

type WithdrawDB interface {
	WithdrawsView

	StoreWithdraw(requestId string, withdraw *Withdraws) error
	UpdateWithdrawById(requestId string, guid string, signedTx string, status constant.TxStatus) error
	UpdateWithdrawStatusByTxHash(requestId string, status constant.TxStatus, withdrawsList []*Withdraws) error
	UpdateWithdrawListById(requestId string, withdrawsList []*Withdraws) error
	HandleFallBackWithdraw(requestId string, startBlock, EndBlock *big.Int) error

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

/*提现状态更新*/
func (db *withdrawsDB) UpdateWithdrawStatusByTxHash(requestId string, status constant.TxStatus, withdrawsList []*Withdraws) error {
	if len(withdrawsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("withdraws_%s", requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var txHashList []string
		for _, withdraw := range withdrawsList {
			txHashList = append(txHashList, withdraw.TxHash.String())
		}

		result := tx.Table(tableName).
			Where("hash IN ?", txHashList).
			Update("status", status)

		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No withdraws updated",
				"requestId", requestId,
				"expectedCount", len(withdrawsList),
			)
		}

		log.Info("Batch update withdraws status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
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

/*查询所有已签名未发送提现*/
func (db *withdrawsDB) UnSendWithdrawsList(requestId string) ([]*Withdraws, error) {
	var withdrawsList []*Withdraws
	err := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", constant.TxStatusSigned).
		Find(&withdrawsList).Error

	if err != nil {
		return nil, fmt.Errorf("query unsend withdraws failed: %w", err)
	}

	return withdrawsList, nil
}

/*提现回滚实现*/
func (db *withdrawsDB) HandleFallBackWithdraw(requestId string, startBlock, EndBlock *big.Int) error {
	for indexBlock := startBlock.Uint64(); indexBlock <= EndBlock.Uint64(); indexBlock++ {
		var withdrawsSingle = Withdraws{}
		result := db.gorm.Table("withdraws_"+requestId).Where("block_number=?", indexBlock).Take(&withdrawsSingle)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			return result.Error
		}
		err := db.gorm.Table("withdraws_"+requestId).
			Where("guid = ?", withdrawsSingle.GUID).
			Update("status", constant.TxStatusFallback).Error
		if err != nil {
			return err
		}
	}
	return nil
}

/*根据 id批量更新提现表状态*/
func (db *withdrawsDB) UpdateWithdrawListById(requestId string, withdrawsList []*Withdraws) error {
	if len(withdrawsList) == 0 {
		return nil
	}

	tableName := fmt.Sprintf("withdraws_%s", requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, withdraw := range withdrawsList {
			// Update each record individually based on TxHash
			result := tx.Table(tableName).
				Where("guid = ?", withdraw.GUID.String()).
				Updates(map[string]interface{}{
					"status": withdraw.Status,
					"amount": withdraw.Amount,
					"hash":   withdraw.TxHash.String(),
					// Add other fields to update as necessary
				})

			// Check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash %s: %w", withdraw.TxHash.Hex(), result.Error)
			}

			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				fmt.Printf("No withdraws updated for TxHash: %s\n", withdraw.TxHash.Hex())
			} else {
				// Log success message with the number of rows affected
				fmt.Printf("Updated withdraw for TxHash: %s, status: %s, amount: %s\n", withdraw.TxHash.Hex(), withdraw.Status, withdraw.Amount.String())
			}
		}

		return nil
	})
}

/*查询提现通知*/
func (db *withdrawsDB) QueryNotifyWithdraws(requestId string) ([]*Withdraws, error) {
	var notifyWithdraws []*Withdraws
	result := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", constant.TxStatusWalletDone).
		Find(&notifyWithdraws)

	if result.Error != nil {
		return nil, fmt.Errorf("query notify withdraws failed: %w", result.Error)
	}

	return notifyWithdraws, nil
}
