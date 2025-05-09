package database

import (
	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"math/big"
)

type Blocks struct {
	Hash       common.Hash `gorm:"primaryKey;serializer:bytes"`
	ParentHash common.Hash `gorm:"serializer:bytes"`
	Number     *big.Int    `gorm:"serializer:u256"`
	Timestamp  uint64
}

type BlocksView interface {
	//LatestBlocks() (*rpcclient.BlockHeader, error)
	//QueryBlocksByNumber(*big.Int) (*rpcclient.BlockHeader, error)
}

type BlocksDB interface {
	BlocksView

	StoreBlocks([]Blocks) error
	//DeleteBlocksByNumber(blockHeader []Blocks) error
}

type blocksDB struct {
	gorm *gorm.DB
}

func (b *blocksDB) StoreBlocks(blocks []Blocks) error {
	//TODO implement me
	panic("implement me")
}

func NewBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{gorm: db}
}
