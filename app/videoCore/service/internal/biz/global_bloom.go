// biz/global_bloom.go - 修复版
package biz

import (
	"context"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/go-kratos/kratos/v2/log"
	"sync"
)

// GlobalVideoBloomFilter 全局视频ID布隆过滤器，用于防止缓存穿透
type GlobalVideoBloomFilter struct {
	filter    *bloom.BloomFilter
	mu        sync.RWMutex
	videoRepo VideoRepo
	log       *log.Helper
	expectedN uint
	fpRate    float64
}

// NewGlobalVideoBloomFilter 创建全局布隆过滤器
func NewGlobalVideoBloomFilter(videoRepo VideoRepo, logger log.Logger, expectedN uint, fpRate float64) *GlobalVideoBloomFilter {
	return &GlobalVideoBloomFilter{
		videoRepo: videoRepo,
		log:       log.NewHelper(logger),
		expectedN: expectedN,
		fpRate:    fpRate,
		filter:    bloom.NewWithEstimates(expectedN, fpRate),
	}
}

// Init 从数据库加载所有视频ID，初始化过滤器（建议异步调用）
func (g *GlobalVideoBloomFilter) Init(ctx context.Context) error {
	g.log.Info("开始初始化全局视频布隆过滤器")
	newFilter := bloom.NewWithEstimates(g.expectedN, g.fpRate)

	var offset int64 = 0
	limit := 10000
	totalLoaded := 0

	for {
		ids, err := g.videoRepo.GetAllVideoIDs(ctx, offset, limit)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			newFilter.AddString(id)
		}
		totalLoaded += len(ids)
		offset += int64(limit)
	}

	g.mu.Lock()
	g.filter = newFilter
	g.mu.Unlock()

	g.log.Infof("全局视频布隆过滤器初始化完成，已加载 %d 个视频ID", totalLoaded)
	return nil
}

// Rebuild 重建过滤器（可定时调用，增量更新方式）
func (g *GlobalVideoBloomFilter) Rebuild(ctx context.Context) error {
	g.log.Info("开始重建全局视频布隆过滤器")
	// 先创建一个新的过滤器，然后从数据库加载所有ID
	newFilter := bloom.NewWithEstimates(g.expectedN, g.fpRate)

	var offset int64 = 0
	limit := 10000
	totalLoaded := 0

	for {
		ids, err := g.videoRepo.GetAllVideoIDs(ctx, offset, limit)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			newFilter.AddString(id)
		}
		totalLoaded += len(ids)
		offset += int64(limit)
	}

	g.mu.Lock()
	g.filter = newFilter
	g.mu.Unlock()

	g.log.Infof("全局视频布隆过滤器重建完成，已加载 %d 个视频ID", totalLoaded)
	return nil
}

// Exists 检查视频ID是否存在（可能误判，但不会漏判）
func (g *GlobalVideoBloomFilter) Exists(videoID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.filter.TestString(videoID)
}

// Add 添加新视频ID（视频发布后调用）
func (g *GlobalVideoBloomFilter) Add(videoID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.filter.AddString(videoID)
}
