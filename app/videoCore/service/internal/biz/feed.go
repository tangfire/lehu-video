// biz/feed.go - 最终修复版
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
	Score     float64 `json:"score"`
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

type FeedStrategy struct {
	PushThreshold     int64         // 推模式粉丝数阈值
	BigVCacheKey      string        // 大V缓存key
	BigVCacheTTL      time.Duration // 大V缓存过期时间
	TimelineMaxSize   int           // 单个用户Timeline保留的最大视频数
	TimelineTTL       time.Duration // Timeline过期时间
	MaxFollowing      int           // 拉取关注列表时最多考虑的用户数（防止过度拉取）
	MaxPullSize       int           // 单次拉取最大视频数
	FollowerBatchSize int           // 粉丝分批处理每批数量
}

type FeedUsecase struct {
	videoRepo     VideoRepo
	followRepo    FollowRepo
	redis         *redis.Client
	kafkaProducer KafkaProducer
	log           *log.Helper
	recentViewed  *RecentViewedManager
	strategy      *FeedStrategy
	hotPool       *HotPoolService
	cancelHotPool context.CancelFunc
}

func NewFeedUsecase(
	videoRepo VideoRepo,
	followRepo FollowRepo,
	redis *redis.Client,
	kafka KafkaProducer,
	recentViewed *RecentViewedManager,
	logger log.Logger,
) *FeedUsecase {
	strategy := &FeedStrategy{
		PushThreshold:     10000,
		BigVCacheKey:      "big_v_users",
		BigVCacheTTL:      10 * time.Minute, // 缩短缓存时间
		TimelineMaxSize:   1000,
		TimelineTTL:       7 * 24 * time.Hour,
		MaxFollowing:      1000,
		MaxPullSize:       100,
		FollowerBatchSize: 500,
	}
	usecase := &FeedUsecase{
		videoRepo:     videoRepo,
		followRepo:    followRepo,
		redis:         redis,
		kafkaProducer: kafka,
		log:           log.NewHelper(logger),
		recentViewed:  recentViewed,
		strategy:      strategy,
	}
	ctx, cancel := context.WithCancel(context.Background())
	usecase.hotPool = NewHotPoolService(videoRepo, redis, logger)
	usecase.cancelHotPool = cancel
	go usecase.hotPool.Run(ctx)
	return usecase
}

func (uc *FeedUsecase) Close() {
	if uc.cancelHotPool != nil {
		uc.cancelHotPool()
	}
}

// GetFeed 核心Feed流接口
func (uc *FeedUsecase) GetFeed(ctx context.Context, query *FeedQuery) (*FeedResult, error) {
	var items []*FeedItem
	var err error

	switch query.FeedType {
	case 0:
		items, err = uc.getFollowingFeed(ctx, query)
	case 1:
		items, err = uc.getRecommendFeed(ctx, query)
	case 2:
		items, err = uc.getHotFeed(ctx, query)
	case 3:
		items, err = uc.getMixedFeed(ctx, query)
	default:
		items, err = uc.getMixedFeed(ctx, query)
	}
	if err != nil {
		return nil, err
	}

	// 去重、过滤已看过的
	items = uc.filterFeedItems(ctx, query.UserID, items)

	nextTime := uc.calculateNextTime(items)
	return &FeedResult{Items: items, NextTime: nextTime}, nil
}

// todo 那是不是要有代码，我们发布视频的时候，要推给粉丝呢
// getFollowingFeed 关注流（推拉结合）
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

	// 如果已经足够，直接返回
	if len(allItems) >= int(query.PageSize) {
		return allItems[:query.PageSize], nil
	}

	// 2. 如果不够，从关注的普通用户拉取（拉模式）
	remaining := int(query.PageSize) - len(allItems)
	// 获取已存在于timeline中的videoID，用于去重
	existingIDs := make([]string, 0, len(allItems))
	for _, item := range allItems {
		existingIDs = append(existingIDs, item.VideoID)
	}
	pullItems, err := uc.pullFromFollowing(ctx, query.UserID, query.LatestTime, remaining, existingIDs)
	if err != nil {
		uc.log.Warnf("从关注列表拉取失败: %v", err)
	} else {
		allItems = append(allItems, pullItems...)
	}
	return allItems, nil
}

