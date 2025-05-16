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
	"strings"
	"time"
)

type Balances struct {
	GUID         uuid.UUID            `gorm:"primary_key" json:"guid"`
	Address      common.Address       `gorm:"type:varchar;not null;serializer:bytes" json:"address"`
	TokenAddress common.Address       `gorm:"type:varchar;not null;serializer:bytes" json:"token_address"`
	AddressType  constant.AddressType `gorm:"type:varchar(10);not null;default:'eoa'" json:"address_type"`
	Balance      *big.Int             `gorm:"type:numeric;not null;default:0;check:balance >= 0;serializer:u256" json:"balance"`
	/*
		锁定余额。例如：
		1. 充值 100 ETH
		2. balance不变，lockBalance = lockBalance + 100；
		3. 确认位到了后，balance = balance + 100；lockBalance = lockBalance - 100；
		4. 对于可用余额：直接就是 balance。对于总余额：balance + lockBalance；
	*/
	LockBalance *big.Int `gorm:"type:numeric;not null;default:0;serializer:u256" json:"lock_balance"`
	Timestamp   uint64   `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
}

type BalancesView interface {
	QueryWalletBalanceByTokenAndAddress(
		requestId string,
		addressType constant.AddressType,
		address,
		tokenAddress common.Address,
	) (*Balances, error)
}

type BalancesDB interface {
	BalancesView

	StoreBalances(string, []*Balances) error
	UpdateOrCreate(string, []*TokenBalance) error
	UpdateBalanceListByTwoAddress(string, []*Balances) error
	UpdateFallBackBalance(string, []*TokenBalance) error
	//todo
}

type balancesDB struct {
	gorm *gorm.DB
}

/*批量余额存库*/
func (db *balancesDB) StoreBalances(requestId string, balances []*Balances) error {
	valueList := make([]*Balances, len(balances))
	for i, balance := range balances {
		if balance != nil {
			balance.Address = common.HexToAddress(balance.Address.Hex())
			balance.TokenAddress = common.HexToAddress(balance.TokenAddress.Hex())
			valueList[i] = balance
		}
	}
	return db.gorm.Table("balances_"+requestId).CreateInBatches(&valueList, len(valueList)).Error
}

/*通过地址和 token 地址查询余额*/
func (db *balancesDB) QueryWalletBalanceByTokenAndAddress(
	requestId string,
	addressType constant.AddressType,
	address,
	tokenAddress common.Address,
) (*Balances, error) {
	balance, err := db.queryBalance(requestId, address, tokenAddress)
	if err == nil {
		return balance, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.createInitialBalance(requestId, addressType, address, tokenAddress)
	}

	return nil, fmt.Errorf("query balance failed: %w", err)
}

/*首次创建余额表*/
func (db *balancesDB) createInitialBalance(
	requestId string,
	addressType constant.AddressType,
	address,
	tokenAddress common.Address,
) (*Balances, error) {
	balance := &Balances{
		GUID:         uuid.New(),
		Address:      address,
		TokenAddress: tokenAddress,
		AddressType:  addressType,
		Balance:      big.NewInt(0),
		LockBalance:  big.NewInt(0),
		Timestamp:    uint64(time.Now().Unix()),
	}

	if err := db.gorm.Table("balances_" + requestId).Create(balance).Error; err != nil {
		log.Error("Failed to create initial balance",
			"requestId", requestId,
			"address", address.String(),
			"tokenAddress", tokenAddress.String(),
			"error", err,
		)
		return nil, fmt.Errorf("create initial balance failed: %w", err)
	}

	log.Debug("Created initial balance",
		"requestId", requestId,
		"address", address.String(),
		"tokenAddress", tokenAddress.String(),
	)

	return balance, nil
}

/*查询余额*/
func (db *balancesDB) queryBalance(
	requestId string,
	address,
	tokenAddress common.Address,
) (*Balances, error) {
	var balance Balances

	err := db.gorm.Table("balances_"+requestId).
		Where("address = ? AND token_address = ?",
			strings.ToLower(address.String()),
			strings.ToLower(tokenAddress.String()),
		).
		Take(&balance).
		Error

	if err != nil {
		return nil, err
	}

	return &balance, nil
}

/*查询余额*/
func (db *balancesDB) queryBalanceByType(
	requestId string,
	addressType constant.AddressType,
	address,
	tokenAddress common.Address,
) (*Balances, error) {
	var balance Balances

	err := db.gorm.Table("balances_"+requestId).
		Where("address = ? AND token_address = ? AND address_type = ?",
			strings.ToLower(address.String()),
			strings.ToLower(tokenAddress.String()),
			strings.ToLower(addressType.String()),
		).
		Take(&balance).
		Error

	if err != nil {
		return nil, err
	}

	return &balance, nil
}

/*有则更新，无则创建*/
func (db *balancesDB) UpdateOrCreate(requestId string, balanceList []*TokenBalance) error {
	if len(balanceList) == 0 {
		return nil
	}
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balanceList {
			log.Info("Processing balance update",
				"txType", balance.TxType,
				"from", balance.FromAddress,
				"to", balance.ToAddress,
				"token", balance.TokenAddress,
				"amount", balance.Balance)

			/*分类更新余额*/
			if err := db.handleBalanceUpdate(tx, requestId, balance); err != nil {
				return fmt.Errorf("failed to handle balance update: %w", err)
			}
		}
		return nil
	})
}

/*根据不同交易类型分别更新余额*/
func (db *balancesDB) handleBalanceUpdate(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	switch balance.TxType {
	case constant.TxTypeDeposit:
		return db.handleDeposit(tx, requestId, balance)
	case constant.TxTypeWithdraw:
		return db.handleWithdraw(tx, requestId, balance)
	case constant.TxTypeCollection:
		return db.handleCollection(tx, requestId, balance)
	case constant.TxTypeHot2Cold:
		return db.handleHotToCold(tx, requestId, balance)
	case constant.TxTypeCold2Hot:
		return db.handleColdToHot(tx, requestId, balance)
	default:
		return fmt.Errorf("unsupported transaction type: %s", balance.TxType)
	}
}

/*存充值余额(用户地址)*/
func (db *balancesDB) handleDeposit(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	/*查 to 地址、用户地址的余额记录*/
	userAddress, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeUser, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user address failed", "err", err)
		return err
	}
	log.Info("Processing handleDeposit",
		"txType", balance.TxType,
		"from", balance.FromAddress,
		"to", balance.ToAddress,
		"token", balance.TokenAddress,
		"amount", balance.Balance,
		"userAddress.Balance,", userAddress.Balance)
	userAddress.Balance = new(big.Int).Add(userAddress.Balance, balance.Balance)
	log.Info("userAddress.Balance after", "Balance after", new(big.Int).Add(userAddress.Balance, balance.Balance))
	return db.UpdateAndSaveBalance(tx, requestId, userAddress)
}

/*冷转热余额更新*/
func (db *balancesDB) handleColdToHot(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeCold, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query cold wallet failed", "err", err)
		return err
	}
	coldWallet.Balance = new(big.Int).Sub(coldWallet.Balance, balance.Balance)
	if err := db.UpdateAndSaveBalance(tx, requestId, coldWallet); err != nil {
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

/*热转冷余额更新*/
func (db *balancesDB) handleHotToCold(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	if err := db.UpdateAndSaveBalance(tx, requestId, hotWallet); err != nil {
		return err
	}

	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeCold, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query cold wallet failed", "err", err)
		return err
	}
	coldWallet.Balance = new(big.Int).Add(coldWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, coldWallet)
}

/*归集余额更新*/
func (db *balancesDB) handleCollection(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	userWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeUser, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user wallet failed", "err", err)
		return err
	}
	userWallet.Balance = new(big.Int).Sub(userWallet.Balance, balance.Balance)
	if err := db.UpdateAndSaveBalance(tx, requestId, userWallet); err != nil {
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

/*更新余额*/
func (db *balancesDB) UpdateAndSaveBalance(tx *gorm.DB, requestId string, balance *Balances) error {
	if balance == nil {
		return fmt.Errorf("balance cannot be nil")
	}

	var currentBalance Balances
	/*查出地址*/
	result := tx.Table("balances_"+requestId).
		Where("address = ? AND token_address = ?",
			strings.ToLower(balance.Address.String()),
			strings.ToLower(balance.TokenAddress.String()),
		).
		Take(&currentBalance)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Debug("Balance record not found",
				"requestId", requestId,
				"address", balance.Address.String(),
				"tokenAddress", balance.TokenAddress.String())
			return nil
		}
		return fmt.Errorf("query balance failed: %w", result.Error)
	}

	currentBalance.Balance = balance.Balance //上游修改这里不做重复计算
	/*
		锁定余额。例如：
		1. 充值 100 ETH
		2. balance不变，lockBalance = lockBalance + 100；
		3. 确认位到了后，balance = balance + 100；lockBalance = lockBalance - 100；
		4. 对于可用余额：直接就是 balance。对于总余额：balance + lockBalance；
	*/
	currentBalance.LockBalance = new(big.Int).Add(currentBalance.LockBalance, balance.LockBalance)
	currentBalance.Timestamp = uint64(time.Now().Unix())

	/*修改*/
	if err := tx.Table("balances_" + requestId).Save(&currentBalance).Error; err != nil {
		log.Error("Failed to save balance",
			"requestId", requestId,
			"address", balance.Address.String(),
			"error", err)
		return fmt.Errorf("save balance failed: %w", err)
	}

	log.Debug("Balance updated and saved successfully",
		"requestId", requestId,
		"address", balance.Address.String(),
		"tokenAddress", balance.TokenAddress.String(),
		"newBalance", currentBalance.Balance.String(),
		"lockBalance", currentBalance.LockBalance.String())

	return nil
}

/*提现交易余额更新*/
func (db *balancesDB) handleWithdraw(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}

	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

/*更新已有的地址余额*/
func (db *balancesDB) UpdateBalanceListByTwoAddress(requestId string, balanceList []*Balances) error {
	if len(balanceList) == 0 {
		return nil
	}

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balanceList {
			var currentBalance Balances
			/*todo bug！！！take 地址是大写的，查不出来，无法插入库*/
			result := tx.Table("balances_"+requestId).
				Where("address = ? AND token_address = ?",
					balance.Address.String(),
					balance.TokenAddress.String()).
				Take(&currentBalance)

			/*todo 这里才是正确的*/
			//result := tx.Table("balances_"+requestId).
			//	Where("address = ? AND token_address = ?",
			//		strings.ToLower(balance.Address.String()),
			//		strings.ToLower(balance.TokenAddress.String())).
			//	Take(&currentBalance)

			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					continue
				}
				return fmt.Errorf("query balance failed: %w", result.Error)
			}
			/*
				提现：
				1. 提现 100 eth
				2. balance = balance - 100； lockBalance = lockBalance + 100；
				3. 发现器发现提现确认后：balance 不变；lockBalance = lockBalance - 100；
				其他交易可类比
			*/
			currentBalance.Balance = new(big.Int).Sub(currentBalance.Balance, balance.LockBalance)
			/*todo 此处应加上而不是直接赋值（可能有多笔交易）*/
			currentBalance.LockBalance = balance.LockBalance
			currentBalance.Timestamp = uint64(time.Now().Unix())

			if err := tx.Table("balances_" + requestId).Save(&currentBalance).Error; err != nil {
				return fmt.Errorf("save balance failed: %w", err)
			}
		}
		return nil
	})
}

func (db *balancesDB) UpdateFallBackBalance(requestId string, balanceList []*TokenBalance) error {
	if len(balanceList) == 0 {
		return nil
	}
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balanceList {
			switch balance.TxType {
			case constant.TxTypeDeposit:
				return db.handleFallBackDeposit(tx, requestId, balance)
			case constant.TxTypeWithdraw:
				return db.handleFallBackWithdraw(tx, requestId, balance)
			case constant.TxTypeCollection:
				return db.handleFallBackCollection(tx, requestId, balance)
			case constant.TxTypeHot2Cold:
				return db.handleFallBackHotToCold(tx, requestId, balance)
			case constant.TxTypeCold2Hot:
				return db.handleFallBackColdToHot(tx, requestId, balance)
			default:
				return fmt.Errorf("unsupported transaction type: %s", balance.TxType)
			}
		}
		return nil
	})
}

/*冷转热余额回滚，冷+，热-*/
func (db *balancesDB) handleFallBackColdToHot(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeCold, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query cold wallet failed", "err", err)
		return err
	}
	coldWallet.Balance = new(big.Int).Add(coldWallet.Balance, balance.Balance)
	if err := db.UpdateAndSaveBalance(tx, requestId, coldWallet); err != nil {
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

/*归集余额回滚，用户余额+，热钱包余额-*/
func (db *balancesDB) handleFallBackCollection(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	userWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeUser, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user wallet failed", "err", err)
		return err
	}
	userWallet.Balance = new(big.Int).Add(userWallet.Balance, balance.Balance)
	if err := db.UpdateAndSaveBalance(tx, requestId, userWallet); err != nil {
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

/*回滚热转冷余额，热余额+ 冷余额-*/
func (db *balancesDB) handleFallBackHotToCold(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	if err := db.UpdateAndSaveBalance(tx, requestId, hotWallet); err != nil {
		return err
	}

	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeCold, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query cold wallet failed", "err", err)
		return err
	}
	coldWallet.Balance = new(big.Int).Sub(coldWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, coldWallet)
}

/*提现余额回滚，热钱包余额增加*/
func (db *balancesDB) handleFallBackWithdraw(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}

	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, requestId, hotWallet)
}

/*
充值回滚（到达确认位的交易，不处理。未到达确认位的，理论上应减去 lockBalance）
todo： change logic
*/
func (db *balancesDB) handleFallBackDeposit(tx *gorm.DB, requestId string, balance *TokenBalance) error {
	userAddress, err := db.QueryWalletBalanceByTokenAndAddress(requestId, constant.AddressTypeUser, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user address failed", "err", err)
		return err
	}
	log.Info("Processing handleDeposit",
		"txType", balance.TxType,
		"from", balance.FromAddress,
		"to", balance.ToAddress,
		"token", balance.TokenAddress,
		"amount", balance.Balance,
		"userAddress.Balance,", userAddress.Balance)
	userAddress.Balance = new(big.Int).Sub(userAddress.Balance, balance.Balance)
	log.Info("userAddress.Balance after", "Balance after", new(big.Int).Sub(userAddress.Balance, balance.Balance))
	return db.UpdateAndSaveBalance(tx, requestId, userAddress)
}

func NewBalancesDB(db *gorm.DB) BalancesDB {
	return &balancesDB{gorm: db}
}
