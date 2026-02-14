// bloom_filter.go - 布隆过滤器管理（修复版）
package biz

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/redis/go-redis/v9"
)

// BloomFilterManager 管理用户布隆过滤器，使用LRU缓存限制内存占用，并保证并发安全
type BloomFilterManager struct {
	cache *lru.Cache[string, *userFilter] // 用户ID -> 过滤器包装器
	redis *redis.Client
	mu    sync.RWMutex // 仅用于cache初始化，操作cache本身由lru内部并发安全
}

type userFilter struct {
	filter *bloom.BloomFilter
	mu     sync.RWMutex // 保护单个filter的读写
}

// NewBloomFilterManager 创建带LRU缓存的BloomFilterManager
func NewBloomFilterManager(redisClient *redis.Client) (*BloomFilterManager, error) {
	cache, err := lru.New[string, *userFilter](1000) // 最多缓存1000个用户过滤器
	if err != nil {
		return nil, err
	}
	return &BloomFilterManager{
		cache: cache,
		redis: redisClient,
	}, nil
}

// GetOrCreate 获取用户过滤器，若不存在则从Redis加载或新建
func (m *BloomFilterManager) GetOrCreate(ctx context.Context, userID string) *userFilter {
	// 从LRU缓存获取
	if uf, ok := m.cache.Get(userID); ok {
		return uf
	}

	// 双检锁，防止并发创建
	m.mu.Lock()
	defer m.mu.Unlock()
	if uf, ok := m.cache.Get(userID); ok {
		return uf
	}

	// 尝试从Redis加载
	filter, err := m.loadFromRedis(ctx, userID)
	if err != nil || filter == nil {
		// 创建新的布隆过滤器：预计100万条数据，误判率0.01%
		filter = bloom.NewWithEstimates(1_000_000, 0.0001)
	}
	uf := &userFilter{filter: filter}
	m.cache.Add(userID, uf)
	return uf
}

// loadFromRedis 从Redis加载布隆过滤器
func (m *BloomFilterManager) loadFromRedis(ctx context.Context, userID string) (*bloom.BloomFilter, error) {
	key := fmt.Sprintf("bloom:user:%s", userID)
	data, err := m.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) < 16 {
		return nil, nil
	}
	mVal := binary.BigEndian.Uint64(data[0:8])
	kVal := binary.BigEndian.Uint64(data[8:16])
	filter := bloom.New(uint(mVal), uint(kVal))
	filter.BitSet().ReadFrom(bytes.NewReader(data[16:]))
	return filter, nil
}

// Save 保存用户过滤器到Redis
func (m *BloomFilterManager) Save(ctx context.Context, userID string, uf *userFilter) error {
	uf.mu.RLock()
	defer uf.mu.RUnlock()

	var buf bytes.Buffer
	mVal := uf.filter.Cap()
	kVal := uf.filter.K()
	_ = binary.Write(&buf, binary.BigEndian, mVal)
	_ = binary.Write(&buf, binary.BigEndian, kVal)
	_, _ = uf.filter.BitSet().WriteTo(&buf)

	key := fmt.Sprintf("bloom:user:%s", userID)
	return m.redis.Set(ctx, key, buf.Bytes(), 7*24*time.Hour).Err()
}

// Test 测试元素是否存在（只读，不添加）
func (m *BloomFilterManager) Test(ctx context.Context, userID, element string) (bool, error) {
	uf := m.GetOrCreate(ctx, userID)
	uf.mu.RLock()
	defer uf.mu.RUnlock()
	return uf.filter.TestString(element), nil
}

// Add 添加元素（写操作）
func (m *BloomFilterManager) Add(ctx context.Context, userID, element string) error {
	uf := m.GetOrCreate(ctx, userID)
	uf.mu.Lock()
	defer uf.mu.Unlock()
	uf.filter.AddString(element)
	return nil
}

// TestAndAdd 原子测试并添加（兼容旧代码，但建议新代码使用 Test + Add）
func (m *BloomFilterManager) TestAndAdd(ctx context.Context, userID, element string) (bool, error) {
	uf := m.GetOrCreate(ctx, userID)
	uf.mu.Lock()
	defer uf.mu.Unlock()
	exists := uf.filter.TestString(element)
	uf.filter.AddString(element)
	return exists, nil
}

// SaveAsync 异步保存过滤器（使用独立context）
func (m *BloomFilterManager) SaveAsync(userID string, uf *userFilter) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = m.Save(ctx, userID, uf)
	}()
}
