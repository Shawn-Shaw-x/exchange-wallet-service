package cache

import (
	"sync"

	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/log"
)

// 定义一个全局的Cache实例
var globalCache *ristretto.Cache[string, *database.Addresses]
var once sync.Once

// InitGlobalCache 初始化全局缓存实例，只会执行一次
func InitGlobalCache() {
	once.Do(func() {
		// 创建一个新的 Ristretto 缓存实例
		cache, err := ristretto.NewCache[string, *database.Addresses](&ristretto.Config[string, *database.Addresses]{
			NumCounters: 1e7,     // number of keys to track frequency of (10M).
			MaxCost:     1 << 30, // maximum cost of cache (1GB).
			BufferItems: 64,      // number of keys per Get buffer.
		})
		if err != nil {
			log.Error("create ristretto cache failed", "err", err)
			return // 如果发生错误，直接返回，globalCache 仍然是 nil
		}
		globalCache = cache
	})
}

// GetGlobalCache 获取全局缓存实例
func GetGlobalCache() *ristretto.Cache[string, *database.Addresses] {
	if globalCache == nil {
		InitGlobalCache()
	}
	return globalCache
}
