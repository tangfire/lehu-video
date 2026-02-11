package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coocood/freecache"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ============ Command/Query 结构体定义 ============

// CreateCommentCommand 创建评论命令
type CreateCommentCommand struct {
	VideoID     int64
	UserID      int64
	ParentID    int64
	ReplyUserID int64
	Content     string
}

// CreateCommentResult 创建评论结果
type CreateCommentResult struct {
	Comment *Comment
}

// RemoveCommentCommand 删除评论命令
type RemoveCommentCommand struct {
	CommentID int64
	UserID    int64
}

// RemoveCommentResult 删除评论结果
type RemoveCommentResult struct{}

// ListVideoCommentsQuery 查询视频评论列表
type ListVideoCommentsQuery struct {
	VideoID      int64
	PageStats    PageStats
	WithChildren bool // 是否加载子评论
}

// ListVideoCommentsResult 视频评论列表结果
type ListVideoCommentsResult struct {
	Comments []*Comment
	Total    int64
}

// ListChildCommentsQuery 查询子评论列表
type ListChildCommentsQuery struct {
	ParentID  int64
	PageStats PageStats
}

// ListChildCommentsResult 子评论列表结果
type ListChildCommentsResult struct {
	Comments []*Comment
	Total    int64
}

// GetCommentQuery 获取评论详情查询
type GetCommentQuery struct {
	CommentID int64
}

// GetCommentResult 获取评论详情结果
type GetCommentResult struct {
	Comment *Comment
}

// CountVideoCommentsQuery 统计视频评论数查询
type CountVideoCommentsQuery struct {
	VideoIDs []int64
}

// CountVideoCommentsResult 统计视频评论数结果
type CountVideoCommentsResult struct {
	Counts map[int64]int64
}

// CountUserCommentsQuery 统计用户评论数查询
type CountUserCommentsQuery struct {
	UserIDs []int64
}

// CountUserCommentsResult 统计用户评论数结果
type CountUserCommentsResult struct {
	Counts map[int64]int64
}

// ============ 业务模型 ============

// Comment 业务层评论模型
type Comment struct {
	ID          int64
	VideoID     int64
	UserID      int64
	ParentID    int64
	ReplyUserID int64
	Content     string
	CreateTime  time.Time
	LikeCount   int64
	ReplyCount  int64
	ChildCount  int64
	IsDeleted   bool

	// 嵌套的子评论（根据需要加载）
	ChildComments []*Comment
}

func (c *Comment) GenerateId() {
	c.ID = int64(uuid.New().ID())
}

// CommentRepo 数据层接口
type CommentRepo interface {
	// 基础CRUD
	Create(ctx context.Context, comment *Comment) (int64, error)
	GetByID(ctx context.Context, id int64) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id int64, userID int64) error
	SoftDelete(ctx context.Context, id int64, userID int64) error

	// 简单查询
	FindByCondition(ctx context.Context, condition map[string]interface{}) ([]*Comment, error)
	CountByCondition(ctx context.Context, condition map[string]interface{}) (int64, error)

	// 批量查询
	FindByIDs(ctx context.Context, ids []int64) ([]*Comment, error)
	CountByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64]int64, error)
	CountByUserIDs(ctx context.Context, userIDs []int64) (map[int64]int64, error)

	// 新增：按父评论ID分组计数
	CountGroupByParentID(ctx context.Context, parentIDs []int64) (map[int64]int64, error)

	// 获取点赞数统计
	GetLikeCounts(ctx context.Context, commentIDs []int64) (map[int64]int64, error)
}

// CommentUsecase 业务逻辑层
type CommentUsecase struct {
	repo        CommentRepo
	cache       *freecache.Cache  // 本地缓存
	redis       *redis.Client     // Redis 客户端
	hotDetector *HotVideoDetector // 热点检测器
	log         *log.Helper
}

func NewCommentUsecase(repo CommentRepo, cache *freecache.Cache, redis *redis.Client, hotDetector *HotVideoDetector, logger log.Logger) *CommentUsecase {
	return &CommentUsecase{
		repo:        repo,
		cache:       cache,
		redis:       redis,
		hotDetector: hotDetector,
		log:         log.NewHelper(logger),
	}
}

