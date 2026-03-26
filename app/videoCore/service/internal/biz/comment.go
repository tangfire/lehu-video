package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lehu-video/app/videoCore/service/internal/pkg/idgen"
	"sort"
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

// CommentRepo 接口
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
const (
	// 视频评论集合（Hash）- 存储所有评论详情
	cacheKeyVideoComments = "video:comments:%d:all" // Hash: commentID -> JSON
	cacheTTLVideoComments = 1800                    // 30 分钟
)

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
		uc.log.Warnf("更新视频评论计数失败：videoId=%d, err=%v", cmd.VideoID, err)
	}

	// 同步更新视频评论 Hash 缓存（保证一致性）
	videoKey := fmt.Sprintf(cacheKeyVideoComments, cmd.VideoID)
	if err := uc.cacheSingleCommentToHash(ctx, videoKey, comment); err != nil {
		uc.log.Warnf("添加新评论到 Hash 缓存失败：%v", err)
		// 不返回错误，避免影响主流程
	}

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

	// 异步删除视频评论 Hash 缓存中的对应评论
	go func() {
		ctxBg := context.Background()
		videoKey := fmt.Sprintf(cacheKeyVideoComments, comment.VideoID)
		// HDEL video:comments:1:all "123456"
		uc.redis.HDel(ctxBg, videoKey, fmt.Sprintf("%d", comment.ID))
		for _, sub := range subIDs {
			uc.redis.HDel(ctxBg, videoKey, fmt.Sprintf("%d", sub))
		}
	}()

	return &RemoveCommentResult{}, nil
}

// GetCommentByID 获取单个评论详情（从视频评论 Hash 中获取）
func (uc *CommentUsecase) GetCommentByID(ctx context.Context, query *GetCommentQuery) (*GetCommentResult, error) {
	if query.CommentID <= 0 {
		return nil, ErrInvalidParams
	}

	// TODO: 需要知道 video_id 才能从 Hash 中获取
	// 方案 1: 先查数据库获取 video_id，然后从 Hash 中获取
	// 方案 2: 维护 commentID -> videoID 的映射关系
	// 这里暂时直接查数据库（因为 GetCommentByID 通常调用频率不高）

	dbComment, err := uc.repo.GetByID(ctx, query.CommentID)
	if err != nil {
		return nil, err
	}
	if dbComment == nil || dbComment.IsDeleted {
		return nil, ErrCommentNotFound
	}

	return &GetCommentResult{Comment: dbComment}, nil
}

