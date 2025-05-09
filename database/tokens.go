package database

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

type Tokens struct {
	GUID          uuid.UUID      `gorm:"primaryKey" json:"guid"`
	TokenAddress  common.Address `gorm:"serializer:bytes" json:"token_address"`
	Decimals      uint8          `json:"uint"`
	TokenName     string         `json:"tokens_name"`
	CollectAmount *big.Int       `gorm:"serializer:u256" json:"collect_amount"`
	ColdAmount    *big.Int       `gorm:"serializer:u256" json:"cold_amount"`
	Timestamp     uint64         `json:"timestamp"`
}

type TokensView interface {
	/*todo*/
}

type TokensDB interface {
	TokensView

	StoreTokens(string, []Tokens) error
	/*todo*/
}

type tokensDB struct {
	gorm *gorm.DB
}

/*存代币合约地址*/
func (db *tokensDB) StoreTokens(requestId string, tokenList []Tokens) error {
	result := db.gorm.Table("tokens_"+requestId).CreateInBatches(&tokenList, len(tokenList))
	return result.Error
}

func NewTokensDB(db *gorm.DB) TokensDB {
	return &tokensDB{gorm: db}
}
