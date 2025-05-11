package rpcclient

import (
	"errors"
	"exchange-wallet-service/common/bigint"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

var (
	ErrBatchBlockAheadOfProvider = errors.New("the BatchBlock's internal state is ahead of the provider")
	ErrBlockFallBack             = errors.New("the block fallback, fallback handle it now")
)

/*批量扫块工具*/
type BatchBlock struct {
	/*调用外部客户端*/
	rpcClient *ChainsUnionRpcClient

	/*链上最新区块*/
	latestHeader *BlockHeader

	/*最后遍历处理的区块*/
	lastTraversedHeader *BlockHeader

	/*确认位*/
	blockConfirmationDepth *big.Int
}

/*新建区块头批处理工具*/
func NewBatchBlock(rpcClient *ChainsUnionRpcClient, fromHeader *BlockHeader, confDepth *big.Int) *BatchBlock {
	return &BatchBlock{
		rpcClient:              rpcClient,
		lastTraversedHeader:    fromHeader,
		blockConfirmationDepth: confDepth,
	}
}

func (f *BatchBlock) LatestHeader() *BlockHeader {
	return f.latestHeader
}

func (f *BatchBlock) LastTraversedHeader() *BlockHeader {
	return f.lastTraversedHeader
}

/*
批量获取链上区块头
[]BlockHeader 获取到区块头切片
*BlockHeader 发现回滚区块
bool	属于链重组
error	出错
*/
func (f *BatchBlock) NextHeaders(maxSize uint64) ([]BlockHeader, *BlockHeader, bool, error) {
	/*1. 获取链上最新区块*/
	latestHeader, err := f.rpcClient.GetBlockHeader(nil)
	if err != nil {
		return nil, nil, false, fmt.Errorf("error getting latest block header: %w", err)
	} else if latestHeader == nil {
		return nil, nil, false, fmt.Errorf("latest header unreported")
	} else {
		f.latestHeader = latestHeader
	}

	/*计算“已确认”的同步终点高度（防止回滚），即 endHeight = latest - confirmationDepth。*/
	endHeight := new(big.Int).Sub(latestHeader.Number, f.blockConfirmationDepth)
	/*如果还没到确认深度，说明链太短 → 无需同步。
	比如：链上为 5， 确认位为 10， 5-10 = -5 小于 0，则不处理
	*/
	if endHeight.Sign() < 0 {
		return nil, nil, false, nil
	}
	if f.lastTraversedHeader != nil {
		/*比较最后遍历的区块和结束区块*/
		cmp := f.lastTraversedHeader.Number.Cmp(endHeight)
		/*到达最后区块无须处理*/
		if cmp == 0 {
			return nil, nil, false, nil
		} else if cmp > 0 {
			/*最后遍历区块比结束区块大，异常退出*/
			return nil, nil, false, ErrBatchBlockAheadOfProvider
		}
	}
	nextHeight := bigint.Zero
	/*下一个区块指针 = 最后遍历的区块的下一个*/
	if f.lastTraversedHeader != nil {
		nextHeight = new(big.Int).Add(f.lastTraversedHeader.Number, bigint.One)
	}
	/*根据最大大小限制 maxSize，限制这次同步的终点高度。*/
	endHeight = bigint.Clamp(nextHeight, endHeight, maxSize)
	/*计算这次最多拉取多少个区块头*/
	count := new(big.Int).Sub(endHeight, nextHeight).Uint64() + 1
	/*拉取到的区块头存放*/
	var headers []BlockHeader
	for i := uint64(0); i < count; i++ {
		/*拉取一个区块头信息*/
		height := new(big.Int).Add(nextHeight, new(big.Int).SetUint64(i))
		blockHeader, err := f.rpcClient.GetBlockHeader(height)
		if err != nil {
			log.Error("get block info fail", "err", err)
			return nil, nil, false, err
		}
		/*组装进 headers 中*/
		headers = append(headers, *blockHeader)
		/*headers 只有一个数据的情况（边界情况）：
		元素的 parentHash != lastTraversedHeader 的 Hash
		则说明发生链重组-->触发 fallback*/
		if len(headers) == 1 && f.lastTraversedHeader != nil && headers[0].ParentHash != f.lastTraversedHeader.Hash {
			log.Warn("lastTraversedHeader and header zero: parentHash and hash", "parentHash", headers[0].ParentHash, "Hash", f.lastTraversedHeader.Hash)
			return nil, blockHeader, true, ErrBlockFallBack
		}
		/*如果发现第 i 个 header 与 i-1 个不连续（parentHash 不匹配），
		也说明链断开或被重组。*/
		if len(headers) > 1 && headers[i-1].Hash != headers[i].ParentHash {
			log.Warn("headers[i-1] nad headers[i] parentHash and hash", "parentHash", headers[i].ParentHash, "Hash", headers[i-1].Hash)
			return nil, blockHeader, true, ErrBlockFallBack
		}
	}

	numHeaders := len(headers)
	/*没有拉取到区块头，直接退出*/
	if numHeaders == 0 {
		return nil, nil, false, nil
	}
	/*处理最后遍历到的区块头*/
	f.lastTraversedHeader = &headers[numHeaders-1]
	/*正常退出，有数据需要处理*/
	return headers, nil, false, nil

}