// CreateComment 创建评论
func (uc *CommentUsecase) CreateComment(ctx context.Context, cmd *CreateCommentCommand) (*CreateCommentResult, error) {
	// 1. 参数验证
	if cmd.VideoID <= 0 || cmd.UserID <= 0 || cmd.Content == "" {
		return nil, ErrInvalidParams
	}

	// 3. 构建评论对象
	now := time.Now()
	comment := &Comment{
		ID:          0,
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

	comment.GenerateId()

	// 4. 如果parentID > 0，验证父评论是否存在
	if cmd.ParentID > 0 {
		parentComment, err := uc.repo.GetByID(ctx, cmd.ParentID)
		if err != nil {
			return nil, err
		}
		if parentComment == nil || parentComment.IsDeleted {
			return nil, ErrParentCommentNotFound
		}
	}

	// 5. 保存评论
	_, err := uc.repo.Create(ctx, comment)
	if err != nil {
		return nil, err
	}

	return &CreateCommentResult{
		Comment: comment,
	}, nil
}

// RemoveComment 删除评论
func (uc *CommentUsecase) RemoveComment(ctx context.Context, cmd *RemoveCommentCommand) (*RemoveCommentResult, error) {
	// 1. 验证评论是否存在且属于该用户
	comment, err := uc.repo.GetByID(ctx, cmd.CommentID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, ErrCommentNotFound
	}

	// 2. 验证权限（只能删除自己的评论）
	if comment.UserID != cmd.UserID {
		return nil, ErrNoPermission
	}

	// 3. 删除评论（软删除）
	err = uc.repo.SoftDelete(ctx, cmd.CommentID, cmd.UserID)
	if err != nil {
		return nil, err
	}

	return &RemoveCommentResult{}, nil
}

// ListVideoComments 获取视频评论列表（带多级缓存）
// 注意：由于 Comment 结构体包含 ChildComments 字段，
// 且为循环引用，直接 JSON 序列化会有问题。
// 需要修改 Comment 结构体，移除 ChildComments 字段的指针循环，
// 或者在序列化时忽略。简单方案：在缓存时不存储 ChildComments，查询时单独加载。
// 但为了简化，我们缓存完整评论树时使用 []*Comment 本身没有问题（只要没有互相引用），
// 但注意 ChildComments 中的 ParentID 指向父评论，不是循环引用。所以可以正常序列化。
func (uc *CommentUsecase) ListVideoComments(ctx context.Context, query *ListVideoCommentsQuery) (*ListVideoCommentsResult, error) {
	// 1. 记录该视频的一次请求（用于热点统计）
	uc.hotDetector.IncrRequestCount(ctx, query.VideoID)

	// 2. 判断是否为热门视频，决定缓存时间
	isHot := uc.hotDetector.IsHotVideo(ctx, query.VideoID)
	var localTTL, redisTTL int
	if isHot {
		localTTL = 30  // 本地缓存30秒
		redisTTL = 300 // Redis缓存5分钟
	} else {
		localTTL = 5  // 本地缓存5秒
		redisTTL = 60 // Redis缓存1分钟
	}

	// 3. 构建缓存 key
	cacheKey := fmt.Sprintf("video_comments:%d:page:%d:size:%d", query.VideoID, query.PageStats.Page, query.PageStats.PageSize)

	// 4. 尝试从本地缓存读取
	var result ListVideoCommentsResult
	found, err := getFromLocalCache(uc.cache, cacheKey, &result)
	if err == nil && found {
		uc.log.Debugf("本地缓存命中: %s", cacheKey)
		return &result, nil
	}

	// 5. 尝试从 Redis 读取
	found, err = uc.getFromRedis(ctx, cacheKey, &result)
	if err == nil && found {
		uc.log.Debugf("Redis缓存命中: %s", cacheKey)
		// 回填本地缓存
		_ = setToLocalCache(uc.cache, cacheKey, &result, localTTL)
		return &result, nil
	}

	// 6. 缓存未命中，查询数据库
	uc.log.Debugf("缓存未命中，查询数据库: %s", cacheKey)
	dbResult, err := uc.queryVideoComments(ctx, query)
	if err != nil {
		return nil, err
	}

	// 7. 回填缓存
	_ = setToLocalCache(uc.cache, cacheKey, dbResult, localTTL)
	_ = uc.setToRedis(ctx, cacheKey, dbResult, redisTTL)

	return dbResult, nil
}

// queryVideoComments 原 ListVideoComments 的实际查询逻辑
func (uc *CommentUsecase) queryVideoComments(ctx context.Context, query *ListVideoCommentsQuery) (*ListVideoCommentsResult, error) {
	// 原有的 ListVideoComments 业务逻辑（去掉缓存部分）
	if query.VideoID <= 0 {
		return nil, ErrInvalidParams
	}
	condition := map[string]interface{}{
		"video_id":   query.VideoID,
		"parent_id":  0,
		"is_deleted": false,
		"limit":      query.PageStats.PageSize,
		"offset":     (query.PageStats.Page - 1) * query.PageStats.PageSize,
		"order_by":   "created_at DESC",
	}
	comments, err := uc.repo.FindByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}
	countCondition := map[string]interface{}{
		"video_id":  query.VideoID,
		"parent_id": 0,
	}
	total, err := uc.repo.CountByCondition(ctx, countCondition)
	if err != nil {
		return nil, err
	}
	if query.WithChildren && len(comments) > 0 {
		err = uc.loadChildComments(ctx, comments)
		if err != nil {
			return nil, err
		}
	}
	// 获取点赞数
	commentIDs := make([]int64, len(comments))
	for i, comment := range comments {
		commentIDs[i] = comment.ID
	}
	likeCounts, err := uc.repo.GetLikeCounts(ctx, commentIDs)
	if err != nil {
		uc.log.Warnf("获取点赞数失败: %v", err)
	} else {
		for _, comment := range comments {
			if count, ok := likeCounts[comment.ID]; ok {
				comment.LikeCount = count
			}
		}
	}
	return &ListVideoCommentsResult{
		Comments: comments,
		Total:    total,
	}, nil
}

