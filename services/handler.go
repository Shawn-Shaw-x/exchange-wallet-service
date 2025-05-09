package services

import (
	"context"
	exchange_wallet_go "exchange-wallet-service/protobuf/exchange-wallet-go"
)

func (w *WalletBusinessService) BusinessRegister(ctx context.Context, request *exchange_wallet_go.BusinessRegisterRequest) (*exchange_wallet_go.BusinessRegisterResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WalletBusinessService) ExportAddressByPublicKeys(ctx context.Context, request *exchange_wallet_go.ExportAddressRequest) (*exchange_wallet_go.ExportAddressResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WalletBusinessService) BuildUnSignTransaction(ctx context.Context, request *exchange_wallet_go.UnSignTransactionRequest) (*exchange_wallet_go.UnSignTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WalletBusinessService) BuildSignedTransaction(ctx context.Context, request *exchange_wallet_go.SignedTransactionRequest) (*exchange_wallet_go.SignedTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WalletBusinessService) SetTokenAddress(ctx context.Context, request *exchange_wallet_go.SetTokenAddressRequest) (*exchange_wallet_go.SetTokenAddressResponse, error) {
	//TODO implement me
	panic("implement me")
}
