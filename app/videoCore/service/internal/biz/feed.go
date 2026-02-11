// biz/feed.go - 完整修复版
package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type FeedItem struct {
	VideoID   string  `json:"video_id"`
	AuthorID  string  `json:"author_id"`
	Timestamp int64   `json:"timestamp"`
	Score     float64 `json:"score"` // 用于推荐流的热度分数
}

type TimelineItem struct {
	VideoID   string `json:"video_id"`
	AuthorID  string `json:"author_id"`
	Timestamp int64  `json:"timestamp"`
}

type FeedQuery struct {
	UserID     string
	LatestTime int64
	PageSize   int32
	FeedType   int32 // 0:关注 1:推荐 2:热门 3:混合
}

type FeedResult struct {
	Items    []*FeedItem
	NextTime int64
}

type KafkaProducer interface {
	SendMessage(topic string, key, value []byte) error
}

type FeedStrategy struct {
	PushThreshold     int64         // 推模式粉丝数阈值
	BigVCacheKey      string        // 大V缓存key
	TimelineMaxSize   int           // 单个用户Timeline保留的最大视频数
	TimelineTTL       time.Duration // Timeline过期时间
	MaxFollowing      int           // 拉取关注列表时最多考虑的用户数
	MaxPullSize       int           // 单次拉取最大视频数
	FollowerBatchSize int           // 粉丝分批处理每批数量
}

type FeedUsecase struct {
	videoRepo     VideoRepo
	followRepo    FollowRepo
	redis         *redis.Client
	kafkaProducer KafkaProducer
	log           *log.Helper
	bloomManager  *BloomFilterManager
	strategy      *FeedStrategy
	hotPool       *HotPoolService
	cancelHotPool context.CancelFunc // 用于停止 hotPool goroutine
}

func NewFeedUsecase(
	videoRepo VideoRepo,
	followRepo FollowRepo,
	redis *redis.Client,
	kafka KafkaProducer,
	logger log.Logger,
) (*FeedUsecase, error) {
	strategy := &FeedStrategy{
		PushThreshold:     10000,
		BigVCacheKey:      "big_v_users",
		TimelineMaxSize:   1000,
		TimelineTTL:       7 * 24 * time.Hour,
		MaxFollowing:      1000,
		MaxPullSize:       100,
		FollowerBatchSize: 500,
	}

	bloomMgr, err := NewBloomFilterManager(redis)
	if err != nil {
		return nil, err
	}

	usecase := &FeedUsecase{
		videoRepo:     videoRepo,
		followRepo:    followRepo,
		redis:         redis,
		kafkaProducer: kafka,
		log:           log.NewHelper(logger),
		bloomManager:  bloomMgr,
		strategy:      strategy,
	}

	// 启动热门池，并保存 cancel 函数以便关闭
	ctx, cancel := context.WithCancel(context.Background())
	usecase.hotPool = NewHotPoolService(videoRepo, redis, logger)
	usecase.cancelHotPool = cancel
	go usecase.hotPool.Run(ctx)

	return usecase, nil
}

// Close 释放资源，停止后台 goroutine
func (uc *FeedUsecase) Close() {
	if uc.cancelHotPool != nil {
		uc.cancelHotPool()
	}
}

// GetFeed - 核心Feed流接口
func (uc *FeedUsecase) GetFeed(ctx context.Context, query *FeedQuery) (*FeedResult, error) {
	var items []*FeedItem
	var err error

	switch query.FeedType {
	case 0: // 关注流
		items, err = uc.getFollowingFeed(ctx, query)
	case 1: // 推荐流
		items, err = uc.getRecommendFeed(ctx, query)
	case 2: // 热门流
		items, err = uc.getHotFeed(ctx, query)
	case 3: // 混合流
		items, err = uc.getMixedFeed(ctx, query)
	default:
		items, err = uc.getMixedFeed(ctx, query)
	}
	if err != nil {
		return nil, err
	}

	// 去重、过滤已看过的
	items = uc.filterFeedItems(ctx, query.UserID, items)

	// 计算下次请求的时间
	nextTime := uc.calculateNextTime(items)

	return &FeedResult{
		Items:    items,
		NextTime: nextTime,
	}, nil
}

