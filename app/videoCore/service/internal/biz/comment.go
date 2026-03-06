package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lehu-video/app/videoCore/service/internal/pkg/idgen"
	"strconv"
	"time"

	"github.com/coocood/freecache"
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

// CommentRepo 接口（增加事务支持和冗余字段更新）
type CommentRepo interface {
	// 基础操作
	Create(ctx context.Context, comment *Comment) (int64, error)
	GetByID(ctx context.Context, id int64) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id int64, userID int64) error
	SoftDelete(ctx context.Context, id int64, userID int64) error
	BatchSoftDelete(ctx context.Context, ids []int64) error // 新增批量软删除

	// 条件查询
	FindByIDs(ctx context.Context, ids []int64) ([]*Comment, error)
	FindByCondition(ctx context.Context, condition map[string]interface{}) ([]*Comment, error)
	CountByCondition(ctx context.Context, condition map[string]interface{}) (int64, error)

	// 批量统计
	CountByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64]int64, error)
	CountByUserIDs(ctx context.Context, userIDs []int64) (map[int64]int64, error)

	// 冗余字段操作（事务内）
	IncrReplyCount(ctx context.Context, parentID int64, delta int) error
	GetReplyCount(ctx context.Context, commentID int64) (int64, error)

	// 事务支持
	ExecTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// CommentUsecase 业务逻辑
type CommentUsecase struct {
	repo         CommentRepo
	cache        *freecache.Cache
	redis        *redis.Client
	videoCounter VideoCounterRepo
	idGen        idgen.Generator
	log          *log.Helper
	sfg          singleflight.Group
}

func NewCommentUsecase(
	repo CommentRepo,
	cache *freecache.Cache,
	redis *redis.Client,
	videoCounter VideoCounterRepo,
	idGen idgen.Generator,
	logger log.Logger,
) *CommentUsecase {
	return &CommentUsecase{
		repo:         repo,
		cache:        cache,
		redis:        redis,
		videoCounter: videoCounter,
		idGen:        idGen,
		log:          log.NewHelper(logger),
	}
}

// ---------- 缓存键常量 ----------
const (
	// ZSet 键：video_comments:{videoId} 存储主评论ID，score为时间戳
	zsetVideoComments = "video_comments:%d"
	// ZSet 键：child_comments:{parentId} 存储子评论ID，score为时间戳
	zsetChildComments = "child_comments:%d"

	cacheKeyComment      = "comment:%d"      // 单个评论详情
	cacheKeyCommentList  = "comment_list:%s" // 批量评论列表（MD5(ids)）
	cacheKeyVideoCount   = "video_comment_count:%d"
	cacheKeyUserCount    = "user_comment_count:%d"
	cacheNullPlaceholder = "NULL"
)

