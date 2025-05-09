package services

import (
	"context"
	"exchange-wallet-service/database"
	"exchange-wallet-service/database/dynamic"
	exchange_wallet_go "exchange-wallet-service/protobuf/exchange-wallet-go"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"time"
)

/*项目方注册*/
func (w *WalletBusinessService) BusinessRegister(ctx context.Context, request *exchange_wallet_go.BusinessRegisterRequest) (*exchange_wallet_go.BusinessRegisterResponse, error) {
	if request.RequestId == "" || request.NotifyUrl == "" {
		return &exchange_wallet_go.BusinessRegisterResponse{
			Code: exchange_wallet_go.ReturnCode_SUCCESS,
			Msg:  "invalid requestId or NotifiUrl",
		}, nil
	}
	business := &database.Business{
		GUID:        uuid.New(),
		BusinessUid: request.RequestId,
		NotifyUrl:   request.NotifyUrl,
		Timestamp:   uint64(time.Now().Unix()),
	}
	err := w.db.Business.StoreBusiness(business)
	if err != nil {
		log.Error("failed to store business", "business", business, "err", err)
		return &exchange_wallet_go.BusinessRegisterResponse{
			Code: exchange_wallet_go.ReturnCode_ERROR,
			Msg:  "store business db fail",
		}, nil
	}
	dynamic.CreateTableFromTemplate(request.RequestId, w.db)
	return &exchange_wallet_go.BusinessRegisterResponse{
		Code: exchange_wallet_go.ReturnCode_SUCCESS,
		Msg:  "register business success",
	}, nil
}

/*批量公钥转地址*/
func (w *WalletBusinessService) ExportAddressByPublicKeys(ctx context.Context, request *exchange_wallet_go.ExportAddressRequest) (*exchange_wallet_go.ExportAddressResponse, error) {
	//TODO implement me
	panic("implement me")
}

/*构建未签名交易*/
func (w *WalletBusinessService) BuildUnSignTransaction(ctx context.Context, request *exchange_wallet_go.UnSignTransactionRequest) (*exchange_wallet_go.UnSignTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

/*构建已签名交易*/
func (w *WalletBusinessService) BuildSignedTransaction(ctx context.Context, request *exchange_wallet_go.SignedTransactionRequest) (*exchange_wallet_go.SignedTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

/*设定支持的 token 合约*/
func (w *WalletBusinessService) SetTokenAddress(ctx context.Context, request *exchange_wallet_go.SetTokenAddressRequest) (*exchange_wallet_go.SetTokenAddressResponse, error) {
	//TODO implement me
	panic("implement me")
}
