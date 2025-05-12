package database

import (
	"exchange-wallet-service/database/constant"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

/*更新余额表用*/
type TokenBalance struct {
	FromAddress  common.Address           `json:"from_address"`
	ToAddress    common.Address           `json:"to_address"`
	TokenAddress common.Address           `json:"to_ken_address"`
	Balance      *big.Int                 `json:"balance"`
	TxType       constant.TransactionType `json:"tx_type"`
}
