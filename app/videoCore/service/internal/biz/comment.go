package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lehu-video/app/videoCore/service/internal/pkg/idgen"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// ---------- Command/Query ----------
type CreateCommentCommand struct {
	VideoID     int64
	UserID      int64
	ParentID    int64
	ReplyUserID int64
	Content     string
}

type CreateCommentResult struct {
	Comment *Comment
}

type RemoveCommentCommand struct {
	CommentID int64
	UserID    int64
}

type RemoveCommentResult struct{}

type ListVideoCommentsQuery struct {
	VideoID   int64
	PageStats PageStats
}

type ListVideoCommentsResult struct {
	Comments []*Comment
	Total    int64
}

type ListChildCommentsQuery struct {
	ParentID  int64
	PageStats PageStats
}

type ListChildCommentsResult struct {
	Comments []*Comment
	Total    int64
}

type GetCommentQuery struct {
	CommentID int64
}

type GetCommentResult struct {
	Comment *Comment
}

type CountVideoCommentsQuery struct {
	VideoIDs []int64
}

type CountVideoCommentsResult struct {
	Counts map[int64]int64
}

type CountUserCommentsQuery struct {
	UserIDs []int64
}

type CountUserCommentsResult struct {
	Counts map[int64]int64
}

// ---------- 业务模型 ----------
type Comment struct {
	ID          int64
	VideoID     int64
	UserID      int64
	ParentID    int64
	ReplyUserID int64
	Content     string
	CreateTime  time.Time
	LikeCount   int64 // 冗余字段
	ReplyCount  int64 // 冗余字段，直接子评论数
	IsDeleted   bool
}

// CommentRepo 接口（不变，保持原有方法）
type CommentRepo interface {
	Create(ctx context.Context, comment *Comment) (int64, error)
	GetByID(ctx context.Context, id int64) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id int64, userID int64) error
	SoftDelete(ctx context.Context, id int64, userID int64) error
	BatchSoftDelete(ctx context.Context, ids []int64) error

	FindByIDs(ctx context.Context, ids []int64) ([]*Comment, error)
	FindByCondition(ctx context.Context, condition map[string]interface{}) ([]*Comment, error)
	CountByCondition(ctx context.Context, condition map[string]interface{}) (int64, error)

	CountByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64]int64, error)
	CountByUserIDs(ctx context.Context, userIDs []int64) (map[int64]int64, error)

	IncrReplyCount(ctx context.Context, parentID int64, delta int) error
	GetReplyCount(ctx context.Context, commentID int64) (int64, error)

	ExecTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// CommentUsecase 业务逻辑（简化版）
type CommentUsecase struct {
	repo         CommentRepo
	redis        *redis.Client
	videoCounter VideoCounterRepo
	idGen        idgen.Generator
	log          *log.Helper
	sfg          singleflight.Group
}

func NewCommentUsecase(
	repo CommentRepo,
	redis *redis.Client,
	videoCounter VideoCounterRepo,
	idGen idgen.Generator,
	logger log.Logger,
) *CommentUsecase {
	return &CommentUsecase{
		repo:         repo,
		redis:        redis,
		videoCounter: videoCounter,
		idGen:        idGen,
		log:          log.NewHelper(logger),
	}
}

// 缓存键常量
const cacheKeyComment = "comment:%d"

