package services

import (
	"context"
	"encoding/base64"
	"errors"
	"exchange-wallet-service/common/json2"
	"exchange-wallet-service/database"
	"exchange-wallet-service/database/constant"
	"exchange-wallet-service/database/dynamic"
	exchange_wallet_go "exchange-wallet-service/protobuf/exchange-wallet-go"
	"exchange-wallet-service/rpcclient/chainsunion"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
	"strconv"
	"time"
)

const (
	ChainName = "Ethereum"
	Network   = "mainnet"
)

var (
	EthGasLimit   uint64 = 60000
	TokenGasLimit uint64 = 120000
	Min1Gwei      uint64 = 1000000000
	//maxFeePerGas                = "135177480"
	//maxPriorityFeePerGas        = "535177480"
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
	response := &exchange_wallet_go.UnSignTransactionResponse{
		Code:     exchange_wallet_go.ReturnCode_ERROR,
		UnSignTx: "0x00",
	}
	if err := validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request:%w", err)
	}

	transactionType, err := constant.ParseTransactionType(request.TxType)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction type: %w", err)
	}
	amountBig, ok := new(big.Int).SetString(request.Value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", request.Value)
	}
	guid := uuid.New()
	nonce, err := w.getAccountNonce(ctx, request.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get account nonce: %w", err)
	}
	feeInfo, err := w.getFeeInfo(ctx, request.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee info: %w", err)
	}
	gasLimit, contractAddress := w.getGasAndContractInfo(request.ContractAddress)

	var returnTx *chainsunion.UnSignTransactionResponse
	/*开启事务*/
	err = w.db.Gorm.Transaction(func(tx *gorm.DB) error {
		switch transactionType {
		/*似乎用不到，充值交易是扫链触发的，而不是业务方调用*/
		case constant.TxTypeDeposit:
			err := w.StoreDeposits(ctx, request, guid, amountBig, gasLimit, feeInfo, transactionType)
			if err != nil {
				log.Error("failed to store deposit", "guid", guid, "err", err)
				return err
			}
		case constant.TxTypeWithdraw:
			if err := w.storeWithdraw(request, guid, amountBig, gasLimit, feeInfo, transactionType); err != nil {
				log.Error("failed to store withdraw", "guid", guid, "err", err)
				return err
			}
		case constant.TxTypeCollection, constant.TxTypeHot2Cold, constant.TxTypeCold2Hot:
			if err := w.storeInternal(request, guid, amountBig, gasLimit, feeInfo, transactionType); err != nil {
				log.Error("failed to store internal", "guid", guid, "err", err)
				return err
			}
		default:
			log.Error("invalid transaction type", "transactionType", transactionType)
			err := errors.New("invalid transaction type")
			return err
		}

		dynamicFeeTxReq := Eip1559DynamicFeeTx{
			ChainId:              request.ChainId,
			Nonce:                uint64(nonce),
			FromAddress:          request.From,
			ToAddress:            request.To,
			GasLimit:             gasLimit,                        /*gas 总限制*/
			MaxFeePerGas:         feeInfo.MaxPriorityFee.String(), /*每单位最大 gas = baseFee + priorityFee*/
			MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),  /*矿工优先费*/
			Amount:               request.Value,
			ContractAddress:      contractAddress,
		}
		data := json2.ToJSON(dynamicFeeTxReq)
		log.Info("WalletBusinessService CreateUnSignTransaction dynamicFeeTxReq", "dynamicFeeTxReq", json2.ToJSONString(dynamicFeeTxReq))
		base64Str := base64.StdEncoding.EncodeToString(data)
		unsignTx := &chainsunion.UnSignTransactionRequest{
			Chain:    ChainName,
			Network:  Network,
			Base64Tx: base64Str,
		}
		log.Info("WalletBusinessService CreateUnSignTransaction unsignTx", "unsignTx", json2.ToJSONString(unsignTx))
		returnTx, err = w.chainUnionClient.ChainsRpcClient.BuildUnSignTransaction(ctx, unsignTx)
		log.Info("WalletBusinessService CreateUnSignTransaction returnTx", "returnTx", json2.ToJSONString(returnTx))
		if err != nil {
			log.Error("WalletBusinessService CreateUnSignTransaction returnTx", "err", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Error("database transaction execution fail", "err", err)
		response.Msg = "database transaction execution fail" + err.Error()
		return response, err
	}
	response.Code = exchange_wallet_go.ReturnCode_SUCCESS
	response.Msg = "build unsign transaction success"
	response.TransactionId = guid.String()
	response.UnSignTx = returnTx.UnSignTx
	return response, nil
}

/*构建已签名交易*/
func (w *WalletBusinessService) BuildSignedTransaction(ctx context.Context, request *exchange_wallet_go.SignedTransactionRequest) (*exchange_wallet_go.SignedTransactionResponse, error) {
	response := &exchange_wallet_go.SignedTransactionResponse{
		Code: exchange_wallet_go.ReturnCode_ERROR,
	}
	/*1. 从数据库中获取交易类型*/
	var (
		fromAddress          string
		toAddress            string
		amount               string
		tokenAddress         string
		gasLimit             uint64
		maxFeePerGas         string
		maxPriorityFeePerGas string
	)
	transactionType, err := constant.ParseTransactionType(request.TxType)
	if err != nil {
		return nil, fmt.Errorf("invalid request TxType: %w", err)
	}
	switch transactionType {
	case constant.TxTypeDeposit:
		tx, err := w.db.Deposits.QueryDepositsById(request.RequestId, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query deposits by id fail: %w", err)
		}
		if tx == nil {
			response.Msg = "query deposits by id fail"
			return response, nil
		}
		fromAddress = tx.FromAddress.String()
		toAddress = tx.ToAddress.String()
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress.String()
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeePerGas
	case constant.TxTypeWithdraw:
		tx, err := w.db.Withdraws.QueryWithdrawsById(request.RequestId, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query withdraw failed: %w", err)
		}
		if tx == nil {
			response.Msg = "Withdraw transaction not found"
			return response, nil
		}
		fromAddress = tx.FromAddress.String()
		toAddress = tx.ToAddress.String()
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress.String()
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeePerGas
	case constant.TxTypeCollection, constant.TxTypeHot2Cold, constant.TxTypeCold2Hot:
		tx, err := w.db.Internals.QueryInternalsById(request.RequestId, request.TransactionId)
		if err != nil {
			return nil, fmt.Errorf("query internal failed: %w", err)
		}
		if tx == nil {
			response.Msg = "Internal transaction not found"
			return response, nil
		}
		fromAddress = tx.FromAddress.String()
		toAddress = tx.ToAddress.String()
		amount = tx.Amount.String()
		tokenAddress = tx.TokenAddress.String()
		gasLimit = tx.GasLimit
		maxFeePerGas = tx.MaxFeePerGas
		maxPriorityFeePerGas = tx.MaxPriorityFeePerGas
	default:
		response.Msg = "Unsupported transaction type"
		response.SignedTx = "0x00"
		return response, nil
	}

	/*2. 获取当前账户 nonce*/
	nonce, err := w.getAccountNonce(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get account nonce fail: %w", err)
	}

	/*3. 构建 EIP-1159 交易类型*/
	dynamicFeeTx := Eip1559DynamicFeeTx{
		ChainId:              request.ChainId,
		Nonce:                uint64(nonce),
		FromAddress:          fromAddress,
		ToAddress:            toAddress,
		GasLimit:             gasLimit,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		Amount:               amount,
		ContractAddress:      tokenAddress,
	}

	/*4. 构建已签名交易*/
	data := json2.ToJSON(&dynamicFeeTx)
	base64Str := base64.StdEncoding.EncodeToString(data)
	signedTxReq := &chainsunion.SignedTransactionRequest{
		Chain:     ChainName,
		Network:   Network,
		Signature: request.Signature,
		Base64Tx:  base64Str,
	}

	log.Info("BuildSignedTransaction request", "dynamicFeeTx", json2.ToJSONString(dynamicFeeTx))
	returnTx, err := w.chainUnionClient.ChainsRpcClient.BuildSignedTransaction(ctx, signedTxReq)
	log.Info("BuildSignedTransaction request", "returnTx", json2.ToJSONString(returnTx))
	if err != nil {
		return nil, fmt.Errorf("build signed transaction failed: %w", err)
	}

	/*5. 更新数据库状态*/
	var updateErr error
	switch transactionType {
	case constant.TxTypeDeposit:
		updateErr = w.db.Deposits.UpdateDepositById(request.RequestId, request.TransactionId, returnTx.SignedTx, constant.TxStatusSigned)
	case constant.TxTypeWithdraw:
		updateErr = w.db.Withdraws.UpdateWithdrawById(request.RequestId, request.TransactionId, returnTx.SignedTx, constant.TxStatusSigned)
	case constant.TxTypeCollection, constant.TxTypeHot2Cold, constant.TxTypeCold2Hot:
		updateErr = w.db.Internals.UpdateInternalById(request.RequestId, request.TransactionId, returnTx.SignedTx, constant.TxStatusSigned)
	default:
		response.Msg = "Unsupported transaction type"
		response.SignedTx = "0x00"
		return response, nil
	}
	if updateErr != nil {
		return nil, fmt.Errorf("update transaction status failed: %w", updateErr)
	}
	response.SignedTx = returnTx.SignedTx
	response.Msg = "build signed tx success"
	response.Code = exchange_wallet_go.ReturnCode_SUCCESS
	return response, nil
}

/*设定支持的 token 合约*/
func (w *WalletBusinessService) SetTokenAddress(ctx context.Context, request *exchange_wallet_go.SetTokenAddressRequest) (*exchange_wallet_go.SetTokenAddressResponse, error) {
	var (
		tokenList []database.Tokens
	)
	for _, value := range request.TokenList {
		CollectAmountBigInt, _ := new(big.Int).SetString(value.CollectAmount, 10)
		ColdAmountBigInt, _ := new(big.Int).SetString(value.ColdAmount, 10)
		token := database.Tokens{
			GUID:          uuid.New(),
			TokenAddress:  common.HexToAddress(value.Address),
			Decimals:      uint8(value.Decimals),
			TokenName:     value.TokenName,
			CollectAmount: CollectAmountBigInt,
			ColdAmount:    ColdAmountBigInt,
			Timestamp:     uint64(time.Now().Unix()),
		}
		tokenList = append(tokenList, token)
	}

	/*token 合约存储*/
	err := w.db.Tokens.StoreTokens(request.RequestId, tokenList)
	if err != nil {
		log.Error("failed to store tokens", "err", err)
		return nil, err
	}
	return &exchange_wallet_go.SetTokenAddressResponse{
		Code: exchange_wallet_go.ReturnCode_SUCCESS,
		Msg:  "set token address success",
	}, nil

}

/*请求验证*/
func validateRequest(request *exchange_wallet_go.UnSignTransactionRequest) error {
	if request == nil {
		return errors.New("request cannot be nil")
	}
	if request.From == "" {
		return errors.New("from address cannot be empty")
	}
	if request.To == "" {
		return errors.New("to address cannot be empty")
	}
	if request.Value == "" {
		return errors.New("value cannot be empty")
	}
	return nil
}

/*获取账号信息：nonce*/
func (w *WalletBusinessService) getAccountNonce(ctx context.Context, address string) (int, error) {
	accountReq := &chainsunion.AccountRequest{
		Chain:           ChainName,
		Network:         Network,
		Address:         address,
		ContractAddress: "0x00",
	}

	accountInfo, err := w.chainUnionClient.ChainsRpcClient.GetAccount(ctx, accountReq)
	if err != nil {
		return 0, fmt.Errorf("get account info failed: %w", err)
	}

	return strconv.Atoi(accountInfo.Sequence)
}

/*获取默认 gasLimit*/
func (w *WalletBusinessService) getGasAndContractInfo(contractAddress string) (uint64, string) {
	if contractAddress == "0x00" {
		return EthGasLimit, "0x00"
	}
	return TokenGasLimit, contractAddress
}

/*封装存储充值*/
func (w *WalletBusinessService) StoreDeposits(ctx context.Context,
	depositsRequest *exchange_wallet_go.UnSignTransactionRequest, transactionId uuid.UUID, amountBig *big.Int,
	gasLimit uint64, feeInfo *FeeInfo, transactionType constant.TransactionType) error {

	dbDeposit := &database.Deposits{
		GUID:                 transactionId,
		Timestamp:            uint64(time.Now().Unix()),
		Status:               constant.TxStatusCreateUnsigned,
		Confirms:             0,
		BlockHash:            common.Hash{},
		BlockNumber:          big.NewInt(1),
		TxHash:               common.Hash{},
		TxType:               transactionType,
		FromAddress:          common.HexToAddress(depositsRequest.From),
		ToAddress:            common.HexToAddress(depositsRequest.To),
		Amount:               amountBig,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		TokenType:            determineTokenType(depositsRequest.ContractAddress),
		TokenAddress:         common.HexToAddress(depositsRequest.ContractAddress),
		TokenId:              depositsRequest.TokenId,
		TokenMeta:            depositsRequest.TokenMeta,
		TxSignHex:            "",
	}

	return w.db.Deposits.StoreDeposits(depositsRequest.RequestId, []*database.Deposits{dbDeposit})
}

/*确定合约类型*/
func determineTokenType(contractAddress string) constant.TokenType {
	if contractAddress == "0x00" {
		return constant.TokenTypeETH
	}
	// 这里可以添加更多的 token 类型判断逻辑
	return constant.TokenTypeERC20
}

/*存储提现封装*/
func (w *WalletBusinessService) storeWithdraw(request *exchange_wallet_go.UnSignTransactionRequest,
	transactionId uuid.UUID, amountBig *big.Int, gasLimit uint64, feeInfo *FeeInfo, transactionType constant.TransactionType) error {

	withdraw := &database.Withdraws{
		GUID:                 transactionId,
		Timestamp:            uint64(time.Now().Unix()),
		Status:               constant.TxStatusCreateUnsigned,
		BlockHash:            common.Hash{},
		BlockNumber:          big.NewInt(1),
		TxHash:               common.Hash{},
		TxType:               transactionType,
		FromAddress:          common.HexToAddress(request.From),
		ToAddress:            common.HexToAddress(request.To),
		Amount:               amountBig,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		TokenType:            determineTokenType(request.ContractAddress),
		TokenAddress:         common.HexToAddress(request.ContractAddress),
		TokenId:              request.TokenId,
		TokenMeta:            request.TokenMeta,
		TxSignHex:            "",
	}

	return w.db.Withdraws.StoreWithdraw(request.RequestId, withdraw)
}

// 存储内部交易(冷热互转、归集)
func (w *WalletBusinessService) storeInternal(request *exchange_wallet_go.UnSignTransactionRequest,
	transactionId uuid.UUID, amountBig *big.Int, gasLimit uint64, feeInfo *FeeInfo, transactionType constant.TransactionType) error {

	internal := &database.Internals{
		GUID:                 transactionId,
		Timestamp:            uint64(time.Now().Unix()),
		Status:               constant.TxStatusCreateUnsigned,
		BlockHash:            common.Hash{},
		BlockNumber:          big.NewInt(1),
		TxHash:               common.Hash{},
		TxType:               transactionType,
		FromAddress:          common.HexToAddress(request.From),
		ToAddress:            common.HexToAddress(request.To),
		Amount:               amountBig,
		GasLimit:             gasLimit,
		MaxFeePerGas:         feeInfo.MaxPriorityFee.String(),
		MaxPriorityFeePerGas: feeInfo.MultipliedTip.String(),
		TokenType:            determineTokenType(request.ContractAddress),
		TokenAddress:         common.HexToAddress(request.ContractAddress),
		TokenId:              request.TokenId,
		TokenMeta:            request.TokenMeta,
		TxSignHex:            "",
	}

	return w.db.Internals.StoreInternal(request.RequestId, internal)
}
