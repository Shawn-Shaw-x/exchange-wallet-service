package services

import (
	"context"
	"exchange-wallet-service/database"
	"exchange-wallet-service/database/constant"
	"exchange-wallet-service/database/dynamic"
	exchange_wallet_go "exchange-wallet-service/protobuf/exchange-wallet-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
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
	var (
		retAddresses []*exchange_wallet_go.Address
		dbAddresses  []*database.Address
		balances     []*database.Balances
	)

	for _, value := range request.PublicKeys {
		address := w.chainUnionClient.ExportAddressByPublicKey("", value.PublicKey)
		item := &exchange_wallet_go.Address{
			Type:    value.Type,
			Address: address,
		}
		parseAddressType, err := constant.ParseAddressType(value.Type)
		if err != nil {
			log.Error("failed to parse addressType", "addressType", value.Type, "err", err)
			return nil, err
		}

		/*地址表*/
		dbAddress := &database.Address{
			GUID:        uuid.New(),
			Address:     common.HexToAddress(address),
			AddressType: parseAddressType,
			PublicKey:   value.PublicKey,
			Timestamp:   uint64(time.Now().Unix()),
		}
		dbAddresses = append(dbAddresses, dbAddress)

		/*余额表*/
		balanceItem := &database.Balances{
			GUID:         uuid.New(),
			Address:      common.HexToAddress(address),
			TokenAddress: common.Address{},
			AddressType:  parseAddressType,
			Balance:      big.NewInt(0),
			LockBalance:  big.NewInt(0),
			Timestamp:    uint64(time.Now().Unix()),
		}
		balances = append(balances, balanceItem)

		/*返回地址*/
		retAddresses = append(retAddresses, item)
	}

	err := w.db.Gorm.Transaction(func(tx *gorm.DB) error {
		/*地址存库*/
		err := w.db.Address.StoreAddresses(request.RequestId, dbAddresses)
		if err != nil {
			log.Error("failed to store addresses", "addresses", dbAddresses, "err", err)
			return err
		}
		/*余额存库*/
		err = w.db.Balances.StoreBalances(request.RequestId, balances)
		if err != nil {
			log.Error("failed to store balances", "err", err)
			return err
		}
		return nil
	})
	if err != nil {
		return &exchange_wallet_go.ExportAddressResponse{
			Code: exchange_wallet_go.ReturnCode_ERROR,
			Msg:  "store  db fail" + err.Error(),
		}, nil
	}
	return &exchange_wallet_go.ExportAddressResponse{
		Code:      exchange_wallet_go.ReturnCode_SUCCESS,
		Msg:       "generate addresses success",
		Addresses: retAddresses,
	}, nil

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