// 本地缓存操作封装
func getFromLocalCache(cache *freecache.Cache, key string, dest interface{}) (bool, error) {
	data, err := cache.Get([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
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

func setToLocalCache(cache *freecache.Cache, key string, value interface{}, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.Set([]byte(key), data, ttl)
}

// Redis 缓存操作
func (uc *CommentUsecase) getFromRedis(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := uc.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
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

// ListChildComments 获取子评论列表
func (uc *CommentUsecase) ListChildComments(ctx context.Context, query *ListChildCommentsQuery) (*ListChildCommentsResult, error) {
	// 1. 参数验证
	if query.ParentID <= 0 {
		return nil, ErrInvalidParams
	}

	// 2. 验证父评论是否存在
	parent, err := uc.repo.GetByID(ctx, query.ParentID)
	if err != nil {
		return nil, err
	}
	if parent == nil || parent.IsDeleted {
		return nil, ErrParentCommentNotFound
	}

	// 3. 构建查询条件
	condition := map[string]interface{}{
		"parent_id": query.ParentID,
		"limit":     query.PageStats.PageSize,
		"offset":    (query.PageStats.Page - 1) * query.PageStats.PageSize,
		"order_by":  "created_at ASC",
	}

	// 4. 查询子评论
	comments, err := uc.repo.FindByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}

	// 5. 查询总数
	countCondition := map[string]interface{}{
		"parent_id": query.ParentID,
	}
	total, err := uc.repo.CountByCondition(ctx, countCondition)
	if err != nil {
		return nil, err
	}

	// 6. 获取点赞数统计
	commentIDs := make([]int64, len(comments))
	for i, comment := range comments {
		commentIDs[i] = comment.ID
	}
	likeCounts, err := uc.repo.GetLikeCounts(ctx, commentIDs)
	if err != nil {
		uc.log.Warnf("获取点赞数失败: %v", err)
	} else {
		for _, comment := range comments {
			if count, ok := likeCounts[comment.ID]; ok {
				comment.LikeCount = count
			}
		}
	}

	return &ListChildCommentsResult{
		Comments: comments,
		Total:    total,
	}, nil
}

// GetCommentByID 根据ID获取评论
func (uc *CommentUsecase) GetCommentByID(ctx context.Context, query *GetCommentQuery) (*GetCommentResult, error) {
	// 1. 参数验证
	if query.CommentID <= 0 {
		return nil, ErrInvalidParams
	}

	// 2. 查询评论
	comment, err := uc.repo.GetByID(ctx, query.CommentID)
	if err != nil {
		return nil, err
	}

	// 3. 如果评论已删除，返回特定错误
	if comment != nil && comment.IsDeleted {
		return nil, ErrCommentNotFound
	}

	return &GetCommentResult{
		Comment: comment,
	}, nil
}

// CountVideoComments 统计视频评论数
func (uc *CommentUsecase) CountVideoComments(ctx context.Context, query *CountVideoCommentsQuery) (*CountVideoCommentsResult, error) {
	if len(query.VideoIDs) == 0 {
		return &CountVideoCommentsResult{
			Counts: map[int64]int64{},
		}, nil
	}

	// 批量统计
	counts, err := uc.repo.CountByVideoIDs(ctx, query.VideoIDs)
	if err != nil {
		return nil, err
	}

	return &CountVideoCommentsResult{
		Counts: counts,
	}, nil
}

// CountUserComments 统计用户评论数
func (uc *CommentUsecase) CountUserComments(ctx context.Context, query *CountUserCommentsQuery) (*CountUserCommentsResult, error) {
	if len(query.UserIDs) == 0 {
		return &CountUserCommentsResult{
			Counts: map[int64]int64{},
		}, nil
	}

	// 批量统计
	counts, err := uc.repo.CountByUserIDs(ctx, query.UserIDs)
	if err != nil {
		return nil, err
	}

	return &CountUserCommentsResult{
		Counts: counts,
	}, nil
}

// loadChildComments 批量加载子评论
func (uc *CommentUsecase) loadChildComments(ctx context.Context, parentComments []*Comment) error {
	if len(parentComments) == 0 {
		return nil
	}

	// 1. 收集父评论ID
	parentIDs := make([]int64, len(parentComments))
	for i, comment := range parentComments {
		parentIDs[i] = comment.ID
	}

	// 2. 批量查询子评论数量
	childCounts, err := uc.repo.CountGroupByParentID(ctx, parentIDs)
	if err != nil {
		return err
	}

	// 3. 批量查询子评论（只查前5条）
	childComments, err := uc.repo.FindByCondition(ctx, map[string]interface{}{
		"parent_id":  parentIDs,
		"is_deleted": false, // 明确指定不查询已删除的
		"limit":      5,
		"order_by":   "created_at ASC",
	})
	if err != nil {
		return err
	}

	// 4. 获取子评论点赞数
	childCommentIDs := make([]int64, len(childComments))
	for i, child := range childComments {
		childCommentIDs[i] = child.ID
	}
	likeCounts, err := uc.repo.GetLikeCounts(ctx, childCommentIDs)
	if err != nil {
		uc.log.Warnf("获取子评论点赞数失败: %v", err)
	} else {
		for _, child := range childComments {
			if count, ok := likeCounts[child.ID]; ok {
				child.LikeCount = count
			}
		}
	}

	// 5. 按父评论ID分组
	childCommentsMap := make(map[int64][]*Comment)
	for _, child := range childComments {
		childCommentsMap[child.ParentID] = append(childCommentsMap[child.ParentID], child)
	}

	// 6. 组装数据
	for _, parent := range parentComments {
		if count, exists := childCounts[parent.ID]; exists {
			parent.ReplyCount = count
			parent.ChildCount = count
		}
		if children, exists := childCommentsMap[parent.ID]; exists {
			parent.ChildComments = children
		}
	}

	return nil
}

// 错误定义
var (
	ErrInvalidParams         = errors.New("invalid parameters")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrParentCommentNotFound = errors.New("parent comment not found")
	ErrNoPermission          = errors.New("no permission")
)
