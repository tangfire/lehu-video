package biz

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

const (
	hotVideoRequestCountKey = "hot:video:reqcnt" // ZSet，score 为请求次数
	hotVideoListKey         = "hot:video:top100" // 缓存热门视频ID列表
	hotVideoTTL             = 5 * time.Minute    // 热门视频缓存时长
)

type HotVideoDetector struct {
	redis  *redis.Client
	log    *log.Helper
	stopCh chan struct{}
	topN   int
	window time.Duration // 统计窗口，默认 1分钟
}

func NewHotVideoDetector(redis *redis.Client, logger log.Logger) *HotVideoDetector {
	return &HotVideoDetector{
		redis:  redis,
		log:    log.NewHelper(logger),
		stopCh: make(chan struct{}),
		topN:   100,
		window: 1 * time.Minute,
	}
}

func (d *HotVideoDetector) Start() {
	go d.run()
}

func (d *HotVideoDetector) Stop() {
	close(d.stopCh)
}

func (d *HotVideoDetector) run() {
	ticker := time.NewTicker(d.window)
	defer ticker.Stop()
	d.calcTopN()
	for {
		select {
		case <-ticker.C:
			d.calcTopN()
		case <-d.stopCh:
			return
		}
	}
}

// IncrRequestCount 记录视频的一次请求
func (d *HotVideoDetector) IncrRequestCount(ctx context.Context, videoID int64) {
	key := hotVideoRequestCountKey
	member := strconv.FormatInt(videoID, 10)
	// 使用 ZIncrBy 增加计数，并设置过期时间（自动删除）
	err := d.redis.ZIncrBy(ctx, key, 1, member).Err()
	if err != nil {
		d.log.Warnf("增加视频请求计数失败: %v", err)
	}
	// 设置 ZSet 的过期时间，避免无限增长
	d.redis.Expire(ctx, key, 2*d.window)
}

// calcTopN 计算 Top N 热门视频并存入 Redis
func (d *HotVideoDetector) calcTopN() {
	ctx := context.Background()
	key := hotVideoRequestCountKey
	// 获取分数最高的 N 个成员
	res, err := d.redis.ZRevRangeWithScores(ctx, key, 0, int64(d.topN-1)).Result()
	if err != nil {
		d.log.Errorf("获取热门视频排行失败: %v", err)
		return
	}
	hotIDs := make([]string, 0, len(res))
	for _, z := range res {
		hotIDs = append(hotIDs, z.Member.(string))
	}
	// 缓存热门视频ID列表
	if len(hotIDs) > 0 {
		err = d.redis.Set(ctx, hotVideoListKey, hotIDs, hotVideoTTL).Err()
		if err != nil {
			d.log.Errorf("缓存热门视频列表失败: %v", err)
		}
	}
}

// IsHotVideo 判断视频是否为热门视频
func (d *HotVideoDetector) IsHotVideo(ctx context.Context, videoID int64) bool {
	// 从 Redis 获取热门视频列表
	cmd := d.redis.Get(ctx, hotVideoListKey)
	if cmd.Err() != nil {
		return false
	}
	var hotIDs []string
	err := cmd.ScanSlice(&hotIDs)
	if err != nil {
		return false
	}
	target := strconv.FormatInt(videoID, 10)
	for _, id := range hotIDs {
		if id == target {
			return true
		}
	}
	return false
}
