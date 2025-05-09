package rpcclient

import (
	"context"
	"exchange-wallet-service/rpcclient/chainsunion"
	"github.com/ethereum/go-ethereum/log"
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
