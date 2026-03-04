package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lehu-video/app/videoCore/service/internal/pkg/idgen"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// FavoriteEvent 点赞/点踩事件
type FavoriteEvent struct {
	UserId       int64 `json:"user_id"`
	TargetId     int64 `json:"target_id"`
	TargetType   int32 `json:"target_type"`   // 0:视频 1:评论
	FavoriteType int32 `json:"favorite_type"` // 0:点赞 1:点踩
	Action       int32 `json:"action"`        // 1:添加 -1:取消
	Timestamp    int64 `json:"timestamp"`
}

// ----------------------------- 类型定义 ---------------------------------
type Favorite struct {
	Id           int64
	UserId       int64
	TargetType   int32 // 0: video, 1: comment
	TargetId     int64
	FavoriteType int32 // 0: like, 1: dislike
	DeleteAt     int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CacheKeyGenerator interface {
	UserTargetKey(userId, targetId int64, targetType int32) string
	TargetCountKey(targetId int64, targetType, favoriteType int32) string
}

type defaultCacheKeyGenerator struct{}

func (g *defaultCacheKeyGenerator) UserTargetKey(userId, targetId int64, targetType int32) string {
	return fmt.Sprintf("fav:u%d:t%d:ty%d", userId, targetId, targetType)
}

func (g *defaultCacheKeyGenerator) TargetCountKey(targetId int64, targetType, favoriteType int32) string {
	return fmt.Sprintf("fav:count:t%d:ty%d:ft%d", targetId, targetType, favoriteType)
}

type AddFavoriteCommand struct {
	UserId       int64
	TargetId     int64
	TargetType   int32
	FavoriteType int32
	ClientIP     string
	Timestamp    int64
}

type AddFavoriteResult struct {
	AlreadyFavorited bool
	TotalCount       int64
	PreviousType     int32
}

type RemoveFavoriteCommand struct {
	UserId       int64
	TargetId     int64
	TargetType   int32
	FavoriteType int32
}

type ListFavoriteQuery struct {
	Id             int64
	AggregateType  int32
	FavoriteType   int32
	PageStats      PageStats
	IncludeDeleted bool
}

type ListFavoriteResult struct {
	TargetIds  []int64
	Total      int64
	TotalCount int64
}

type CountFavoriteQuery struct {
	Ids           []int64
	AggregateType int32
	FavoriteType  int32
	NeedDetail    bool
}

type CountFavoriteResultItem struct {
	BizId        int64
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
}

type CountFavoriteResult struct {
	Items []CountFavoriteResultItem
}

type IsFavoriteQuery struct {
	UserId       int64
	TargetId     int64
	TargetType   int32
	FavoriteType int32
}

type IsFavoriteResult struct {
	IsFavorite    bool
	FavoriteType  int32
	TotalLikes    int64
	TotalDislikes int64
}

type BatchIsFavoriteQuery struct {
	UserIds    []int64
	TargetIds  []int64
	TargetType int32
}

type BatchIsFavoriteResultItem struct {
	UserId       int64
	TargetId     int64
	IsLiked      bool
	IsDisliked   bool
	LikeCount    int64
	DislikeCount int64
}

type BatchIsFavoriteResult struct {
	Items []BatchIsFavoriteResultItem
}

type FavoriteRepo interface {
	CreateFavorite(ctx context.Context, favorite *Favorite) error
	UpdateFavorite(ctx context.Context, favorite *Favorite) error
	GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (*Favorite, error)
	GetFavoriteByUserTarget(ctx context.Context, userId, targetId int64, targetType int32) (*Favorite, error)
	GetFavoriteIncludeDeleted(ctx context.Context, userId, targetId int64, targetType int32) (*Favorite, error)
	SoftDeleteFavorite(ctx context.Context, favoriteId int64) error
	HardDeleteFavorite(ctx context.Context, favoriteId int64) error

	ListFavorites(ctx context.Context, query *ListFavoriteQuery) ([]*Favorite, int64, error)
	CountFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (int64, error)
	CountFavoritesByTargetIds(ctx context.Context, targetIds []int64, targetType int32) (map[int64]FavoriteCount, error)
	CountFavoritesByUserIds(ctx context.Context, userIds []int64, targetType int32) (map[int64]FavoriteCount, error)
	GetFavoritesByUserAndTargets(ctx context.Context, userId int64, targetIds []int64, targetType int32) ([]*Favorite, error)
	BatchGetFavorites(ctx context.Context, userIds, targetIds []int64, targetType int32) ([]*Favorite, error)

	GetFavoriteStats(ctx context.Context, targetId int64, targetType int32) (*FavoriteStats, error)
	BatchGetFavoriteStats(ctx context.Context, targetIds []int64, targetType int32) (map[int64]*FavoriteStats, error)

	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type FavoriteCount struct {
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
}

type FavoriteStats struct {
	TargetId     int64
	TargetType   int32
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
	HotScore     float64
}

// ----------------------------- FavoriteUsecase ---------------------------------
type FavoriteUsecase struct {
	repo             FavoriteRepo
	videoRepo        VideoRepo
	userCounter      UserCounterRepo
	videoCounter     VideoCounterRepo
	cache            *redis.Client
	log              *log.Helper
	limiter          *rate.Limiter
	keyGen           CacheKeyGenerator
	idGen            idgen.Generator
	maxBatchSize     int
	favoriteCooldown time.Duration
	mutex            sync.RWMutex
	rateLimiters     map[string]*rate.Limiter
	kafkaProducer    KafkaProducer
	favoriteTopic    string
}

func NewFavoriteUsecase(
	repo FavoriteRepo,
	videoRepo VideoRepo,
	userCounter UserCounterRepo,
	videoCounter VideoCounterRepo,
	cache *redis.Client,
	idGen idgen.Generator,
	kafkaProducer KafkaProducer,
	logger log.Logger,
) *FavoriteUsecase {
	return &FavoriteUsecase{
		repo:             repo,
		videoRepo:        videoRepo,
		userCounter:      userCounter,
		videoCounter:     videoCounter,
		cache:            cache,
		idGen:            idGen,
		log:              log.NewHelper(logger),
		limiter:          rate.NewLimiter(rate.Limit(1000), 100),
		keyGen:           &defaultCacheKeyGenerator{},
		maxBatchSize:     1000,
		favoriteCooldown: time.Second,
		rateLimiters:     make(map[string]*rate.Limiter),
		kafkaProducer:    kafkaProducer,
		favoriteTopic:    "favorite_topic",
	}
}

func (uc *FavoriteUsecase) getUserLimiter(userId int64) *rate.Limiter {
	key := fmt.Sprintf("user_%d", userId)
	uc.mutex.RLock()
	limiter, exists := uc.rateLimiters[key]
	uc.mutex.RUnlock()
	if !exists {
		uc.mutex.Lock()
		limiter, exists = uc.rateLimiters[key]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(10), 5)
			uc.rateLimiters[key] = limiter
		}
		uc.mutex.Unlock()
	}
	return limiter
}

