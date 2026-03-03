package data

import (
	"context"
	"fmt"
	"lehu-video/app/videoCore/service/internal/biz"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type userCounterRepo struct {
	redis *redis.Client
	log   *log.Helper
}

func NewUserCounterRepo(redis *redis.Client, logger log.Logger) biz.UserCounterRepo {
	return &userCounterRepo{
		redis: redis,
		log:   log.NewHelper(logger),
	}
}

func userCounterKey(userId int64) string {
	return fmt.Sprintf("user:counter:%d", userId)
}

const dirtyUserSetKey = "user:counter:dirty"

func (r *userCounterRepo) GetUserCounters(ctx context.Context, userId int64) (map[string]int64, error) {
	key := userCounterKey(userId)
	vals, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, nil
	}
	res := make(map[string]int64)
	for k, v := range vals {
		val, _ := strconv.ParseInt(v, 10, 64)
		res[k] = val
	}
	return res, nil
}

func (r *userCounterRepo) BatchGetUserCounters(ctx context.Context, userIds []int64, fields []string) (map[int64]map[string]int64, error) {
	pipe := r.redis.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(userIds))
	for i, uid := range userIds {
		key := userCounterKey(uid)
		cmds[i] = pipe.HGetAll(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]map[string]int64)
	for i, uid := range userIds {
		vals, err := cmds[i].Result()
		if err != nil {
			continue
		}
		if len(vals) == 0 {
			continue
		}
		counters := make(map[string]int64)
		for _, f := range fields {
			if v, ok := vals[f]; ok {
				counters[f], _ = strconv.ParseInt(v, 10, 64)
			} else {
				counters[f] = 0
			}
		}
		if len(counters) > 0 {
			result[uid] = counters
		}
	}
	return result, nil
}

func (r *userCounterRepo) IncrUserCounter(ctx context.Context, userId int64, field string, delta int64) (int64, error) {
	key := userCounterKey(userId)
	pipe := r.redis.Pipeline()
	incr := pipe.HIncrBy(ctx, key, field, delta)
	pipe.SAdd(ctx, dirtyUserSetKey, userId)
	pipe.Expire(ctx, key, 7*24*time.Hour)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (r *userCounterRepo) DecrUserCounter(ctx context.Context, userId int64, field string, delta int64) (int64, error) {
	return r.IncrUserCounter(ctx, userId, field, -delta)
}

func (r *userCounterRepo) SetUserCounters(ctx context.Context, userId int64, counters map[string]int64) error {
	key := userCounterKey(userId)
	fields := make(map[string]interface{})
	for k, v := range counters {
		fields[k] = v
	}
	pipe := r.redis.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.SAdd(ctx, dirtyUserSetKey, userId)
	pipe.Expire(ctx, key, 7*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *userCounterRepo) GetDirtyUserIDs(ctx context.Context) ([]int64, error) {
	vals, err := r.redis.SMembers(ctx, dirtyUserSetKey).Result()
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

func (r *userCounterRepo) ClearDirtyFlag(ctx context.Context, userId int64) error {
	return r.redis.SRem(ctx, dirtyUserSetKey, userId).Err()
}
