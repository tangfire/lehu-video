package data

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"lehu-video/app/videoCore/service/internal/biz"
)

type videoCounterRepo struct {
	redis *redis.Client
	log   *log.Helper
}

func NewVideoCounterRepo(redis *redis.Client, logger log.Logger) biz.VideoCounterRepo {
	return &videoCounterRepo{
		redis: redis,
		log:   log.NewHelper(logger),
	}
}

func videoCounterKey(videoId int64) string {
	return fmt.Sprintf("video:counter:%d", videoId)
}

const dirtyVideoSetKey = "video:counter:dirty"

// IncrVideoCounter 增加视频计数
func (r *videoCounterRepo) IncrVideoCounter(ctx context.Context, videoId int64, field string, delta int64) error {
	key := videoCounterKey(videoId)
	pipe := r.redis.Pipeline()
	pipe.HIncrBy(ctx, key, field, delta)
	pipe.Expire(ctx, key, 7*24*time.Hour)
	pipe.SAdd(ctx, dirtyVideoSetKey, videoId)
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.Warnf("IncrVideoCounter failed: videoId=%d, field=%s, delta=%d, err=%v", videoId, field, delta, err)
	}
	return err
}

// BatchIncrVideoCounters 批量增加多个视频的计数（field固定为view_count）
func (r *videoCounterRepo) BatchIncrVideoCounters(ctx context.Context, counts map[int64]int64) error {
	if len(counts) == 0 {
		return nil
	}
	pipe := r.redis.Pipeline()
	for vid, delta := range counts {
		key := videoCounterKey(vid)
		pipe.HIncrBy(ctx, key, "view_count", delta)
		pipe.Expire(ctx, key, 7*24*time.Hour)
		pipe.SAdd(ctx, dirtyVideoSetKey, vid)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.Warnf("BatchIncrVideoCounters failed: %v", err)
	}
	return err
}

// BatchIncrFields 批量增加多个视频的多个字段（原子操作）
func (r *videoCounterRepo) BatchIncrFields(ctx context.Context, counts map[int64]map[string]int64) error {
	if len(counts) == 0 {
		return nil
	}
	pipe := r.redis.Pipeline()
	for vid, fields := range counts {
		key := videoCounterKey(vid)
		for field, delta := range fields {
			if delta == 0 {
				continue
			}
			pipe.HIncrBy(ctx, key, field, delta)
		}
		pipe.Expire(ctx, key, 7*24*time.Hour)
		pipe.SAdd(ctx, dirtyVideoSetKey, vid)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.Warnf("BatchIncrFields failed: %v", err)
	}
	return err
}

// GetVideoCounters 获取视频计数
func (r *videoCounterRepo) GetVideoCounters(ctx context.Context, videoId int64, fields ...string) (map[string]int64, error) {
	key := videoCounterKey(videoId)
	if len(fields) == 0 {
		vals, err := r.redis.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		res := make(map[string]int64, len(vals))
		for k, v := range vals {
			val, _ := strconv.ParseInt(v, 10, 64)
			res[k] = val
		}
		return res, nil
	}
	vals, err := r.redis.HMGet(ctx, key, fields...).Result()
	if err != nil {
		return nil, err
	}
	res := make(map[string]int64, len(fields))
	for i, field := range fields {
		if vals[i] == nil {
			res[field] = 0
		} else {
			val, _ := strconv.ParseInt(vals[i].(string), 10, 64)
			res[field] = val
		}
	}
	return res, nil
}

// BatchGetVideoCounters 批量获取视频计数
func (r *videoCounterRepo) BatchGetVideoCounters(ctx context.Context, videoIds []int64, fields ...string) (map[int64]map[string]int64, error) {
	if len(videoIds) == 0 {
		return map[int64]map[string]int64{}, nil
	}
	pipe := r.redis.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(videoIds))
	for i, vid := range videoIds {
		key := videoCounterKey(vid)
		cmds[i] = pipe.HGetAll(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]map[string]int64)
	for i, vid := range videoIds {
		vals, err := cmds[i].Result()
		if err != nil {
			continue
		}
		counters := make(map[string]int64)
		if len(fields) == 0 {
			for k, v := range vals {
				val, _ := strconv.ParseInt(v, 10, 64)
				counters[k] = val
			}
		} else {
			for _, field := range fields {
				if v, ok := vals[field]; ok {
					val, _ := strconv.ParseInt(v, 10, 64)
					counters[field] = val
				} else {
					counters[field] = 0
				}
			}
		}
		if len(counters) > 0 {
			result[vid] = counters
		}
	}
	return result, nil
}

// MarkVideoDirty 标记视频为脏
func (r *videoCounterRepo) MarkVideoDirty(ctx context.Context, videoId int64) error {
	return r.redis.SAdd(ctx, dirtyVideoSetKey, videoId).Err()
}

// GetDirtyVideoIDs 获取所有脏视频ID
func (r *videoCounterRepo) GetDirtyVideoIDs(ctx context.Context) ([]int64, error) {
	vals, err := r.redis.SMembers(ctx, dirtyVideoSetKey).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(vals))
	for _, v := range vals {
		id, _ := strconv.ParseInt(v, 10, 64)
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ClearDirtyFlag 清除脏标记
func (r *videoCounterRepo) ClearDirtyFlag(ctx context.Context, videoId int64) error {
	return r.redis.SRem(ctx, dirtyVideoSetKey, videoId).Err()
}

// SetVideoCounters 设置视频计数（覆盖，用于回填）
func (r *videoCounterRepo) SetVideoCounters(ctx context.Context, videoId int64, counters map[string]int64) error {
	key := videoCounterKey(videoId)
	fields := make(map[string]interface{})
	for k, v := range counters {
		fields[k] = v
	}
	pipe := r.redis.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, 7*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}