// CreateComment 创建评论
func (uc *CommentUsecase) CreateComment(ctx context.Context, cmd *CreateCommentCommand) (*CreateCommentResult, error) {
	if cmd.VideoID <= 0 || cmd.UserID <= 0 || cmd.Content == "" {
		return nil, ErrInvalidParams
	}

	now := time.Now()
	comment := &Comment{
		ID:          uc.idGen.NextID(),
		VideoID:     cmd.VideoID,
		UserID:      cmd.UserID,
		ParentID:    cmd.ParentID,
		ReplyUserID: cmd.ReplyUserID,
		Content:     cmd.Content,
		CreateTime:  now,
		LikeCount:   0,
		ReplyCount:  0,
		IsDeleted:   false,
	}

	// 事务内创建评论并更新父评论回复计数
	err := uc.repo.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := uc.repo.Create(txCtx, comment); err != nil {
			return err
		}
		if comment.ParentID > 0 {
			if err := uc.repo.IncrReplyCount(txCtx, comment.ParentID, 1); err != nil {
				return err
			}
			// 删除父评论的缓存（因为回复数变了）
			uc.deleteCommentCache(ctx, comment.ParentID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 同步更新视频评论计数
	if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.VideoID, "comment_count", 1); err != nil {
		uc.log.Warnf("更新视频评论计数失败: videoId=%d, err=%v", cmd.VideoID, err)
	}

	// 删除可能存在的视频评论列表缓存（如果之前缓存过总数，可考虑删除，但列表未缓存，无需操作）
	// 本设计未缓存列表，所以无需额外操作

	return &CreateCommentResult{Comment: comment}, nil
}

// RemoveComment 删除评论
func (uc *CommentUsecase) RemoveComment(ctx context.Context, cmd *RemoveCommentCommand) (*RemoveCommentResult, error) {
	comment, err := uc.repo.GetByID(ctx, cmd.CommentID)
	if err != nil {
		return nil, err
	}
	if comment == nil || comment.IsDeleted {
		return nil, ErrCommentNotFound
	}
	if comment.UserID != cmd.UserID {
		return nil, ErrNoPermission
	}

	var subIDs []int64
	err = uc.repo.ExecTx(ctx, func(txCtx context.Context) error {
		// 软删除当前评论
		if err := uc.repo.SoftDelete(txCtx, cmd.CommentID, cmd.UserID); err != nil {
			return err
		}

		if comment.ParentID == 0 { // 主评论，需要级联删除子评论
			subs, err := uc.repo.FindByCondition(txCtx, map[string]interface{}{
				"parent_id":  comment.ID,
				"is_deleted": false,
			})
			if err != nil {
				return err
			}
			for _, sub := range subs {
				subIDs = append(subIDs, sub.ID)
			}
			if len(subIDs) > 0 {
				if err := uc.repo.BatchSoftDelete(txCtx, subIDs); err != nil {
					return err
				}
			}
		} else { // 子评论，减少父评论的回复计数
			if err := uc.repo.IncrReplyCount(txCtx, comment.ParentID, -1); err != nil {
				return err
			}
			// 删除父评论缓存
			uc.deleteCommentCache(ctx, comment.ParentID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 异步更新视频评论计数
	go func() {
		bgCtx := context.Background()
		delta := int64(-1) // 当前评论
		if comment.ParentID == 0 {
			delta -= int64(len(subIDs)) // 加上所有子评论
		}
		if err := uc.videoCounter.IncrVideoCounter(bgCtx, comment.VideoID, "comment_count", delta); err != nil {
			uc.log.Warnf("更新视频评论计数失败: videoId=%d, err=%v", comment.VideoID, err)
		}
	}()

	// 删除当前评论缓存
	uc.deleteCommentCache(ctx, comment.ID)
	for _, subID := range subIDs {
		uc.deleteCommentCache(ctx, subID)
	}

	return &RemoveCommentResult{}, nil
}

// GetCommentByID 获取单个评论详情（带 Redis 缓存 + SingleFlight）
func (uc *CommentUsecase) GetCommentByID(ctx context.Context, query *GetCommentQuery) (*GetCommentResult, error) {
	if query.CommentID <= 0 {
		return nil, ErrInvalidParams
	}
	cacheKey := fmt.Sprintf(cacheKeyComment, query.CommentID)

	// 尝试从 Redis 读取
	var comment Comment
	found, err := uc.getFromRedis(ctx, cacheKey, &comment)
	if err == nil && found {
		if comment.IsDeleted {
			return nil, ErrCommentNotFound
		}
		return &GetCommentResult{Comment: &comment}, nil
	}

	// 使用 SingleFlight 合并并发请求，防止缓存击穿
	v, err, _ := uc.sfg.Do(cacheKey, func() (interface{}, error) {
		// 双重检查缓存（防止在等待期间其他请求已回填）
		found, err = uc.getFromRedis(ctx, cacheKey, &comment)
		if err == nil && found {
			return &comment, nil
		}

		dbComment, err := uc.repo.GetByID(ctx, query.CommentID)
		if err != nil {
			return nil, err
		}
		if dbComment == nil || dbComment.IsDeleted {
			// 缓存空值，防止穿透（短时间）
			nullCache := Comment{ID: query.CommentID, IsDeleted: true}
			_ = uc.setToRedis(ctx, cacheKey, nullCache, 30) // 30秒空缓存
			return nil, ErrCommentNotFound
		}
		// 存入 Redis 缓存（1小时）
		_ = uc.setToRedis(ctx, cacheKey, dbComment, 3600)
		return dbComment, nil
	})

	if err != nil {
		return nil, err
	}
	return &GetCommentResult{Comment: v.(*Comment)}, nil
}

// ListVideoComments 获取视频的主评论列表（直接查数据库，按时间倒序）
func (uc *CommentUsecase) ListVideoComments(ctx context.Context, query *ListVideoCommentsQuery) (*ListVideoCommentsResult, error) {
	if query.VideoID <= 0 {
		return nil, ErrInvalidParams
	}
	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize
	condition := map[string]interface{}{
		"video_id":   query.VideoID,
		"parent_id":  0,
		"is_deleted": false,
		"order_by":   "created_at DESC",
		"offset":     offset,
		"limit":      query.PageStats.PageSize,
	}
	comments, err := uc.repo.FindByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}
	total, err := uc.repo.CountByCondition(ctx, map[string]interface{}{
		"video_id":   query.VideoID,
		"parent_id":  0,
		"is_deleted": false,
	})
	if err != nil {
		return nil, err
	}
	return &ListVideoCommentsResult{Comments: comments, Total: total}, nil
}

// ListChildComments 获取子评论列表（按时间正序）
func (uc *CommentUsecase) ListChildComments(ctx context.Context, query *ListChildCommentsQuery) (*ListChildCommentsResult, error) {
	if query.ParentID <= 0 {
		return nil, ErrInvalidParams
	}
	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize
	condition := map[string]interface{}{
		"parent_id":  query.ParentID,
		"is_deleted": false,
		"order_by":   "created_at ASC",
		"offset":     offset,
		"limit":      query.PageStats.PageSize,
	}
	comments, err := uc.repo.FindByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}
	total, err := uc.repo.CountByCondition(ctx, map[string]interface{}{
		"parent_id": query.ParentID,
	})
	if err != nil {
		return nil, err
	}
	return &ListChildCommentsResult{Comments: comments, Total: total}, nil
}

// CountVideoComments 统计视频评论数（优先从计数器获取）
func (uc *CommentUsecase) CountVideoComments(ctx context.Context, query *CountVideoCommentsQuery) (*CountVideoCommentsResult, error) {
	if len(query.VideoIDs) == 0 {
		return &CountVideoCommentsResult{Counts: map[int64]int64{}}, nil
	}
	// 从 videoCounter 获取（Redis）
	countsMap, err := uc.videoCounter.BatchGetVideoCounters(ctx, query.VideoIDs, "comment_count")
	if err == nil {
		counts := make(map[int64]int64, len(query.VideoIDs))
		for _, vid := range query.VideoIDs {
			if m, ok := countsMap[vid]; ok {
				counts[vid] = m["comment_count"]
			} else {
				counts[vid] = 0
			}
		}
		return &CountVideoCommentsResult{Counts: counts}, nil
	}
	// 降级查数据库
	uc.log.Warnf("从 videoCounter 获取计数失败，降级查询数据库: %v", err)
	dbCounts, err := uc.repo.CountByVideoIDs(ctx, query.VideoIDs)
	if err != nil {
		return nil, err
	}
	return &CountVideoCommentsResult{Counts: dbCounts}, nil
}

// CountUserComments 统计用户评论数
func (uc *CommentUsecase) CountUserComments(ctx context.Context, query *CountUserCommentsQuery) (*CountUserCommentsResult, error) {
	if len(query.UserIDs) == 0 {
		return &CountUserCommentsResult{Counts: map[int64]int64{}}, nil
	}
	dbCounts, err := uc.repo.CountByUserIDs(ctx, query.UserIDs)
	if err != nil {
		return nil, err
	}
	return &CountUserCommentsResult{Counts: dbCounts}, nil
}

// ---------- 私有辅助方法 ----------
func (uc *CommentUsecase) deleteCommentCache(ctx context.Context, commentID int64) {
	key := fmt.Sprintf(cacheKeyComment, commentID)
	if err := uc.redis.Del(ctx, key).Err(); err != nil {
		uc.log.Warnf("删除评论缓存失败: key=%s, err=%v", key, err)
	}
}

func (uc *CommentUsecase) getFromRedis(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := uc.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	err = json.Unmarshal(data, dest)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (uc *CommentUsecase) setToRedis(ctx context.Context, key string, value interface{}, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return uc.redis.Set(ctx, key, data, time.Duration(ttl)*time.Second).Err()
}

// ---------- 错误定义 ----------
var (
	ErrInvalidParams         = errors.New("invalid parameters")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrParentCommentNotFound = errors.New("parent comment not found")
	ErrNoPermission          = errors.New("no permission")
)
