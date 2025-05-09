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
