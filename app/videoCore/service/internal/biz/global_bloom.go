// biz/global_bloom.go
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
	// 预期数据量（可根据实际调整）
	expectedN uint
	// 误判率
	fpRate float64
}

// NewGlobalVideoBloomFilter 创建全局布隆过滤器
func NewGlobalVideoBloomFilter(videoRepo VideoRepo, logger log.Logger, expectedN uint, fpRate float64) *GlobalVideoBloomFilter {
	return &GlobalVideoBloomFilter{
		videoRepo: videoRepo,
		log:       log.NewHelper(logger),
		expectedN: expectedN,
		fpRate:    fpRate,
		filter:    bloom.NewWithEstimates(expectedN, fpRate), // 先初始化一个空过滤器
	}
}

// Init 从数据库加载所有视频ID，初始化过滤器（建议异步调用）
func (g *GlobalVideoBloomFilter) Init(ctx context.Context) error {
	g.log.Info("开始初始化全局视频布隆过滤器")
	newFilter := bloom.NewWithEstimates(g.expectedN, g.fpRate)

	var offset int64 = 0
	limit := 10000 // 每批加载1万个ID，防止内存暴涨
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

// Rebuild 重建过滤器（可定时调用，例如每天凌晨）
func (g *GlobalVideoBloomFilter) Rebuild(ctx context.Context) error {
	g.log.Info("开始重建全局视频布隆过滤器")
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
