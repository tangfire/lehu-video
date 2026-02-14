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
