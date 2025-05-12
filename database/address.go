package database

import (
	"errors"
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
)

type Address struct {
	GUID        uuid.UUID            `gorm:"primary_key" json:"guid"`
	Address     common.Address       `gorm:"type:varchar;unique;not null;serializer:bytes" json:"address"`
	AddressType constant.AddressType `gorm:"type:varchar(10);not null;default:'user'" json:"address_type"`
	PublicKey   string               `gorm:"type:varchar;not null" json:"public_key"`
	Timestamp   uint64               `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
}

type AddressesView interface {
	AddressExist(requestId string, address *common.Address) (bool, constant.AddressType)

	//	todo
}

type AddressDB interface {
	AddressesView

	StoreAddresses(string, []*Address) error
}

type addressDB struct {
	gorm *gorm.DB
}

/*存地址*/
func (db *addressDB) StoreAddresses(requestId string, addressList []*Address) error {
	for _, addr := range addressList {
		addr.Address = common.HexToAddress(addr.Address.Hex())
	}

	return db.gorm.Table("addresses_"+requestId).
		CreateInBatches(&addressList, len(addressList)).Error
}

/*是否存在地址*/
func (db *addressDB) AddressExist(requestId string, address *common.Address) (bool, constant.AddressType) {
	var addressEntry Address
	err := db.gorm.Table("addresses_"+requestId).
		Where("address = ?", strings.ToLower(address.String())).
		First(&addressEntry).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, constant.AddressTypeUser
		}
		return false, constant.AddressTypeUser
	}
	return true, addressEntry.AddressType
}

func NewAddressDB(db *gorm.DB) AddressDB {
	return &addressDB{gorm: db}
}