// getFollowingFeed - 关注流（推拉结合）
func (uc *FeedUsecase) getFollowingFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	var allItems []*FeedItem

	// 1. 从Redis Timeline获取（推模式）
	timelineKey := fmt.Sprintf("timeline:%s", query.UserID)
	timelineItems, err := uc.getTimelineItems(ctx, timelineKey, query.LatestTime, int(query.PageSize))
	if err != nil {
		uc.log.Warnf("从timeline获取失败: %v", err)
	} else {
		allItems = append(allItems, timelineItems...)
	}

	// 2. 如果不够，从关注的普通用户拉取（拉模式）
	if len(allItems) < int(query.PageSize) {
		remaining := int(query.PageSize) - len(allItems)
		pullItems, err := uc.pullFromFollowing(ctx, query.UserID, query.LatestTime, remaining)
		if err != nil {
			uc.log.Warnf("从关注列表拉取失败: %v", err)
		} else {
			allItems = append(allItems, pullItems...)
		}
	}
	return allItems, nil
}

// getTimelineItems - 从Redis有序集合获取timeline，score为时间戳
func (uc *FeedUsecase) getTimelineItems(ctx context.Context, key string, maxScore int64, limit int) ([]*FeedItem, error) {
	opts := &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(maxScore, 10),
		Offset: 0,
		Count:  int64(limit),
	}
	members, err := uc.redis.ZRevRangeByScoreWithScores(ctx, key, opts).Result()
	if err != nil {
		return nil, err
	}
	items := make([]*FeedItem, 0, len(members))
	for _, z := range members {
		// member格式: video_id:author_id:timestamp
		parts := strings.Split(z.Member.(string), ":")
		if len(parts) == 3 {
			timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
			items = append(items, &FeedItem{
				VideoID:   parts[0],
				AuthorID:  parts[1],
				Timestamp: timestamp,
				Score:     z.Score, // 这里score是时间戳
			})
		}
	}
	return items, nil
}

// pullFromFollowing - 从关注的**普通用户**拉取最新视频（大V的内容应已推送）
func (uc *FeedUsecase) pullFromFollowing(ctx context.Context, userID string, latestTime int64, limit int) ([]*FeedItem, error) {
	// 1. 获取用户关注列表（限制数量）
	following, err := uc.followRepo.ListFollowing(ctx, userID, 0, &PageStats{
		Page:     1,
		PageSize: int32(uc.strategy.MaxFollowing),
	})
	if err != nil || len(following) == 0 {
		return nil, err
	}

	// 2. 过滤出普通用户（非大V）
	var normalUsers []string
	for _, authorID := range following {
		isBigV, err := uc.isBigV(ctx, authorID)
		if err != nil {
			uc.log.Warnf("检查大V状态失败: %v", err)
			continue
		}
		if !isBigV {
			normalUsers = append(normalUsers, authorID)
		}
	}
	if len(normalUsers) == 0 {
		return nil, nil
	}

	// 3. 从数据库获取最新视频
	videos, err := uc.videoRepo.GetVideosByAuthors(ctx, normalUsers, latestTime, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*FeedItem, 0, len(videos))
	for _, video := range videos {
		items = append(items, &FeedItem{
			VideoID:   strconv.FormatInt(video.Id, 10),
			AuthorID:  strconv.FormatInt(video.Author.Id, 10),
			Timestamp: video.UploadTime.Unix(),
			Score:     float64(video.UploadTime.Unix()),
		})
	}
	return items, nil
}

// getRecommendFeed - 推荐流（基于热门池）
func (uc *FeedUsecase) getRecommendFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	// 从热门池获取FeedItem（已包含作者、时间戳、热度分数）
	return uc.hotPool.GetHotFeedItems(ctx, int(query.PageSize))
}