// getTimelineItems 从Redis有序集合获取timeline，score为时间戳
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
		parts := strings.Split(z.Member.(string), ":")
		if len(parts) == 3 {
			timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
			items = append(items, &FeedItem{
				VideoID:   parts[0],
				AuthorID:  parts[1],
				Timestamp: timestamp,
				Score:     z.Score,
			})
		}
	}
	return items, nil
}

// pullFromFollowing 从关注的普通用户拉取最新视频，支持排除已有视频
func (uc *FeedUsecase) pullFromFollowing(ctx context.Context, userID string, latestTime int64, limit int, excludeIDs []string) ([]*FeedItem, error) {
	// 1. 获取用户关注列表（分页获取全部）
	var allFollowing []string
	page := 1
	pageSize := 1000 // 每次获取1000个
	for {
		pageStats := &PageStats{
			Page:     int32(page),
			PageSize: int32(pageSize),
		}
		following, err := uc.followRepo.ListFollowing(ctx, userID, 0, pageStats)
		if err != nil {
			return nil, err
		}
		if len(following) == 0 {
			break
		}
		allFollowing = append(allFollowing, following...)
		if len(following) < pageSize {
			break
		}
		page++
	}
	if len(allFollowing) == 0 {
		return nil, nil
	}

	// 2. 过滤出普通用户（非大V）
	var normalUsers []string
	for _, authorID := range allFollowing {
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

	// 3. 从数据库获取最新视频（排除已存在ID）
	videos, err := uc.videoRepo.GetVideosByAuthorsExclude(ctx, normalUsers, latestTime, limit, excludeIDs)
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

// getRecommendFeed 推荐流（基于热门池，降级为最新视频）
func (uc *FeedUsecase) getRecommendFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	items, err := uc.hotPool.GetHotFeedItems(ctx, int(query.PageSize))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		uc.log.Infof("热门池为空，降级为最新视频，user_id=%s", query.UserID)
		return uc.getLatestFeed(ctx, query)
	}
	return items, nil
}

// getHotFeed 热门流（直接取热门池）
func (uc *FeedUsecase) getHotFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	items, err := uc.hotPool.GetHotFeedItems(ctx, int(query.PageSize))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return uc.getLatestFeed(ctx, query)
	}
	return items, nil
}

// getLatestFeed 获取最新发布的视频（后备方法）
func (uc *FeedUsecase) getLatestFeed(ctx context.Context, query *FeedQuery) ([]*FeedItem, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}
	videos, err := uc.videoRepo.GetVideoListByTime(ctx, latestTime, int(query.PageSize))
	if err != nil {
		return nil, err
	}
	items := make([]*FeedItem, 0, len(videos))
	for _, v := range videos {
		items = append(items, &FeedItem{
			VideoID:   strconv.FormatInt(v.Id, 10),
			AuthorID:  strconv.FormatInt(v.Author.Id, 10),
			Timestamp: v.UploadTime.Unix(),
			Score:     float64(v.UploadTime.Unix()),
		})
	}
	return items, nil
}

// getMixedFeed 混合流（30%关注 + 70%推荐，去重）
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

	// 合并并去重（优先保留关注流的顺序）
	allItems := uc.mergeAndDeduplicate(followItems, recommendItems)
	return allItems, nil
}

// mergeAndDeduplicate 合并两个切片并去重，保持第一个切片顺序
func (uc *FeedUsecase) mergeAndDeduplicate(first, second []*FeedItem) []*FeedItem {
	seen := make(map[string]bool)
	result := make([]*FeedItem, 0, len(first)+len(second))
	for _, item := range first {
		if !seen[item.VideoID] {
			seen[item.VideoID] = true
			result = append(result, item)
		}
	}
	for _, item := range second {
		if !seen[item.VideoID] {
			seen[item.VideoID] = true
			result = append(result, item)
		}
	}
	return result
}