func (uc *FavoriteUsecase) checkRateLimit(ctx context.Context, userId int64, clientIP string) error {
	if !uc.limiter.Allow() {
		return fmt.Errorf("系统繁忙，请稍后再试")
	}
	userLimiter := uc.getUserLimiter(userId)
	if !userLimiter.Allow() {
		return fmt.Errorf("操作过于频繁，请稍后再试")
	}
	return nil
}

func (uc *FavoriteUsecase) validateCommand(cmd *AddFavoriteCommand) error {
	if cmd.UserId <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if cmd.TargetId <= 0 {
		return fmt.Errorf("目标ID无效")
	}
	if cmd.TargetType != 0 && cmd.TargetType != 1 {
		return fmt.Errorf("目标类型无效")
	}
	if cmd.FavoriteType != 0 && cmd.FavoriteType != 1 {
		return fmt.Errorf("点赞类型无效")
	}
	return nil
}

// AddFavorite 添加点赞/点踩
func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, cmd *AddFavoriteCommand) error {
	if err := uc.validateCommand(cmd); err != nil {
		return err
	}
	if err := uc.checkRateLimit(ctx, cmd.UserId, cmd.ClientIP); err != nil {
		return err
	}

	// 1. 更新Redis计数（前端即时反馈）
	if cmd.TargetType == 0 { // 视频
		switch cmd.FavoriteType {
		case 0: // 点赞
			if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.TargetId, "like_count", 1); err != nil {
				uc.log.Warnf("更新视频点赞计数失败: %v", err)
			}
			// 获取作者ID并增加其 be_liked_count
			exist, video, err := uc.videoRepo.GetVideoById(ctx, cmd.TargetId)
			if err != nil {
				uc.log.Warnf("获取视频信息失败: %v", err)
			} else if exist {
				authorId := video.Author.Id
				if _, err := uc.userCounter.IncrUserCounter(ctx, authorId, "be_liked_count", 1); err != nil {
					uc.log.Warnf("更新作者获赞数失败: %v", err)
				}
			}
		case 1: // 点踩
			if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.TargetId, "dislike_count", 1); err != nil {
				uc.log.Warnf("更新视频点踩计数失败: %v", err)
			}
			// 点踩不计入作者获赞数，无需更新
		}
	} else if cmd.TargetType == 1 { // 评论，类似处理（如有需要）
		// 可根据业务补充评论点赞逻辑
	}

	// 2. 发送Kafka事件（持久化明细）
	event := FavoriteEvent{
		UserId:       cmd.UserId,
		TargetId:     cmd.TargetId,
		TargetType:   cmd.TargetType,
		FavoriteType: cmd.FavoriteType,
		Action:       1,
		Timestamp:    time.Now().Unix(),
	}
	data, _ := json.Marshal(event)
	key := fmt.Sprintf("%d:%d:%d", cmd.UserId, cmd.TargetId, cmd.TargetType)
	if err := uc.kafkaProducer.SendMessage(uc.favoriteTopic, []byte(key), data); err != nil {
		uc.log.Errorf("发送点赞事件失败: %v", err)
		return fmt.Errorf("操作失败，请稍后重试")
	}

	// 3. 主动更新缓存（确保后续查询立即正确）
	if uc.cache != nil {
		// 获取最新计数（用于缓存）
		var likeCount, dislikeCount int64
		if cmd.TargetType == 0 {
			counters, _ := uc.videoCounter.GetVideoCounters(ctx, cmd.TargetId, "like_count", "dislike_count")
			likeCount = counters["like_count"]
			dislikeCount = counters["dislike_count"]
		}

		result := &IsFavoriteResult{
			IsFavorite:    true,
			FavoriteType:  cmd.FavoriteType,
			TotalLikes:    likeCount,
			TotalDislikes: dislikeCount,
		}
		if data, err := json.Marshal(result); err == nil {
			key := uc.keyGen.UserTargetKey(cmd.UserId, cmd.TargetId, cmd.TargetType)
			if err := uc.cache.Set(ctx, key, string(data), time.Hour).Err(); err != nil {
				uc.log.Warnf("写入用户状态缓存失败: %v", err)
			}
		}
	}
	return nil
}

