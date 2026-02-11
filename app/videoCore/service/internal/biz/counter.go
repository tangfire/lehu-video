package biz

import "context"

type CounterRepo interface {
	// 获取单个用户的多个计数字段
	GetUserCounters(ctx context.Context, userId int64) (map[string]int64, error)
	// 批量获取多个用户的指定计数字段
	BatchGetUserCounters(ctx context.Context, userIds []int64, fields []string) (map[int64]map[string]int64, error)
	// 增加用户计数
	IncrUserCounter(ctx context.Context, userId int64, field string, delta int64) (int64, error)
	// 减少用户计数
	DecrUserCounter(ctx context.Context, userId int64, field string, delta int64) (int64, error)
	// 设置用户计数（覆盖）
	SetUserCounters(ctx context.Context, userId int64, counters map[string]int64) error
	// 获取所有需要同步的用户ID（最近有更新的）
	GetDirtyUserIDs(ctx context.Context) ([]int64, error)
	// 清除脏标记（同步后调用）
	ClearDirtyFlag(ctx context.Context, userId int64) error
}
