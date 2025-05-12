package database

import (
	"errors"
	"exchange-wallet-service/rpcclient"
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
	LatestBlocks() (*rpcclient.BlockHeader, error)
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

/*存储区块*/
func (db *blocksDB) StoreBlocks(headers []Blocks) error {
	result := db.gorm.CreateInBatches(&headers, len(headers))
	return result.Error
}

func NewBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{gorm: db}
}

/*最新区块*/
func (db *blocksDB) LatestBlocks() (*rpcclient.BlockHeader, error) {
	var header Blocks
	result := db.gorm.Order("number DESC").Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return (*rpcclient.BlockHeader)(&header), nil
}
