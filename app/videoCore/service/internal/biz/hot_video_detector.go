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

type incrCmd struct {
	videoID int64
	delta   int64
}

type HotVideoDetector struct {
	redis      *redis.Client
	log        *log.Helper
	stopCh     chan struct{}
	topN       int
	bucketSize time.Duration // 单个桶大小（1分钟）
	windowSize int           // 滑动窗口桶数量（例如5分钟）
	calcTicker *time.Ticker
	// 新增：批量处理器
	batchProc *BatchProcessor[*incrCmd]
}

func NewHotVideoDetector(redis *redis.Client, logger log.Logger) *HotVideoDetector {
	d := &HotVideoDetector{
		redis:      redis,
		log:        log.NewHelper(logger),
		stopCh:     make(chan struct{}),
		topN:       100,
		bucketSize: time.Minute,
		windowSize: 5,
	}
	// 初始化批量处理器：每2秒或积攒500条刷新
	d.batchProc = NewBatchProcessor[*incrCmd](
		500,
		2*time.Second,
		d.batchIncr, // 批量执行函数
		logger,
	)
	return d
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
	if d.batchProc != nil {
		d.batchProc.Stop()
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

// IncrRequestCount 原方法改为聚合
func (d *HotVideoDetector) IncrRequestCount(ctx context.Context, videoID int64) {
	d.batchProc.Add(&incrCmd{
		videoID: videoID,
		delta:   1,
	})
}

// batchIncr 批量执行 ZINCRBY
func (d *HotVideoDetector) batchIncr(cmds []*incrCmd) error {
	if len(cmds) == 0 {
		return nil
	}
	now := time.Now()
	key := d.getBucketKey(now)
	expire := time.Duration(d.windowSize+1) * d.bucketSize

	// 聚合相同 videoID 的 delta
	agg := make(map[int64]int64)
	for _, cmd := range cmds {
		agg[cmd.videoID] += cmd.delta
	}

	// 使用 Pipeline 批量执行
	pipe := d.redis.Pipeline()
	for videoID, delta := range agg {
		member := strconv.FormatInt(videoID, 10)
		pipe.ZIncrBy(context.Background(), key, float64(delta), member)
	}
	pipe.Expire(context.Background(), key, expire)
	_, err := pipe.Exec(context.Background())
	if err != nil {
		d.log.Warnf("批量增加视频请求计数失败: %v", err)
	}
	return err
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