// getHotFeed - 热门流（直接取热门池）
func (uc *FeedUsecase) getHotFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	return uc.hotPool.GetHotFeedItems(ctx, int(query.PageSize))
}

// getMixedFeed - 混合流（30%关注 + 70%推荐）
func (uc *FeedUsecase) getMixedFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	followCount := int(float64(query.PageSize) * 0.3)
	recommendCount := int(query.PageSize) - followCount

	var wg sync.WaitGroup
	var followItems, recommendItems []*FeedItem
	var followErr, recErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		followQuery := *query
		followQuery.PageSize = int32(followCount)
		followQuery.FeedType = 0
		followItems, followErr = uc.getFollowingFeed(ctx, &followQuery)
	}()
	go func() {
		defer wg.Done()
		recommendQuery := *query
		recommendQuery.PageSize = int32(recommendCount)
		recommendQuery.FeedType = 1
		recommendItems, recErr = uc.getRecommendFeed(ctx, &recommendQuery)
	}()
	wg.Wait()

	if followErr != nil {
		return nil, followErr
	}
	if recErr != nil {
		return nil, recErr
	}
	allItems := append(followItems, recommendItems...)
	uc.shuffleItems(allItems)
	return allItems, nil
}

// filterFeedItems - 布隆过滤器去重（已看过）
func (uc *FeedUsecase) filterFeedItems(ctx context.Context, userID string, items []*FeedItem) []*FeedItem {
	if userID == "0" || userID == "" {
		return items
	}
	filtered := make([]*FeedItem, 0, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%s:%s", item.VideoID, userID)
		exists, err := uc.bloomManager.TestAndAdd(ctx, userID, key)
		if err != nil {
			uc.log.Warnf("布隆过滤器操作失败: %v", err)
		}
		if !exists {
			filtered = append(filtered, item)
		}
	}
	// 异步保存过滤器
	if uf := uc.bloomManager.GetOrCreate(ctx, userID); uf != nil {
		uc.bloomManager.SaveAsync(userID, uf)
	}
	return filtered
}

// shuffleItems 打乱顺序
func (uc *FeedUsecase) shuffleItems(items []*FeedItem) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range items {
		j := rand.Intn(i + 1)
		items[i], items[j] = items[j], items[i]
	}
}

// calculateNextTime 下次请求时间 = 最小时间戳 - 1
func (uc *FeedUsecase) calculateNextTime(items []*FeedItem) int64 {
	if len(items) == 0 {
		return time.Now().Unix()
	}
	minTime := items[0].Timestamp
	for _, item := range items {
		if item.Timestamp < minTime {
			minTime = item.Timestamp
		}
	}
	return minTime - 1
}

// isBigV 判断是否为大V（粉丝数≥阈值）
func (uc *FeedUsecase) isBigV(ctx context.Context, authorID string) (bool, error) {
	cacheKey := fmt.Sprintf("%s:%s", uc.strategy.BigVCacheKey, authorID)
	cached, err := uc.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		return cached == "1", nil
	}
	followerCount, err := uc.followRepo.CountFollowers(ctx, authorID)
	if err != nil {
		return false, err
	}
	isBigV := followerCount >= uc.strategy.PushThreshold
	_ = uc.redis.Set(ctx, cacheKey, strconv.FormatBool(isBigV), time.Hour).Err()
	return isBigV, nil
}

