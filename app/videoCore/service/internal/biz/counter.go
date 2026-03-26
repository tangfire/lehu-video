package biz

import "context"

type UserCounterRepo interface {
	// GetUserCounters 获取单个用户的多个计数字段
	GetUserCounters(ctx context.Context, userId int64) (map[string]int64, error)
	// BatchGetUserCounters 批量获取多个用户的指定计数字段
	BatchGetUserCounters(ctx context.Context, userIds []int64, fields []string) (map[int64]map[string]int64, error)
	// IncrUserCounter 增加用户计数
	IncrUserCounter(ctx context.Context, userId int64, field string, delta int64) (int64, error)
	// DecrUserCounter 减少用户计数
	DecrUserCounter(ctx context.Context, userId int64, field string, delta int64) (int64, error)
	// SetUserCounters 设置用户计数（覆盖）
	SetUserCounters(ctx context.Context, userId int64, counters map[string]int64) error
	// GetDirtyUserIDs 获取所有需要同步的用户ID（最近有更新的）
	GetDirtyUserIDs(ctx context.Context) ([]int64, error)
	// ClearDirtyFlag 清除脏标记（同步后调用）
	ClearDirtyFlag(ctx context.Context, userId int64) error
	// BatchIncrUserCounters 批量增加多个用户的指定计数字段（用于消费者批量更新）
	BatchIncrUserCounters(ctx context.Context, counts map[int64]map[string]int64) error
}

// VideoCounterRepo 视频计数器仓储接口
type VideoCounterRepo interface {
	// IncrVideoCounter 增加视频某个字段的计数
	IncrVideoCounter(ctx context.Context, videoId int64, field string, delta int64) error

	// GetVideoCounters 获取视频的多个计数字段
	GetVideoCounters(ctx context.Context, videoId int64, fields ...string) (map[string]int64, error)

	// BatchGetVideoCounters 批量获取视频计数
	BatchGetVideoCounters(ctx context.Context, videoIds []int64, fields ...string) (map[int64]map[string]int64, error)

	// MarkVideoDirty 标记视频为脏（需要同步到MySQL）
	MarkVideoDirty(ctx context.Context, videoId int64) error

	// GetDirtyVideoIDs 获取所有脏视频ID
	GetDirtyVideoIDs(ctx context.Context) ([]int64, error)

	// ClearDirtyFlag 清除脏标记
	ClearDirtyFlag(ctx context.Context, videoId int64) error

	SetVideoCounters(ctx context.Context, videoId int64, counters map[string]int64) error

	// BatchIncrVideoCounters 批量增加多个视频的计数（field固定为view_count）
	BatchIncrVideoCounters(ctx context.Context, counts map[int64]int64) error

	// 新增：批量增加多个视频的多个字段
	BatchIncrFields(ctx context.Context, counts map[int64]map[string]int64) error
}

// CommentCounterRepo 评论计数器仓储接口
type CommentCounterRepo interface {
	// IncrCommentCounter 增加评论某个字段的计数
	IncrCommentCounter(ctx context.Context, commentId int64, field string, delta int64) error

	// GetCommentCounters 获取评论的多个计数字段
	GetCommentCounters(ctx context.Context, commentId int64, fields ...string) (map[string]int64, error)

	// BatchGetCommentCounters 批量获取评论计数
	BatchGetCommentCounters(ctx context.Context, commentIds []int64, fields ...string) (map[int64]map[string]int64, error)

	// MarkCommentDirty 标记评论为脏（需要同步到 MySQL）
	MarkCommentDirty(ctx context.Context, commentId int64) error

	// GetDirtyCommentIDs 获取所有脏评论 ID
	GetDirtyCommentIDs(ctx context.Context) ([]int64, error)

	// ClearDirtyFlag 清除脏标记
	ClearDirtyFlag(ctx context.Context, commentId int64) error

	// SetCommentCounters 设置评论计数（覆盖）
	SetCommentCounters(ctx context.Context, commentId int64, counters map[string]int64) error

	// BatchIncrCommentCounters 批量增加多个评论的计数
	BatchIncrCommentCounters(ctx context.Context, counts map[int64]int64) error

	// BatchIncrFields 批量增加多个评论的多个字段
	BatchIncrFields(ctx context.Context, counts map[int64]map[string]int64) error
}
