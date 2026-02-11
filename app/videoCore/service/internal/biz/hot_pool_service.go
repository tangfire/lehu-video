// biz/hot_pool_service.go - 修复版
package biz

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// HotPoolService 热门池管理，存储FeedItem所需全部信息
type HotPoolService struct {
	videoRepo VideoRepo
	redis     *redis.Client
	log       *log.Helper
}

func NewHotPoolService(videoRepo VideoRepo, redis *redis.Client, logger log.Logger) *HotPoolService {
	return &HotPoolService{
		videoRepo: videoRepo,
		redis:     redis,
		log:       log.NewHelper(logger),
	}
}

// Run 定时刷新热门池
func (s *HotPoolService) Run(ctx context.Context) {
	s.refreshHotPool(ctx)
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.refreshHotPool(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// GetHotFeedItems 获取热门Feed项（包含完整信息）
func (s *HotPoolService) GetHotFeedItems(ctx context.Context, limit int) ([]*FeedItem, error) {
	key := "feed:hot:pool"
	if limit <= 0 {
		limit = 20
	}
	// 按分数从高到低取
	results, err := s.redis.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	items := make([]*FeedItem, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		// member格式: video_id:author_id:timestamp
		parts := strings.Split(member, ":")
		if len(parts) != 3 {
			continue
		}
		timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
		items = append(items, &FeedItem{
			VideoID:   parts[0],
			AuthorID:  parts[1],
			Timestamp: timestamp,
			Score:     z.Score,
		})
	}
	return items, nil
}

// AddVideo 新发布视频加入热门池（初始分 = 当前时间戳）
func (s *HotPoolService) AddVideo(ctx context.Context, videoID, authorID string, timestamp int64) {
	key := "feed:hot:pool"
	member := s.buildMember(videoID, authorID, timestamp)
	z := redis.Z{
		Score:  float64(timestamp), // 初始用时间戳，后续刷新会重新计算
		Member: member,
	}
	if err := s.redis.ZAdd(ctx, key, z).Err(); err != nil {
		s.log.Warnf("添加视频到热门池失败: %v", err)
	}
	// 修剪池子大小，保留最新1000个（按分数，即时间戳）
	s.redis.ZRemRangeByRank(ctx, key, 0, -1000)
}

// refreshHotPool 刷新热门池，重新计算热度分数
func (s *HotPoolService) refreshHotPool(ctx context.Context) {
	// 1. 从数据库获取候选视频（近7天，按互动量排序取2000条）
	hotVideos, err := s.videoRepo.GetHotVideos(ctx, 2000)
	if err != nil {
		s.log.Errorf("获取热门视频失败: %v", err)
		return
	}
	// 2. 计算热度分数并存入Redis
	scoredMembers := make([]redis.Z, 0, len(hotVideos))
	now := time.Now().Unix()
	for _, video := range hotVideos {
		score := s.calculateHotScore(video, now)
		member := s.buildMember(
			strconv.FormatInt(video.Id, 10),
			strconv.FormatInt(video.Author.Id, 10),
			video.UploadTime.Unix(),
		)
		scoredMembers = append(scoredMembers, redis.Z{
			Score:  score,
			Member: member,
		})
	}
	// 3. 原子替换热门池
	key := "feed:hot:pool"
	pipe := s.redis.Pipeline()
	pipe.Del(ctx, key)
	if len(scoredMembers) > 0 {
		pipe.ZAdd(ctx, key, scoredMembers...)
	}
	pipe.Expire(ctx, key, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Errorf("写入热门池失败: %v", err)
		return
	}
	s.log.Infof("热门池刷新完成，视频数量: %d", len(scoredMembers))
}

// calculateHotScore 热度算法：威尔逊区间 + 时间衰减
func (s *HotPoolService) calculateHotScore(video *Video, now int64) float64 {
	like := float64(video.LikeCount)
	comment := float64(video.CommentCount)
	// 总互动量（可调整权重）
	n := like*1.5 + comment*1.0
	p := 0.0
	if n > 0 {
		p = (like * 1.5) / n
	}
	z := 1.96 // 95%置信度
	score := 0.0
	if n > 0 {
		score = (p + z*z/(2*n) - z*math.Sqrt((p*(1-p)+z*z/(4*n))/n)) / (1 + z*z/n)
	}
	// 时间衰减：24小时半衰期
	hours := float64(now-video.UploadTime.Unix()) / 3600.0
	timeDecay := math.Pow(0.5, hours/24.0)
	finalScore := score * timeDecay * 1000
	if finalScore < 0 {
		finalScore = 0
	}
	return finalScore
}

// buildMember 构造member字符串
func (s *HotPoolService) buildMember(videoID, authorID string, timestamp int64) string {
	return strings.Join([]string{videoID, authorID, strconv.FormatInt(timestamp, 10)}, ":")
}

// 以下为旧接口兼容或内部使用，可保留
func (s *HotPoolService) GetHotVideos(ctx context.Context, limit int) ([]string, error) {
	items, err := s.GetHotFeedItems(ctx, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.VideoID
	}
	return ids, nil
}
