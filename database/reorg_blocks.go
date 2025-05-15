package database

import (
	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"math/big"
)

type ReorgBlocks struct {
	Hash       common.Hash `gorm:"primaryKey;serializer:bytes"`
	ParentHash common.Hash `gorm:"serializer:bytes"`
	Number     *big.Int    `gorm:"serializer:u256"`
	Timestamp  uint64
}

type ReorgBlocksView interface {
	//LatestReorgBlocks() (*rpcclient.BlockHeader, error)
}

type ReorgBlocksDB interface {
	ReorgBlocksView

	StoreReorgBlocks([]ReorgBlocks) error
}

type reorgBlocksDB struct {
	gorm *gorm.DB
}

func (r *reorgBlocksDB) StoreReorgBlocks(headers []ReorgBlocks) error {
	result := r.gorm.CreateInBatches(&headers, len(headers))
	return result.Error
}

func NewReorgBlocksDB(db *gorm.DB) ReorgBlocksDB {
	return &reorgBlocksDB{gorm: db}
}