// filterFeedItems 过滤掉用户最近看过的视频
func (uc *FeedUsecase) filterFeedItems(ctx context.Context, userID string, items []*FeedItem) []*FeedItem {
	if userID == "0" || userID == "" {
		return items
	}
	videoIDs := make([]string, 0, len(items))
	for _, item := range items {
		videoIDs = append(videoIDs, item.VideoID)
	}
	existsMap, err := uc.recentViewed.BatchExists(ctx, userID, videoIDs)
	if err != nil {
		uc.log.Warnf("批量查询最近观看失败: %v", err)
		return items
	}
	filtered := make([]*FeedItem, 0, len(items))
	for _, item := range items {
		if !existsMap[item.VideoID] {
			filtered = append(filtered, item)
		}
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

// isBigV 判断是否为大V，缓存10分钟
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
	_ = uc.redis.Set(ctx, cacheKey, strconv.FormatBool(isBigV), uc.strategy.BigVCacheTTL).Err()
	return isBigV, nil
}

// PushToUserTimeline 推送视频到用户Timeline（推模式）
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
	pipe.ZRemRangeByRank(ctx, key, 0, -int64(uc.strategy.TimelineMaxSize)-1)
	pipe.Expire(ctx, key, uc.strategy.TimelineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// VideoPublishedHandler 视频发布事件处理
func (uc *FeedUsecase) VideoPublishedHandler(ctx context.Context, videoID, authorID string) error {
	timestamp := time.Now().Unix()
	// 所有用户都通过 Kafka 异步推送
	go uc.pushPublishEventToKafka(videoID, authorID, timestamp)
	// 加入热门池（异步）
	go uc.hotPool.AddVideo(context.Background(), videoID, authorID, timestamp)
	return nil
}

// pushPublishEventToKafka 将发布事件发往Kafka（原pushBigVEventToKafka改名）
func (uc *FeedUsecase) pushPublishEventToKafka(videoID, authorID string, timestamp int64) {
	event := VideoPublishEvent{
		VideoID:   videoID,
		AuthorID:  authorID,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(event)
	if err := uc.kafkaProducer.SendMessage("video_publish_topic", []byte(videoID), data); err != nil {
		uc.log.Errorf("发送Kafka消息失败: %v", err)
	}
}

// pushToFollowersSync 同步推送到粉丝（批量）
func (uc *FeedUsecase) pushToFollowersSync(ctx context.Context, authorID string, item *TimelineItem) {
	offset := 0
	limit := uc.strategy.FollowerBatchSize
	for {
		followers, total, err := uc.followRepo.GetFollowersPaginated(ctx, authorID, offset, limit)
		if err != nil {
			uc.log.Errorf("分页获取粉丝失败: %v", err)
			return
		}
		uc.pushTimelineToUsersBatch(ctx, followers, item)
		if int64(len(followers)+offset) >= total {
			break
		}
		offset += limit
	}
}

// pushTimelineToUsersBatch 批量推送 Timeline 给多个用户（Redis Pipeline）
func (uc *FeedUsecase) pushTimelineToUsersBatch(ctx context.Context, userIDs []string, item *TimelineItem) {
	if len(userIDs) == 0 {
		return
	}
	// 分批处理，每批不超过500人，避免Pipeline过大
	const batchSize = 500
	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		batch := userIDs[i:end]
		// 使用 Redis Pipeline 批量操作
		pipe := uc.redis.Pipeline()
		member := fmt.Sprintf("%s:%s:%d", item.VideoID, item.AuthorID, item.Timestamp)
		score := float64(item.Timestamp)
		// 为每个粉丝的 timeline 添加视频
		for _, uid := range batch {
			key := fmt.Sprintf("timeline:%s", uid)
			// ZAdd: 添加到有序集合，score 是时间戳（用于排序）
			pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
			// 保持 timeline 大小不超过限制
			pipe.ZRemRangeByRank(ctx, key, 0, -int64(uc.strategy.TimelineMaxSize)-1)
			// 设置过期时间
			pipe.Expire(ctx, key, uc.strategy.TimelineTTL)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			uc.log.Errorf("批量推送 timeline 失败: %v", err)
		}
	}
}

// pushBigVEventToKafka 大V事件发往Kafka
func (uc *FeedUsecase) pushBigVEventToKafka(videoID, authorID string, timestamp int64) {
	event := VideoPublishEvent{
		VideoID:   videoID,
		AuthorID:  authorID,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(event)
	if err := uc.kafkaProducer.SendMessage("video_publish_topic", []byte(videoID), data); err != nil {
		uc.log.Errorf("发送Kafka消息失败: %v", err)
	}
}

// GetHotVideos 获取热门视频ID列表
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