// RemoveFavorite 取消点赞/点踩
func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, cmd *RemoveFavoriteCommand) error {
	if cmd.UserId <= 0 || cmd.TargetId <= 0 {
		return fmt.Errorf("参数无效")
	}

	// 1. 更新Redis计数
	if cmd.TargetType == 0 {
		switch cmd.FavoriteType {
		case 0: // 取消点赞
			if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.TargetId, "like_count", -1); err != nil {
				uc.log.Warnf("更新视频点赞计数失败: %v", err)
			}
			exist, video, err := uc.videoRepo.GetVideoById(ctx, cmd.TargetId)
			if err != nil {
				uc.log.Warnf("获取视频信息失败: %v", err)
			} else if exist {
				authorId := video.Author.Id
				if _, err := uc.userCounter.IncrUserCounter(ctx, authorId, "be_liked_count", -1); err != nil {
					uc.log.Warnf("更新作者获赞数失败: %v", err)
				}
			}
		case 1: // 取消点踩
			if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.TargetId, "dislike_count", -1); err != nil {
				uc.log.Warnf("更新视频点踩计数失败: %v", err)
			}
		}
	}

	// 2. 发送Kafka事件
	event := FavoriteEvent{
		UserId:       cmd.UserId,
		TargetId:     cmd.TargetId,
		TargetType:   cmd.TargetType,
		FavoriteType: cmd.FavoriteType,
		Action:       -1,
		Timestamp:    time.Now().Unix(),
	}
	data, _ := json.Marshal(event)
	key := fmt.Sprintf("%d:%d:%d", cmd.UserId, cmd.TargetId, cmd.TargetType)
	if err := uc.kafkaProducer.SendMessage(uc.favoriteTopic, []byte(key), data); err != nil {
		uc.log.Errorf("发送取消点赞事件失败: %v", err)
		return fmt.Errorf("操作失败，请稍后重试")
	}

	// 3. 主动更新缓存（覆盖）
	if uc.cache != nil {
		var likeCount, dislikeCount int64
		if cmd.TargetType == 0 {
			counters, _ := uc.videoCounter.GetVideoCounters(ctx, cmd.TargetId, "like_count", "dislike_count")
			likeCount = counters["like_count"]
			dislikeCount = counters["dislike_count"]
		}

		result := &IsFavoriteResult{
			IsFavorite:    false,
			FavoriteType:  -1,
			TotalLikes:    likeCount,
			TotalDislikes: dislikeCount,
		}
		if data, err := json.Marshal(result); err == nil {
			key := uc.keyGen.UserTargetKey(cmd.UserId, cmd.TargetId, cmd.TargetType)
			// 短过期，覆盖 Kafka 批处理延迟
			if err := uc.cache.Set(ctx, key, string(data), 5*time.Second).Err(); err != nil {
				uc.log.Warnf("写入用户状态缓存失败: %v", err)
			}
		}
	}
	return nil
}

