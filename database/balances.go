package database

import (
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Balances struct {
	GUID         uuid.UUID            `gorm:"primary_key" json:"guid"`
	Address      common.Address       `gorm:"type:varchar;not null;serializer:bytes" json:"address"`
	TokenAddress common.Address       `gorm:"type:varchar;not null;serializer:bytes" json:"token_address"`
	AddressType  constant.AddressType `gorm:"type:varchar(10);not null;default:'eoa'" json:"address_type"`
	Balance      *big.Int             `gorm:"type:numeric;not null;default:0;check:balance >= 0;serializer:u256" json:"balance"`
	LockBalance  *big.Int             `gorm:"type:numeric;not null;default:0;serializer:u256" json:"lock_balance"`
	Timestamp    uint64               `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
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

func (db *balancesDB) QueryWalletBalanceByTokenAndAddress(requestId string, addressType constant.AddressType, address, tokenAddress common.Address) (*Balances, error) {
	//TODO implement me
	panic("implement me")
}

func NewBalancesDB(db *gorm.DB) BalancesDB {
	return &balancesDB{gorm: db}
}