// PushToUserTimeline - 推送视频到用户Timeline（推模式）
// score直接使用时间戳，member格式: video_id:author_id:timestamp
func (uc *FeedUsecase) PushToUserTimeline(ctx context.Context, userID string, items []*TimelineItem) error {
	if len(items) == 0 {
		return nil
	}
	key := fmt.Sprintf("timeline:%s", userID)
	pipe := uc.redis.Pipeline()
	for _, item := range items {
		member := fmt.Sprintf("%s:%s:%d", item.VideoID, item.AuthorID, item.Timestamp)
		pipe.ZAdd(ctx, key, redis.Z{
			Score:  float64(item.Timestamp),
			Member: member,
		})
	}
	// 修剪，保留最新的 N 条
	pipe.ZRemRangeByRank(ctx, key, 0, -int64(uc.strategy.TimelineMaxSize)-1)
	pipe.Expire(ctx, key, uc.strategy.TimelineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// VideoPublishedHandler - 视频发布事件处理
// VideoPublishedHandler - 视频发布事件处理（✅ 使用 context.Background 异步）
func (uc *FeedUsecase) VideoPublishedHandler(ctx context.Context, videoID, authorID string) error {
	isBigV, err := uc.isBigV(ctx, authorID)
	if err != nil {
		return err
	}
	timestamp := time.Now().Unix()
	item := &TimelineItem{
		VideoID:   videoID,
		AuthorID:  authorID,
		Timestamp: timestamp,
	}
	if isBigV {
		// 大V：异步发送Kafka
		go uc.pushBigVEventToKafka(videoID, authorID, timestamp)
	} else {
		// 普通用户：使用独立 context.Background 推送
		go uc.pushToFollowersSync(context.Background(), authorID, item)
	}
	// 加入热门池（异步）
	go uc.hotPool.AddVideo(context.Background(), videoID, authorID, timestamp)
	return nil
}

// pushToFollowersSync - 同步推送到粉丝（✅ 批量 Pipeline 优化）
func (uc *FeedUsecase) pushToFollowersSync(ctx context.Context, authorID string, item *TimelineItem) {
	offset := 0
	limit := uc.strategy.FollowerBatchSize
	for {
		followers, total, err := uc.followRepo.GetFollowersPaginated(ctx, authorID, offset, limit)
		if err != nil {
			uc.log.Errorf("分页获取粉丝失败: %v", err)
			return
		}
		// ✅ 批量推送：一个 Pipeline 处理一批粉丝的所有 ZAdd 操作
		uc.pushTimelineToUsersBatch(ctx, followers, item)
		if int64(len(followers)+offset) >= total {
			break
		}
		offset += limit
	}
}

// pushTimelineToUsersBatch - 批量推送 Timeline 给多个用户（Redis Pipeline）
func (uc *FeedUsecase) pushTimelineToUsersBatch(ctx context.Context, userIDs []string, item *TimelineItem) {
	if len(userIDs) == 0 {
		return
	}
	pipe := uc.redis.Pipeline()
	member := fmt.Sprintf("%s:%s:%d", item.VideoID, item.AuthorID, item.Timestamp)
	score := float64(item.Timestamp)

	for _, uid := range userIDs {
		key := fmt.Sprintf("timeline:%s", uid)
		pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
		// 修剪并设置过期（每个 key 单独操作，但放在同一个 pipeline 中）
		pipe.ZRemRangeByRank(ctx, key, 0, -int64(uc.strategy.TimelineMaxSize)-1)
		pipe.Expire(ctx, key, uc.strategy.TimelineTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		uc.log.Errorf("批量推送 timeline 失败: %v", err)
	}
}

// pushBigVEventToKafka - 大V事件发往Kafka
func (uc *FeedUsecase) pushBigVEventToKafka(videoID, authorID string, timestamp int64) {
	event := VideoPublishEvent{
		VideoID:   videoID,
		AuthorID:  authorID,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(event)
	if err := uc.kafkaProducer.SendMessage("video_publish", []byte(videoID), data); err != nil {
		uc.log.Errorf("发送Kafka消息失败: %v", err)
	}
}

// GetHotVideos - 获取热门视频ID列表（兼容旧调用）
func (uc *FeedUsecase) GetHotVideos(ctx context.Context, limit int) ([]string, error) {
	items, err := uc.hotPool.GetHotFeedItems(ctx, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.VideoID
	}
	return ids, nil
}