// ListFavorite 保持不变
func (uc *FavoriteUsecase) ListFavorite(ctx context.Context, query *ListFavoriteQuery) (*ListFavoriteResult, error) {
	// ... 原有代码不变 ...
	if query.Id <= 0 {
		return nil, fmt.Errorf("查询ID无效")
	}
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}

	favorites, total, err := uc.repo.ListFavorites(ctx, query)
	if err != nil {
		uc.log.Errorf("查询点赞列表失败: aggregateType=%d, id=%d, err=%v", query.AggregateType, query.Id, err)
		return nil, fmt.Errorf("查询点赞列表失败: %w", err)
	}

	targetIds := make([]int64, 0, len(favorites))
	for _, fav := range favorites {
		targetIds = append(targetIds, fav.TargetId)
	}

	return &ListFavoriteResult{
		TargetIds:  targetIds,
		Total:      int64(len(favorites)),
		TotalCount: total,
	}, nil
}

// CountFavorite 保持不变
func (uc *FavoriteUsecase) CountFavorite(ctx context.Context, query *CountFavoriteQuery) (*CountFavoriteResult, error) {
	if len(query.Ids) == 0 {
		return &CountFavoriteResult{Items: []CountFavoriteResultItem{}}, nil
	}
	if len(query.Ids) > uc.maxBatchSize {
		return nil, fmt.Errorf("ID列表过长，最多支持%d个ID", uc.maxBatchSize)
	}

	var resultMap map[int64]FavoriteCount
	var err error
	switch query.AggregateType {
	case 0: // BY_VIDEO
		resultMap, err = uc.repo.CountFavoritesByTargetIds(ctx, query.Ids, 0)
	case 1: // BY_COMMENT
		resultMap, err = uc.repo.CountFavoritesByTargetIds(ctx, query.Ids, 1)
	case 2: // BY_USER
		resultMap, err = uc.repo.CountFavoritesByUserIds(ctx, query.Ids, 0)
	default:
		return nil, fmt.Errorf("聚合类型无效: %d", query.AggregateType)
	}
	if err != nil {
		uc.log.Errorf("统计点赞数量失败: aggregateType=%d, ids=%v, err=%v", query.AggregateType, query.Ids, err)
		return nil, fmt.Errorf("统计点赞数量失败: %w", err)
	}

	items := make([]CountFavoriteResultItem, 0, len(resultMap))
	for id, counts := range resultMap {
		items = append(items, CountFavoriteResultItem{
			BizId:        id,
			LikeCount:    counts.LikeCount,
			DislikeCount: counts.DislikeCount,
			TotalCount:   counts.TotalCount,
		})
	}
	return &CountFavoriteResult{Items: items}, nil
}

