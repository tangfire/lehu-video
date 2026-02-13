package biz

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

const (
	hotVideoKeyPrefix = "hot:video:reqcnt:" // 分桶key前缀
	hotVideoTopKey    = "hot:video:top100"  // 热门视频Set
)

type HotVideoDetector struct {
	redis      *redis.Client
	log        *log.Helper
	stopCh     chan struct{}
	topN       int
	bucketSize time.Duration // 单个桶大小（1分钟）
	windowSize int           // 滑动窗口桶数量（例如5分钟）
	calcTicker *time.Ticker
}

func NewHotVideoDetector(redis *redis.Client, logger log.Logger) *HotVideoDetector {
	return &HotVideoDetector{
		redis:      redis,
		log:        log.NewHelper(logger),
		stopCh:     make(chan struct{}),
		topN:       100,
		bucketSize: time.Minute,
		windowSize: 5, // 最近5分钟
	}
}

func (d *HotVideoDetector) Start() {
	d.calcTicker = time.NewTicker(d.bucketSize)
	go d.run()
}

func (d *HotVideoDetector) Stop() {
	close(d.stopCh)
	if d.calcTicker != nil {
		d.calcTicker.Stop()
	}
}

func (d *HotVideoDetector) run() {
	d.calcTopN()
	for {
		select {
		case <-d.calcTicker.C:
			d.calcTopN()
		case <-d.stopCh:
			return
		}
	}
}

func (d *HotVideoDetector) getBucketKey(t time.Time) string {
	return hotVideoKeyPrefix + t.Format("200601021504")
}

//////////////////////////////////////////////////////
// 记录请求
//////////////////////////////////////////////////////

func (d *HotVideoDetector) IncrRequestCount(ctx context.Context, videoID int64) {
	now := time.Now()
	key := d.getBucketKey(now)
	member := strconv.FormatInt(videoID, 10)

	err := d.redis.ZIncrBy(ctx, key, 1, member).Err()
	if err != nil {
		d.log.Warnf("增加视频请求计数失败: %v", err)
		return
	}

	// 设置过期时间：窗口大小 + 1分钟缓冲
	expire := time.Duration(d.windowSize+1) * d.bucketSize
	d.redis.Expire(ctx, key, expire)
}

//////////////////////////////////////////////////////
// 计算TopN
//////////////////////////////////////////////////////

func (d *HotVideoDetector) calcTopN() {
	ctx := context.Background()

	now := time.Now()

	// 收集最近 windowSize 个桶
	keys := make([]string, 0, d.windowSize)
	for i := 0; i < d.windowSize; i++ {
		t := now.Add(-time.Duration(i) * d.bucketSize)
		keys = append(keys, d.getBucketKey(t))
	}

	unionKey := hotVideoKeyPrefix + "union_tmp"

	// ZUNIONSTORE 聚合
	zStore := &redis.ZStore{
		Keys: keys,
	}

	err := d.redis.ZUnionStore(ctx, unionKey, zStore).Err()
	if err != nil {
		d.log.Errorf("聚合热门视频失败: %v", err)
		return
	}

	d.redis.Expire(ctx, unionKey, time.Minute)

	// 取TopN
	topMembers, err := d.redis.ZRevRange(ctx, unionKey, 0, int64(d.topN-1)).Result()
	if err != nil {
		d.log.Errorf("获取TopN失败: %v", err)
		return
	}

	if len(topMembers) == 0 {
		return
	}

	// 用Set存储TopN
	pipe := d.redis.TxPipeline()
	pipe.Del(ctx, hotVideoTopKey)
	pipe.SAdd(ctx, hotVideoTopKey, topMembers)
	pipe.Expire(ctx, hotVideoTopKey, time.Minute)
	_, err = pipe.Exec(ctx)
	if err != nil {
		d.log.Errorf("更新热门视频Set失败: %v", err)
	}
}

//////////////////////////////////////////////////////
// 判断是否热门
//////////////////////////////////////////////////////

func (d *HotVideoDetector) IsHotVideo(ctx context.Context, videoID int64) bool {
	member := strconv.FormatInt(videoID, 10)

	ok, err := d.redis.SIsMember(ctx, hotVideoTopKey, member).Result()
	if err != nil {
		return false
	}
	return ok
}

//////////////////////////////////////////////////////
// 获取热门列表
//////////////////////////////////////////////////////

func (d *HotVideoDetector) GetHotVideos(ctx context.Context) ([]int64, error) {
	members, err := d.redis.SMembers(ctx, hotVideoTopKey).Result()
	if err != nil {
		return nil, err
	}

	result := make([]int64, 0, len(members))
	for _, m := range members {
		id, _ := strconv.ParseInt(m, 10, 64)
		result = append(result, id)
	}
	return result, nil
}