// ---------- 创建评论 ----------
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

	// 使用事务执行数据库操作
	err := uc.repo.ExecTx(ctx, func(txCtx context.Context) error {
		// 1. 插入评论
		if _, err := uc.repo.Create(txCtx, comment); err != nil {
			return err
		}
		// 2. 如果是子评论，更新父评论的 reply_count
		if comment.ParentID > 0 {
			if err := uc.repo.IncrReplyCount(txCtx, comment.ParentID, 1); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 同步更新视频评论计数（不再异步，保证一致性）
	if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.VideoID, "comment_count", 1); err != nil {
		uc.log.Warnf("更新视频评论计数失败: videoId=%d, err=%v", cmd.VideoID, err)
		// 失败不影响主流程，记录日志即可（可考虑异步补偿）
	}

	// 更新 Redis ZSet
	pipe := uc.redis.Pipeline()
	if comment.ParentID == 0 {
		// 主评论，加入视频评论 ZSet
		zsetKey := fmt.Sprintf(zsetVideoComments, comment.VideoID)
		pipe.ZAdd(ctx, zsetKey, redis.Z{Score: float64(now.Unix()), Member: comment.ID})
		pipe.Expire(ctx, zsetKey, 7*24*time.Hour)
	} else {
		// 子评论，加入父评论的子评论 ZSet
		zsetKey := fmt.Sprintf(zsetChildComments, comment.ParentID)
		pipe.ZAdd(ctx, zsetKey, redis.Z{Score: float64(now.Unix()), Member: comment.ID})
		pipe.Expire(ctx, zsetKey, 7*24*time.Hour)
	}
	pipe.Del(ctx, fmt.Sprintf(cacheKeyComment, comment.ID)) // 确保不存在旧缓存
	_, _ = pipe.Exec(ctx)

	return &CreateCommentResult{Comment: comment}, nil
}

// ---------- 删除评论 ----------
func (uc *CommentUsecase) RemoveComment(ctx context.Context, cmd *RemoveCommentCommand) (*RemoveCommentResult, error) {
	// 查询评论（不加缓存，直接 DB）
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
	// 执行事务软删除（级联删除子评论）
	err = uc.repo.ExecTx(ctx, func(txCtx context.Context) error {
		// 1. 软删除当前评论
		if err := uc.repo.SoftDelete(txCtx, cmd.CommentID, cmd.UserID); err != nil {
			return err
		}

		// 2. 如果是主评论，批量软删除所有子评论
		if comment.ParentID == 0 {
			// 查询所有子评论ID（未删除的）
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
		} else {
			// 如果是子评论，减少父评论的 reply_count
			if err := uc.repo.IncrReplyCount(txCtx, comment.ParentID, -1); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 异步更新视频计数（减去 1 + 子评论数）
	go func() {
		bgCtx := context.Background()
		delta := int64(-1)
		if comment.ParentID == 0 {
			delta -= int64(len(subIDs))
		}
		if err := uc.videoCounter.IncrVideoCounter(bgCtx, comment.VideoID, "comment_count", delta); err != nil {
			uc.log.Warnf("更新视频评论计数失败: videoId=%d, err=%v", comment.VideoID, err)
		}
	}()

	// 删除缓存
	pipe := uc.redis.Pipeline()
	pipe.Del(ctx, fmt.Sprintf(cacheKeyComment, comment.ID))
	if comment.ParentID == 0 {
		// 删除视频主评论 ZSet
		pipe.Del(ctx, fmt.Sprintf(zsetVideoComments, comment.VideoID))
		// 删除子评论 ZSet
		pipe.Del(ctx, fmt.Sprintf(zsetChildComments, comment.ID))
		// 批量删除子评论详情缓存
		for _, subID := range subIDs {
			pipe.Del(ctx, fmt.Sprintf(cacheKeyComment, subID))
		}
	} else {
		// 删除子评论 ZSet
		pipe.Del(ctx, fmt.Sprintf(zsetChildComments, comment.ParentID))
	}
	_, _ = pipe.Exec(ctx)

	return &RemoveCommentResult{}, nil
}

// ---------- 获取视频评论列表（主评论，分页）----------
func (uc *CommentUsecase) ListVideoComments(ctx context.Context, query *ListVideoCommentsQuery) (*ListVideoCommentsResult, error) {
	if query.VideoID <= 0 {
		return nil, ErrInvalidParams
	}
	zsetKey := fmt.Sprintf(zsetVideoComments, query.VideoID)
	start := (query.PageStats.Page - 1) * query.PageStats.PageSize
	end := start + query.PageStats.PageSize - 1

	// 1. 尝试从 ZSet 获取 ID 列表
	ids, err := uc.redis.ZRevRange(ctx, zsetKey, int64(start), int64(end)).Result()
	if err == nil && len(ids) > 0 {
		// 命中缓存，获取总数并构建结果
		total, _ := uc.redis.ZCard(ctx, zsetKey).Result()
		return uc.buildVideoCommentsResult(ctx, ids, total)
	}

	// 2. 缓存未命中，使用 singleflight 合并回源请求
	sfKey := fmt.Sprintf("load_video_zset:%d", query.VideoID)
	v, err, _ := uc.sfg.Do(sfKey, func() (interface{}, error) {
		// 双重检查，防止在等待期间其他请求已回填
		ids, err := uc.redis.ZRevRange(ctx, zsetKey, 0, -1).Result()
		if err == nil && len(ids) > 0 {
			return ids, nil
		}

		// 从数据库加载所有主评论ID
		condition := map[string]interface{}{
			"video_id":   query.VideoID,
			"parent_id":  0,
			"is_deleted": false,
			"order_by":   "created_at DESC",
		}
		comments, err := uc.repo.FindByCondition(ctx, condition)
		if err != nil {
			return nil, err
		}
		if len(comments) == 0 {
			return []string{}, nil
		}

		// 回填 ZSet
		pipe := uc.redis.Pipeline()
		for _, c := range comments {
			pipe.ZAdd(ctx, zsetKey, redis.Z{Score: float64(c.CreateTime.Unix()), Member: c.ID})
		}
		pipe.Expire(ctx, zsetKey, 7*24*time.Hour)
		_, _ = pipe.Exec(ctx)

		// 返回所有ID（字符串形式）
		idStrs := make([]string, len(comments))
		for i, c := range comments {
			idStrs[i] = strconv.FormatInt(c.ID, 10)
		}
		return idStrs, nil
	})
	if err != nil {
		return nil, err
	}

	allIDs := v.([]string)
	total := int32(len(allIDs))
	startIdx := (query.PageStats.Page - 1) * query.PageStats.PageSize
	endIdx := startIdx + query.PageStats.PageSize
	if startIdx > total {
		return &ListVideoCommentsResult{Comments: []*Comment{}, Total: int64(total)}, nil
	}
	if endIdx > total {
		endIdx = total
	}
	pageIDs := allIDs[startIdx:endIdx]

	return uc.buildVideoCommentsResult(ctx, pageIDs, int64(total))
}

// 辅助方法：根据ID列表构建视频评论结果
func (uc *CommentUsecase) buildVideoCommentsResult(ctx context.Context, idStrs []string, total int64) (*ListVideoCommentsResult, error) {
	ids := make([]int64, len(idStrs))
	for i, s := range idStrs {
		ids[i], _ = strconv.ParseInt(s, 10, 64)
	}
	comments, err := uc.batchGetComments(ctx, ids)
	if err != nil {
		return nil, err
	}
	return &ListVideoCommentsResult{Comments: comments, Total: total}, nil
}

// ---------- 获取子评论列表 ----------
func (uc *CommentUsecase) ListChildComments(ctx context.Context, query *ListChildCommentsQuery) (*ListChildCommentsResult, error) {
	if query.ParentID <= 0 {
		return nil, ErrInvalidParams
	}
	zsetKey := fmt.Sprintf(zsetChildComments, query.ParentID)
	start := (query.PageStats.Page - 1) * query.PageStats.PageSize
	end := start + query.PageStats.PageSize - 1

	ids, err := uc.redis.ZRange(ctx, zsetKey, int64(start), int64(end)).Result() // 正序，按时间升序
	if err != nil && !errors.Is(err, redis.Nil) {
		uc.log.Warnf("从 ZSet 获取子评论ID失败: %v", err)
		return uc.queryChildCommentsFromDB(ctx, query)
	}

	var total int64
	if len(ids) > 0 {
		total, err = uc.redis.ZCard(ctx, zsetKey).Result()
		if err != nil {
			total = 0
		}
	} else {
		return uc.queryChildCommentsFromDB(ctx, query)
	}

	commentIDs := make([]int64, len(ids))
	for i, idStr := range ids {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		commentIDs[i] = id
	}
	comments, err := uc.batchGetComments(ctx, commentIDs)
	if err != nil {
		return nil, err
	}
	filtered := make([]*Comment, 0, len(comments))
	for _, c := range comments {
		if c != nil && !c.IsDeleted {
			filtered = append(filtered, c)
		}
	}
	return &ListChildCommentsResult{Comments: filtered, Total: total}, nil
}

func (uc *CommentUsecase) queryChildCommentsFromDB(ctx context.Context, query *ListChildCommentsQuery) (*ListChildCommentsResult, error) {
	condition := map[string]interface{}{
		"parent_id":  query.ParentID,
		"is_deleted": false,
		"limit":      query.PageStats.PageSize,
		"offset":     (query.PageStats.Page - 1) * query.PageStats.PageSize,
		"order_by":   "created_at ASC",
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

// ---------- 获取单个评论详情（带缓存 + SingleFlight）----------
func (uc *CommentUsecase) GetCommentByID(ctx context.Context, query *GetCommentQuery) (*GetCommentResult, error) {
	if query.CommentID <= 0 {
		return nil, ErrInvalidParams
	}
	cacheKey := fmt.Sprintf(cacheKeyComment, query.CommentID)

	// 本地缓存
	var comment Comment
	found, err := getFromLocalCache(uc.cache, cacheKey, &comment)
	if err == nil && found {
		if comment.IsDeleted {
			return nil, ErrCommentNotFound
		}
		return &GetCommentResult{Comment: &comment}, nil
	}

	// Redis 缓存
	found, err = uc.getFromRedis(ctx, cacheKey, &comment)
	if err == nil && found {
		if comment.IsDeleted {
			// 缓存空值
			return nil, ErrCommentNotFound
		}
		_ = setToLocalCache(uc.cache, cacheKey, &comment, 60)
		return &GetCommentResult{Comment: &comment}, nil
	}

	// 回源（SingleFlight）
	v, err, _ := uc.sfg.Do(cacheKey, func() (interface{}, error) {
		// 双重检查缓存
		found, err = uc.getFromRedis(ctx, cacheKey, &comment)
		if err == nil && found {
			return &comment, nil
		}
		dbComment, err := uc.repo.GetByID(ctx, query.CommentID)
		if err != nil {
			return nil, err
		}
		if dbComment == nil || dbComment.IsDeleted {
			// 缓存空值
			nullCache := Comment{ID: query.CommentID, IsDeleted: true}
			_ = uc.setToRedis(ctx, cacheKey, nullCache, 30)
			_ = setToLocalCache(uc.cache, cacheKey, nullCache, 5)
			return nil, ErrCommentNotFound
		}
		// 存入缓存
		_ = uc.setToRedis(ctx, cacheKey, dbComment, 3600)
		_ = setToLocalCache(uc.cache, cacheKey, dbComment, 60)
		return dbComment, nil
	})
	if err != nil {
		return nil, err
	}
	return &GetCommentResult{Comment: v.(*Comment)}, nil
}

// ---------- 批量获取评论详情（带缓存）----------
func (uc *CommentUsecase) batchGetComments(ctx context.Context, ids []int64) ([]*Comment, error) {
	if len(ids) == 0 {
		return []*Comment{}, nil
	}

	// 1. 批量从 Redis 获取
	pipe := uc.redis.Pipeline()
	cmds := make([]*redis.StringCmd, len(ids))
	for i, id := range ids {
		key := fmt.Sprintf(cacheKeyComment, id)
		cmds[i] = pipe.Get(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		uc.log.Warnf("批量获取评论缓存失败: %v", err)
	}

	// 2. 解析缓存结果，记录缺失索引
	result := make([]*Comment, len(ids))
	missIndices := []int{}
	for i, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == nil {
			var c Comment
			if json.Unmarshal(data, &c) == nil && !c.IsDeleted {
				result[i] = &c
				continue
			}
		}
		missIndices = append(missIndices, i)
	}

	// 3. 如有缺失，从数据库批量查询
	if len(missIndices) > 0 {
		missIDs := make([]int64, len(missIndices))
		for idx, pos := range missIndices {
			missIDs[idx] = ids[pos]
		}
		dbComments, err := uc.repo.FindByIDs(ctx, missIDs)
		if err != nil {
			return nil, err
		}
		dbMap := make(map[int64]*Comment)
		for _, c := range dbComments {
			dbMap[c.ID] = c
		}

		// 准备回写缓存的管道
		writePipe := uc.redis.Pipeline()
		for _, pos := range missIndices {
			id := ids[pos]
			if c, ok := dbMap[id]; ok && !c.IsDeleted {
				result[pos] = c
				key := fmt.Sprintf(cacheKeyComment, id)
				data, _ := json.Marshal(c)
				writePipe.Set(ctx, key, data, 3600*time.Second)
				_ = setToLocalCache(uc.cache, key, c, 60) // 本地缓存异步设置，不影响主流程
			} else {
				// 缓存空值
				nullCache := Comment{ID: id, IsDeleted: true}
				key := fmt.Sprintf(cacheKeyComment, id)
				data, _ := json.Marshal(nullCache)
				writePipe.Set(ctx, key, data, 30*time.Second)
				_ = setToLocalCache(uc.cache, key, nullCache, 5)
				result[pos] = nil
			}
		}
		// 执行批量写
		_, _ = writePipe.Exec(ctx)
	}

	// 4. 过滤已删除
	final := make([]*Comment, 0, len(result))
	for _, c := range result {
		if c != nil {
			final = append(final, c)
		}
	}
	return final, nil
}

// ---------- 统计接口（从 Redis 或 DB）----------
func (uc *CommentUsecase) CountVideoComments(ctx context.Context, query *CountVideoCommentsQuery) (*CountVideoCommentsResult, error) {
	if len(query.VideoIDs) == 0 {
		return &CountVideoCommentsResult{Counts: map[int64]int64{}}, nil
	}
	// 优先从 videoCounter 获取
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
	uc.log.Warnf("从 videoCounter 获取计数失败，降级查询数据库: %v", err)
	dbCounts, err := uc.repo.CountByVideoIDs(ctx, query.VideoIDs)
	if err != nil {
		return nil, err
	}
	return &CountVideoCommentsResult{Counts: dbCounts}, nil
}

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

// ---------- 缓存辅助函数 ----------
func getFromLocalCache(cache *freecache.Cache, key string, dest interface{}) (bool, error) {
	data, err := cache.Get([]byte(key))
	if err != nil {
		if errors.Is(err, freecache.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	err = json.Unmarshal(data, dest)
	return err == nil, err
}

func setToLocalCache(cache *freecache.Cache, key string, value interface{}, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.Set([]byte(key), data, ttl)
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