// IsFavorite 查询单个点赞状态（优先读缓存，计数从Redis实时获取）
func (uc *FavoriteUsecase) IsFavorite(ctx context.Context, query *IsFavoriteQuery) (*IsFavoriteResult, error) {
	if query.UserId <= 0 || query.TargetId <= 0 {
		return nil, fmt.Errorf("参数无效")
	}

	// 1. 尝试从缓存读取
	if uc.cache != nil {
		key := uc.keyGen.UserTargetKey(query.UserId, query.TargetId, query.TargetType)
		cached, err := uc.cache.Get(ctx, key).Result()
		if err == nil && cached != "" {
			var result IsFavoriteResult
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				// 从 Redis 获取最新计数覆盖缓存中的计数（可选，保持实时）
				if query.TargetType == 0 {
					if counters, err := uc.videoCounter.GetVideoCounters(ctx, query.TargetId, "like_count", "dislike_count"); err == nil {
						result.TotalLikes = counters["like_count"]
						result.TotalDislikes = counters["dislike_count"]
					}
				}
				return &result, nil
			}
			uc.log.Warnf("缓存反序列化失败: key=%s, err=%v", key, err)
		} else if err != nil && !errors.Is(err, redis.Nil) {
			uc.log.Warnf("缓存读取失败: key=%s, err=%v", key, err)
		}
	}

	// 2. 缓存未命中，查询数据库
	favorite, err := uc.repo.GetFavoriteByUserTarget(ctx, query.UserId, query.TargetId, query.TargetType)
	if err != nil {
		return nil, fmt.Errorf("查询点赞状态失败: %w", err)
	}

	// 3. 从Redis获取实时计数
	var likeCount, dislikeCount int64
	if query.TargetType == 0 {
		if counters, err := uc.videoCounter.GetVideoCounters(ctx, query.TargetId, "like_count", "dislike_count"); err == nil {
			likeCount = counters["like_count"]
			dislikeCount = counters["dislike_count"]
		}
	}

	isFavorite := favorite != nil && favorite.DeleteAt == 0
	favType := int32(-1)
	if isFavorite {
		favType = favorite.FavoriteType
	}

	result := &IsFavoriteResult{
		IsFavorite:    isFavorite,
		FavoriteType:  favType,
		TotalLikes:    likeCount,
		TotalDislikes: dislikeCount,
	}

	// 4. 写入缓存（根据状态设置不同过期时间）
	if uc.cache != nil {
		if data, err := json.Marshal(result); err == nil {
			key := uc.keyGen.UserTargetKey(query.UserId, query.TargetId, query.TargetType)
			expiration := time.Hour
			if !result.IsFavorite {
				expiration = 5 * time.Second // 未点赞状态短过期，避免延迟影响
			}
			setCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := uc.cache.Set(setCtx, key, string(data), expiration).Err(); err != nil {
				uc.log.Warnf("缓存写入失败: key=%s, err=%v", key, err)
			}
		}
	}
	return result, nil
}

