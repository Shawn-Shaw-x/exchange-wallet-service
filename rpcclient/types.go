package rpcclient

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type BlockHeader struct {
	Hash       common.Hash
	ParentHash common.Hash
	Number     *big.Int
	Timestamp  uint64
}
