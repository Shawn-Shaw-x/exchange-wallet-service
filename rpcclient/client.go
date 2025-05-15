package rpcclient

import (
	"context"
	"exchange-wallet-service/rpcclient/chainsunion"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

type ChainsUnionRpcClient struct {
	Ctx             context.Context
	ChainName       string
	ChainsRpcClient chainsunion.ChainsUnionServiceClient
}

/*连接外部chains-union-rpc客户端*/
func NewChainsUnionRpcClient(ctx context.Context, rpc chainsunion.ChainsUnionServiceClient, chainName string) (*ChainsUnionRpcClient, error) {
	log.Info("NewChainsUnionRpcClient", "chainName", chainName)
	return &ChainsUnionRpcClient{Ctx: ctx, ChainsRpcClient: rpc, ChainName: chainName}, nil
}

/*根据公钥获取地址*/
func (c *ChainsUnionRpcClient) ExportAddressByPublicKey(typeOrVersion, publicKey string) string {
	req := &chainsunion.ConvertAddressRequest{
		Chain:     c.ChainName,
		Type:      typeOrVersion,
		PublicKey: publicKey,
	}
	address, err := c.ChainsRpcClient.ConvertAddress(context.Background(), req)
	if err != nil {
		log.Error("convert address failed", "err", err)
		return ""
	}
	if address.Code == chainsunion.ReturnCode_ERROR {
		log.Error("convert address fail", "err", err)
		return ""
	}
	return address.Address
}

/*获取链上最新区块封装*/
func (c *ChainsUnionRpcClient) GetBlockHeader(number *big.Int) (*BlockHeader, error) {
	var height int64
	if number == nil {
		height = 0
	} else {
		height = number.Int64()
	}
	req := &chainsunion.BlockHeaderNumberRequest{
		Chain:   c.ChainName,
		Network: "mainnet",
		Height:  height,
	}
	blockHeader, err := c.ChainsRpcClient.GetBlockHeaderByNumber(c.Ctx, req)
	if err != nil {
		log.Error("get latest block GetBlockHeaderByNumber fail", "err", err)
		return nil, err
	}
	if blockHeader.Code == chainsunion.ReturnCode_ERROR {
		log.Error("get latest block fail", "err", err)
		return nil, err
	}
	blockNumber, _ := new(big.Int).SetString(blockHeader.BlockHeader.Number, 10)
	header := &BlockHeader{
		Hash:       common.HexToHash(blockHeader.BlockHeader.Hash),
		ParentHash: common.HexToHash(blockHeader.BlockHeader.ParentHash),
		Number:     blockNumber,
		Timestamp:  blockHeader.BlockHeader.Time,
	}
	return header, nil
}

/*获取区块交易封装*/
func (c *ChainsUnionRpcClient) GetBlockInfo(blockNumber *big.Int) ([]*chainsunion.BlockInfoTransactionList, error) {
	req := &chainsunion.BlockNumberRequest{
		Chain:  c.ChainName,
		Height: blockNumber.Int64(),
		ViewTx: true,
	}
	blockInfo, err := c.ChainsRpcClient.GetBlockByNumber(c.Ctx, req)
	if err != nil {
		log.Error("get block GetBlockByNumber fail", "err", err)
		return nil, err
	}
	if blockInfo.Code == chainsunion.ReturnCode_ERROR {
		log.Warn("get block info fail", "err", err)
		return nil, err
	}
	return blockInfo.Transactions, nil
}

/*获取单笔交易封装*/
func (c *ChainsUnionRpcClient) GetTransactionByHash(hash string) (*chainsunion.TxMessage, error) {
	req := &chainsunion.TxHashRequest{
		Chain:   c.ChainName,
		Network: "mainnet",
		Hash:    hash,
	}
	txInfo, err := c.ChainsRpcClient.GetTxByHash(c.Ctx, req)
	if err != nil {
		log.Error("get GetTxByHash fail", "err", err)
		return nil, err
	}
	if txInfo.Code == chainsunion.ReturnCode_ERROR {
		log.Warn("get block info fail", "err", err)
		return nil, err
	}
	return txInfo.Tx, nil
}

/*发送交易接口封装*/
func (c *ChainsUnionRpcClient) SendTx(rawTx string) (string, error) {
	log.Info("Send transaction", "rawTx", rawTx, "ChainName", c.ChainName)
	req := &chainsunion.SendTxRequest{
		Chain:   c.ChainName,
		Network: "mainnet",
		RawTx:   rawTx,
	}
	txInfo, err := c.ChainsRpcClient.SendTx(c.Ctx, req)
	if txInfo == nil {
		log.Error("send tx info fail, txInfo is null")
		return "", err
	}
	if txInfo.Code == chainsunion.ReturnCode_ERROR {
		log.Error("send tx info fail", "err", err)
		return "", err
	}
	return txInfo.TxHash, nil
}