// BatchIsFavorite 批量查询点赞状态（优化版，计数从Redis批量获取）
func (uc *FavoriteUsecase) BatchIsFavorite(ctx context.Context, query *BatchIsFavoriteQuery) (*BatchIsFavoriteResult, error) {
	if len(query.TargetIds) == 0 || len(query.UserIds) == 0 {
		return &BatchIsFavoriteResult{Items: []BatchIsFavoriteResultItem{}}, nil
	}
	if len(query.TargetIds) > uc.maxBatchSize || len(query.UserIds) > uc.maxBatchSize {
		return nil, fmt.Errorf("批量查询数量过大，最多支持%d个", uc.maxBatchSize)
	}

	// 生成所有缓存键
	keys := make([]string, 0, len(query.UserIds)*len(query.TargetIds))
	keyToPair := make(map[string]struct{ userId, targetId int64 })
	for _, uid := range query.UserIds {
		for _, tid := range query.TargetIds {
			key := uc.keyGen.UserTargetKey(uid, tid, query.TargetType)
			keys = append(keys, key)
			keyToPair[key] = struct{ userId, targetId int64 }{uid, tid}
		}
	}

	// 1. 批量从缓存读取
	cachedResults := make(map[string]*IsFavoriteResult)
	if uc.cache != nil {
		vals, err := uc.cache.MGet(ctx, keys...).Result()
		if err != nil {
			uc.log.Warnf("批量缓存读取失败: %v", err)
		} else {
			for i, val := range vals {
				if val == nil {
					continue
				}
				key := keys[i]
				if str, ok := val.(string); ok && str != "" {
					var res IsFavoriteResult
					if err := json.Unmarshal([]byte(str), &res); err == nil {
						cachedResults[key] = &res
					} else {
						uc.log.Warnf("缓存反序列化失败: key=%s, err=%v", key, err)
					}
				}
			}
		}
	}

	// 2. 找出未命中的组合
	var missPairs []struct{ userId, targetId int64 }
	for _, key := range keys {
		if _, hit := cachedResults[key]; !hit {
			missPairs = append(missPairs, keyToPair[key])
		}
	}

	// 3. 批量查询数据库（未命中部分）
	dbResults := make(map[string]*IsFavoriteResult)
	if len(missPairs) > 0 {
		// 提取未命中的 userIds 和 targetIds（去重）
		missUserIds := make([]int64, 0, len(missPairs))
		missTargetIds := make([]int64, 0, len(missPairs))
		userSet := make(map[int64]bool)
		targetSet := make(map[int64]bool)
		for _, p := range missPairs {
			if !userSet[p.userId] {
				userSet[p.userId] = true
				missUserIds = append(missUserIds, p.userId)
			}
			if !targetSet[p.targetId] {
				targetSet[p.targetId] = true
				missTargetIds = append(missTargetIds, p.targetId)
			}
		}

		// 批量查询点赞记录（只查有效记录）
		favorites, err := uc.repo.BatchGetFavorites(ctx, missUserIds, missTargetIds, query.TargetType)
		if err != nil {
			return nil, fmt.Errorf("批量查询失败: %w", err)
		}

		// 批量从Redis获取计数（替换原来的数据库统计）
		countersMap, err := uc.videoCounter.BatchGetVideoCounters(ctx, missTargetIds, "like_count", "dislike_count")
		if err != nil {
			uc.log.Warnf("批量获取视频计数器失败: %v", err)
		}

		// 构建 favorite 映射
		favMap := make(map[string]*Favorite)
		for _, f := range favorites {
			key := uc.keyGen.UserTargetKey(f.UserId, f.TargetId, query.TargetType)
			favMap[key] = f
		}

		// 为每个未命中组合构造结果
		for _, p := range missPairs {
			key := uc.keyGen.UserTargetKey(p.userId, p.targetId, query.TargetType)
			fav := favMap[key]

			isLiked := false
			isDisliked := false
			if fav != nil && fav.DeleteAt == 0 {
				if fav.FavoriteType == 0 {
					isLiked = true
				} else {
					isDisliked = true
				}
			}

			// 从 countersMap 获取计数
			likeCount := int64(0)
			dislikeCount := int64(0)
			if counters, ok := countersMap[p.targetId]; ok {
				likeCount = counters["like_count"]
				dislikeCount = counters["dislike_count"]
			}

			res := &IsFavoriteResult{
				IsFavorite:    isLiked || isDisliked,
				FavoriteType:  -1,
				TotalLikes:    likeCount,
				TotalDislikes: dislikeCount,
			}
			if isLiked {
				res.FavoriteType = 0
			} else if isDisliked {
				res.FavoriteType = 1
			}
			dbResults[key] = res
		}

		// 异步回填缓存（按状态设置过期时间）
		if uc.cache != nil {
			go func() {
				bgCtx := context.Background()
				for key, res := range dbResults {
					if data, err := json.Marshal(res); err == nil {
						expiration := time.Hour
						if !res.IsFavorite {
							expiration = 5 * time.Second
						}
						setCtx, cancel := context.WithTimeout(bgCtx, 100*time.Millisecond)
						if err := uc.cache.Set(setCtx, key, string(data), expiration).Err(); err != nil {
							uc.log.Warnf("缓存回填失败: key=%s, err=%v", key, err)
						}
						cancel()
					}
				}
			}()
		}
	}

	// 4. 组装最终结果（按原始顺序）
	items := make([]BatchIsFavoriteResultItem, 0, len(query.UserIds)*len(query.TargetIds))
	for _, uid := range query.UserIds {
		for _, tid := range query.TargetIds {
			key := uc.keyGen.UserTargetKey(uid, tid, query.TargetType)
			var res *IsFavoriteResult
			if cached, ok := cachedResults[key]; ok {
				res = cached
			} else {
				res = dbResults[key]
			}

			item := BatchIsFavoriteResultItem{
				UserId:       uid,
				TargetId:     tid,
				IsLiked:      false,
				IsDisliked:   false,
				LikeCount:    0,
				DislikeCount: 0,
			}
			if res != nil {
				item.LikeCount = res.TotalLikes
				item.DislikeCount = res.TotalDislikes
				if res.IsFavorite {
					if res.FavoriteType == 0 {
						item.IsLiked = true
					} else if res.FavoriteType == 1 {
						item.IsDisliked = true
					}
				}
			}
			items = append(items, item)
		}
	}

	return &BatchIsFavoriteResult{Items: items}, nil
}
