package database

import (
	"exchange-wallet-service/database/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Address struct {
	GUID        uuid.UUID            `gorm:"primary_key" json:"guid"`
	Address     common.Address       `gorm:"type:varchar;unique;not null;serializer:bytes" json:"address"`
	AddressType constant.AddressType `gorm:"type:varchar(10);not null;default:'user'" json:"address_type"`
	PublicKey   string               `gorm:"type:varchar;not null" json:"public_key"`
	Timestamp   uint64               `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
}

type AddressesView interface {
	//	todo
}

type AddressDB interface {
	AddressesView

	StoreAddresses(string, []*Address) error
}

type addressDB struct {
	gorm *gorm.DB
}

func (db *addressDB) StoreAddresses(s string, addresses []*Address) error {
	//TODO implement me
	panic("implement me")
}

func NewAddressDB(db *gorm.DB) AddressDB {
	return &addressDB{gorm: db}
}
