package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RecentViewedManager 管理用户最近观看的视频ID
type RecentViewedManager struct {
	redis *redis.Client
	ttl   time.Duration // 过期时间，默认24小时
	limit int           // 保留的最大数量，默认100
}

func NewRecentViewedManager(redis *redis.Client) *RecentViewedManager {
	return &RecentViewedManager{
		redis: redis,
		ttl:   24 * time.Hour,
		limit: 100,
	}
}

// Add 记录用户观看了一个视频
func (m *RecentViewedManager) Add(ctx context.Context, userID, videoID string) error {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	now := float64(time.Now().Unix())
	pipe := m.redis.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: now, Member: videoID})
	// 只保留最近 limit 个
	pipe.ZRemRangeByRank(ctx, key, 0, -int64(m.limit)-1)
	pipe.Expire(ctx, key, m.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// Exists 检查视频是否在最近观看列表中
func (m *RecentViewedManager) Exists(ctx context.Context, userID, videoID string) (bool, error) {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	_, err := m.redis.ZScore(ctx, key, videoID).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// BatchExists 批量检查多个视频是否在最近观看列表中
// 返回 map[videoID]bool
func (m *RecentViewedManager) BatchExists(ctx context.Context, userID string, videoIDs []string) (map[string]bool, error) {
	if len(videoIDs) == 0 {
		return map[string]bool{}, nil
	}
	key := fmt.Sprintf("recent_viewed:%s", userID)
	pipe := m.redis.Pipeline()
	cmds := make([]*redis.FloatCmd, len(videoIDs))
	for i, vid := range videoIDs {
		cmds[i] = pipe.ZScore(ctx, key, vid)
	}
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	result := make(map[string]bool)
	for i, cmd := range cmds {
		_, err := cmd.Result()
		result[videoIDs[i]] = err == nil
	}
	return result, nil
}

// GetRecentVideoIDs 获取用户最近观看的视频ID 列表（按时间倒序）
// limit: 返回的数量限制，如果 <=0 则返回全部
func (m *RecentViewedManager) GetRecentVideoIDs(ctx context.Context, userID string, limit int) ([]string, error) {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	var start, stop int64
	if limit <= 0 {
		start = 0
		stop = -1 // 返回所有
	} else {
		start = 0
		stop = int64(limit) - 1
	}

	// ZRevRange：按 Score 降序（最新的在前）
	return m.redis.ZRevRange(ctx, key, start, stop).Result()
}

// GetRecentVideoIDsWithTime 获取用户最近观看的视频ID 及观看时间
// 返回：[]VideoWatchTime，按观看时间倒序
func (m *RecentViewedManager) GetRecentVideoIDsWithTime(ctx context.Context, userID string, limit int) ([]VideoWatchTime, error) {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	var start, stop int64
	if limit <= 0 {
		start = 0
		stop = -1
	} else {
		start = 0
		stop = int64(limit) - 1
	}

	// ZRevRangeWithScores：获取成员和分数（时间戳）
	zResults, err := m.redis.ZRevRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, err
	}

	result := make([]VideoWatchTime, len(zResults))
	for i, z := range zResults {
		result[i] = VideoWatchTime{
			VideoID:   z.Member.(string),
			WatchTime: time.Unix(int64(z.Score), 0),
		}
	}
	return result, nil
}

// GetPagination 分页获取用户最近观看记录
// page: 页码（从 1 开始），pageSize: 每页数量
// 返回：视频ID 列表、总数、错误
func (m *RecentViewedManager) GetPagination(ctx context.Context, userID string, page, pageSize int32) ([]string, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	key := fmt.Sprintf("recent_viewed:%s", userID)

	// 获取总数
	total, err := m.redis.ZCard(ctx, key).Result()
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []string{}, 0, nil
	}

	// 计算偏移量
	offset := (page - 1) * pageSize
	stop := offset + pageSize - 1

	// 获取指定范围的记录（倒序，最新的在前）
	videoIDs, err := m.redis.ZRevRange(ctx, key, int64(offset), int64(stop)).Result()
	if err != nil {
		return nil, 0, err
	}

	return videoIDs, total, nil
}

// Remove 移除指定的观看记录
func (m *RecentViewedManager) Remove(ctx context.Context, userID, videoID string) error {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	return m.redis.ZRem(ctx, key, videoID).Err()
}

// BatchRemove 批量移除多个观看记录
func (m *RecentViewedManager) BatchRemove(ctx context.Context, userID string, videoIDs []string) error {
	if len(videoIDs) == 0 {
		return nil
	}
	key := fmt.Sprintf("recent_viewed:%s", userID)
	args := make([]interface{}, len(videoIDs))
	for i, vid := range videoIDs {
		args[i] = vid
	}
	return m.redis.ZRem(ctx, key, args...).Err()
}

// Clear 清空用户的观看记录
func (m *RecentViewedManager) Clear(ctx context.Context, userID string) error {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	return m.redis.Del(ctx, key).Err()
}

// Count 获取用户观看记录总数
func (m *RecentViewedManager) Count(ctx context.Context, userID string) (int64, error) {
	key := fmt.Sprintf("recent_viewed:%s", userID)
	return m.redis.ZCard(ctx, key).Result()
}

// VideoWatchTime 视频观看时间结构
type VideoWatchTime struct {
	VideoID   string
	WatchTime time.Time
}
