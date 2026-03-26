// data/comment_counter.go
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

const (
	commentCounterPrefix = "comment:counter:"
	dirtyCommentSetKey   = "dirty:comments"
	commentCounterTTL    = 7 * 24 * time.Hour // 评论计数器 TTL 7 天
)

type commentCounterRepo struct {
	redis *redis.Client
	log   *log.Helper
}

func NewCommentCounterRepo(redis *redis.Client, logger log.Logger) biz.CommentCounterRepo {
	return &commentCounterRepo{
		redis: redis,
		log:   log.NewHelper(logger),
	}
}

func commentCounterKey(commentId int64) string {
	return fmt.Sprintf("%s%d", commentCounterPrefix, commentId)
}

// IncrCommentCounter 增加评论某个字段的计数
func (r *commentCounterRepo) IncrCommentCounter(ctx context.Context, commentId int64, field string, delta int64) error {
	key := commentCounterKey(commentId)
	if err := r.redis.HIncrBy(ctx, key, field, delta).Err(); err != nil {
		r.log.Warnf("IncrCommentCounter failed: commentId=%d, field=%s, delta=%d, err=%v", commentId, field, delta, err)
		return err
	}
	// 设置过期时间
	r.redis.Expire(ctx, key, commentCounterTTL)
	// 标记为脏数据
	r.redis.SAdd(ctx, dirtyCommentSetKey, commentId)
	return nil
}

// GetCommentCounters 获取评论的多个计数字段
func (r *commentCounterRepo) GetCommentCounters(ctx context.Context, commentId int64, fields ...string) (map[string]int64, error) {
	key := commentCounterKey(commentId)
	vals, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	counters := make(map[string]int64)
	for _, field := range fields {
		if v, ok := vals[field]; ok {
			val, _ := strconv.ParseInt(v, 10, 64)
			counters[field] = val
		} else {
			counters[field] = 0
		}
	}
	return counters, nil
}

// BatchGetCommentCounters 批量获取评论计数
func (r *commentCounterRepo) BatchGetCommentCounters(ctx context.Context, commentIds []int64, fields ...string) (map[int64]map[string]int64, error) {
	if len(commentIds) == 0 {
		return map[int64]map[string]int64{}, nil
	}

	pipe := r.redis.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(commentIds))
	for i, cid := range commentIds {
		key := commentCounterKey(cid)
		cmds[i] = pipe.HGetAll(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]map[string]int64)
	for i, cid := range commentIds {
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
			result[cid] = counters
		}
	}
	return result, nil
}

// MarkCommentDirty 标记评论为脏
func (r *commentCounterRepo) MarkCommentDirty(ctx context.Context, commentId int64) error {
	return r.redis.SAdd(ctx, dirtyCommentSetKey, commentId).Err()
}

// GetDirtyCommentIDs 获取所有脏评论 ID
func (r *commentCounterRepo) GetDirtyCommentIDs(ctx context.Context) ([]int64, error) {
	vals, err := r.redis.SMembers(ctx, dirtyCommentSetKey).Result()
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
func (r *commentCounterRepo) ClearDirtyFlag(ctx context.Context, commentId int64) error {
	return r.redis.SRem(ctx, dirtyCommentSetKey, commentId).Err()
}

// SetCommentCounters 设置评论计数（覆盖）
func (r *commentCounterRepo) SetCommentCounters(ctx context.Context, commentId int64, counters map[string]int64) error {
	key := commentCounterKey(commentId)
	fields := make(map[string]interface{})
	for k, v := range counters {
		fields[k] = v
	}
	pipe := r.redis.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, commentCounterTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// BatchIncrCommentCounters 批量增加多个评论的计数
func (r *commentCounterRepo) BatchIncrCommentCounters(ctx context.Context, counts map[int64]int64) error {
	if len(counts) == 0 {
		return nil
	}
	pipe := r.redis.Pipeline()
	for cid, delta := range counts {
		key := commentCounterKey(cid)
		pipe.HIncrBy(ctx, key, "like_count", delta)
		pipe.Expire(ctx, key, commentCounterTTL)
		pipe.SAdd(ctx, dirtyCommentSetKey, cid)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.Warnf("BatchIncrCommentCounters failed: %v", err)
	}
	return err
}

// BatchIncrFields 批量增加多个评论的多个字段
func (r *commentCounterRepo) BatchIncrFields(ctx context.Context, counts map[int64]map[string]int64) error {
	if len(counts) == 0 {
		return nil
	}
	pipe := r.redis.Pipeline()
	for cid, fields := range counts {
		key := commentCounterKey(cid)
		for field, delta := range fields {
			if delta == 0 {
				continue
			}
			pipe.HIncrBy(ctx, key, field, delta)
		}
		pipe.Expire(ctx, key, commentCounterTTL)
		pipe.SAdd(ctx, dirtyCommentSetKey, cid)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.Warnf("BatchIncrFields failed: %v", err)
	}
	return err
}
