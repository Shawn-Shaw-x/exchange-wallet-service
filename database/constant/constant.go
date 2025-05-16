package constant

import (
	"errors"
	"fmt"
	"strings"
)

type TxStatus string

const (
	TxStatusCreateUnsigned TxStatus = "create_unsign"
	TxStatusSigned         TxStatus = "signed"
	TxStatusBroadcasted    TxStatus = "broadcasted"
	/*交易刚发现*/
	TxStatusSuccess    TxStatus = "success"
	TxStatusWalletDone TxStatus = "wallet_done" /*交易已完全确认*/
	TxStatusNotified   TxStatus = "notified"
	TxStatusFallback   TxStatus = "fallback"
)

type TokenType string

const (
	TokenTypeETH     TokenType = "ETH"
	TokenTypeERC20   TokenType = "ERC20"
	TokenTypeERC721  TokenType = "ERC721"
	TokenTypeERC1155 TokenType = "ERC1155"
)

type AddressType string

const (
	AddressTypeUser AddressType = "user"
	AddressTypeHot  AddressType = "hot"
	AddressTypeCold AddressType = "cold"
)

func (at AddressType) String() string {
	return string(at)
}

func ParseAddressType(s string) (AddressType, error) {
	switch strings.ToLower(s) {
	case string(AddressTypeUser):
		return AddressTypeUser, nil
	case string(AddressTypeHot):
		return AddressTypeHot, nil
	case string(AddressTypeCold):
		return AddressTypeCold, nil
	default:
		return "", fmt.Errorf("invalid address type: %s", s)
	}
}

type TransactionType string

const (
	TxTypeUnKnow     TransactionType = "unknow"
	TxTypeDeposit    TransactionType = "deposit"
	TxTypeWithdraw   TransactionType = "withdraw"
	TxTypeCollection TransactionType = "collection"
	TxTypeHot2Cold   TransactionType = "hot2cold"
	TxTypeCold2Hot   TransactionType = "cold2hot"
)

func ParseTransactionType(s string) (TransactionType, error) {
	switch s {
	case string(TxTypeDeposit):
		return TxTypeDeposit, nil
	case string(TxTypeWithdraw):
		return TxTypeWithdraw, nil
	case string(TxTypeCollection):
		return TxTypeCollection, nil
	case string(TxTypeHot2Cold):
		return TxTypeHot2Cold, nil
	case string(TxTypeCold2Hot):
		return TxTypeCold2Hot, nil
	default:
		return TxTypeUnKnow, errors.New("unknown transaction type")
	}
}