// ListVideoComments 获取视频的主评论列表（Hash 存储所有评论）
func (uc *CommentUsecase) ListVideoComments(ctx context.Context, query *ListVideoCommentsQuery) (*ListVideoCommentsResult, error) {
	if query.VideoID <= 0 {
		return nil, ErrInvalidParams
	}
	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize
	limit := query.PageStats.PageSize

	// 尝试从 Hash 缓存获取所有评论
	cacheKey := fmt.Sprintf(cacheKeyVideoComments, query.VideoID)
	allComments, err := uc.getVideoCommentsFromHash(ctx, cacheKey)

	if err == nil && len(allComments) > 0 {
		uc.log.Infof("[Cache Hit] 视频 %d 的评论列表从缓存获取", query.VideoID)

		// 过滤主评论（parent_id=0）并按时间倒序排序
		mainComments := make([]*Comment, 0)
		for _, c := range allComments {
			if c.ParentID == 0 && !c.IsDeleted {
				mainComments = append(mainComments, c)
			}
		}

		// 按创建时间倒序排序
		sort.Slice(mainComments, func(i, j int) bool {
			return mainComments[i].CreateTime.After(mainComments[j].CreateTime)
		})

		// 分页
		total := int64(len(mainComments))
		start := int(offset)
		if start >= len(mainComments) {
			return &ListVideoCommentsResult{Comments: []*Comment{}, Total: total}, nil
		}
		end := start + int(limit)
		if end > len(mainComments) {
			end = len(mainComments)
		}
		pagedComments := mainComments[start:end]

		return &ListVideoCommentsResult{Comments: pagedComments, Total: total}, nil
	}

	// 缓存未命中，查询数据库
	uc.log.Infof("[Cache Miss] 视频 %d 的评论列表查询数据库", query.VideoID)
	condition := map[string]interface{}{
		"video_id":   query.VideoID,
		"parent_id":  0,
		"is_deleted": false,
		"order_by":   "created_at DESC",
		"offset":     offset,
		"limit":      limit,
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

	// 异步更新缓存（缓存该视频的所有评论）
	go func() {
		ctxBg := context.Background()
		// 获取该视频的所有评论（包括子评论）
		allCondition := map[string]interface{}{
			"video_id":   query.VideoID,
			"is_deleted": false,
		}
		allComments, _ := uc.repo.FindByCondition(ctxBg, allCondition)
		if err := uc.cacheVideoCommentsToHash(ctxBg, cacheKey, allComments); err != nil {
			uc.log.Warnf("缓存视频 %d 的评论到 Hash 失败：%v", query.VideoID, err)
		}
	}()

	return &ListVideoCommentsResult{Comments: comments, Total: total}, nil
}

// ListChildComments 获取子评论列表（从 Hash 缓存获取）
func (uc *CommentUsecase) ListChildComments(ctx context.Context, query *ListChildCommentsQuery) (*ListChildCommentsResult, error) {
	if query.ParentID <= 0 {
		return nil, ErrInvalidParams
	}

	// 简化方案：直接查数据库（子评论数量通常不多）
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

// cacheVideoCommentsToHash 缓存视频的所有评论到 Redis Hash
func (uc *CommentUsecase) cacheVideoCommentsToHash(ctx context.Context, key string, comments []*Comment) error {
	if len(comments) == 0 {
		return nil
	}

	// 使用 Pipeline 批量写入 Hash
	pipe := uc.redis.Pipeline()
	for _, c := range comments {
		data, err := json.Marshal(c)
		if err != nil {
			uc.log.Warnf("序列化评论失败：%v", err)
			continue
		}
		// HSET video:comments:1:all "123456" "{...}"
		pipe.HSet(ctx, key, fmt.Sprintf("%d", c.ID), string(data))
	}
	// 设置过期时间
	pipe.Expire(ctx, key, time.Duration(cacheTTLVideoComments)*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		uc.log.Warnf("cacheVideoCommentsToHash failed: %v", err)
		return err
	}
	uc.log.Infof("缓存 %d 个评论到 Redis Hash", len(comments))
	return nil
}

// getVideoCommentsFromHash 从 Redis Hash 获取视频的所有评论
func (uc *CommentUsecase) getVideoCommentsFromHash(ctx context.Context, key string) ([]*Comment, error) {
	// HGETALL video:comments:1:all
	valMap, err := uc.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(valMap) == 0 {
		return []*Comment{}, nil
	}

	// 反序列化所有评论
	comments := make([]*Comment, 0, len(valMap))
	for _, valStr := range valMap {
		var comment Comment
		if err := json.Unmarshal([]byte(valStr), &comment); err != nil {
			uc.log.Warnf("反序列化评论失败：%v", err)
			continue
		}
		comments = append(comments, &comment)
	}

	return comments, nil
}

// cacheSingleCommentToHash 缓存单条评论到 Redis Hash（用于更新）
func (uc *CommentUsecase) cacheSingleCommentToHash(ctx context.Context, videoKey string, comment *Comment) error {
	data, err := json.Marshal(comment)
	if err != nil {
		return err
	}
	return uc.redis.HSet(ctx, videoKey, fmt.Sprintf("%d", comment.ID), string(data)).Err()
}

// deleteCommentCache 从视频评论 Hash 中删除指定评论
func (uc *CommentUsecase) deleteCommentCache(ctx context.Context, commentID int64) {
	// 直接查询评论获取 video_id，然后从 Hash 中删除
	comment, err := uc.repo.GetByID(ctx, commentID)
	if err == nil && comment != nil && comment.VideoID > 0 {
		videoCommentsKey := fmt.Sprintf(cacheKeyVideoComments, comment.VideoID)
		// HDel 只删除当前评论，不影响其他评论
		if err := uc.redis.HDel(ctx, videoCommentsKey, fmt.Sprintf("%d", commentID)).Err(); err != nil {
			uc.log.Warnf("从视频评论 Hash 中删除评论失败：videoId=%d, err=%v", comment.VideoID, err)
		}
	} else if err == nil && comment == nil {
		uc.log.Warnf("评论 %d 不存在，无法从 Hash 中删除", commentID)
	}
}

// ---------- 错误定义 ----------
var (
	ErrInvalidParams         = errors.New("invalid parameters")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrParentCommentNotFound = errors.New("parent comment not found")
	ErrNoPermission          = errors.New("no permission")
)
