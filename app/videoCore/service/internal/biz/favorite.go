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
	// UserStatusKey 用户对某个目标的点赞状态缓存键
	UserStatusKey(userId, targetId int64, targetType int32) string
	// TargetCountKey 目标点赞/点踩计数缓存键
	TargetCountKey(targetId int64, targetType int32) string
}

type defaultCacheKeyGenerator struct{}

func (g *defaultCacheKeyGenerator) UserStatusKey(userId, targetId int64, targetType int32) string {
	return fmt.Sprintf("fav:status:u%d:t%d:ty%d", userId, targetId, targetType)
}

func (g *defaultCacheKeyGenerator) TargetCountKey(targetId int64, targetType int32) string {
	return fmt.Sprintf("fav:count:t%d:ty%d", targetId, targetType)
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

// IsFavoriteQuery 查询单个用户对目标的点赞状态
type IsFavoriteQuery struct {
	UserId     int64
	TargetId   int64
	TargetType int32
	// 不再需要 FavoriteType，因为查询的是用户对该目标的所有点赞状态
}

// IsFavoriteResult 只包含用户的点赞状态
type IsFavoriteResult struct {
	IsFavorite   bool  // 是否有点赞/点踩记录（有效）
	FavoriteType int32 // 如果 IsFavorite=true，则返回具体的类型（0点赞，1点踩）；否则为 -1
}

type BatchIsFavoriteQuery struct {
	UserIds    []int64
	TargetIds  []int64
	TargetType int32
}

type BatchIsFavoriteResultItem struct {
	UserId     int64
	TargetId   int64
	IsLiked    bool
	IsDisliked bool
	// 不再包含计数
}

type BatchIsFavoriteResult struct {
	Items []BatchIsFavoriteResultItem
}

type FavoriteRepo interface {
	CreateFavorite(ctx context.Context, favorite *Favorite) error
	UpdateFavorite(ctx context.Context, favorite *Favorite) error
	UpdateFavoriteIfNewer(ctx context.Context, favorite *Favorite) error // 新增：只有当 existing.UpdatedAt < favorite.UpdatedAt 时才更新
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
	rateLimiters     sync.Map
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
		repo:          repo,
		videoRepo:     videoRepo,
		userCounter:   userCounter,
		videoCounter:  videoCounter,
		cache:         cache,
		idGen:         idGen,
		log:           log.NewHelper(logger),
		limiter:       rate.NewLimiter(rate.Limit(20000), 5000),
		keyGen:        &defaultCacheKeyGenerator{},
		maxBatchSize:  1000,
		kafkaProducer: kafkaProducer,
		favoriteTopic: "favorite_topic",
	}
}

func (uc *FavoriteUsecase) getUserLimiter(userId int64) *rate.Limiter {
	key := fmt.Sprintf("user_%d", userId)
	val, ok := uc.rateLimiters.Load(key)
	if ok {
		return val.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rate.Limit(10), 5)
	uc.rateLimiters.Store(key, limiter)
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

// getCountField 根据点赞类型返回 Redis 计数字段名（目前仅支持视频）
func (uc *FavoriteUsecase) getCountField(favType int32) string {
	switch favType {
	case 0:
		return "like_count"
	case 1:
		return "dislike_count"
	default:
		return ""
	}
}

// AddFavorite 添加点赞/点踩（自动处理状态切换）
func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, cmd *AddFavoriteCommand) error {
	if err := uc.validateCommand(cmd); err != nil {
		return err
	}
	if err := uc.checkRateLimit(ctx, cmd.UserId, cmd.ClientIP); err != nil {
		return err
	}

	// 1. 查询当前状态
	current, err := uc.IsFavorite(ctx, &IsFavoriteQuery{
		UserId:     cmd.UserId,
		TargetId:   cmd.TargetId,
		TargetType: cmd.TargetType,
	})
	if err != nil {
		return err
	}
	// 如果已经是指定类型，直接返回成功（或可返回已存在错误）
	if current.IsFavorite && current.FavoriteType == cmd.FavoriteType {
		return nil // 幂等处理
	}

	// 2. 构建计数变更（只处理视频类型）
	deltas := make(map[int64]map[string]int64) // videoId -> field -> delta
	if cmd.TargetType == 0 {                   // 视频
		if current.IsFavorite {
			// 需要减少原类型的计数
			oldField := uc.getCountField(current.FavoriteType)
			if deltas[cmd.TargetId] == nil {
				deltas[cmd.TargetId] = make(map[string]int64)
			}
			deltas[cmd.TargetId][oldField] -= 1
		}
		// 增加新类型的计数
		newField := uc.getCountField(cmd.FavoriteType)
		if deltas[cmd.TargetId] == nil {
			deltas[cmd.TargetId] = make(map[string]int64)
		}
		deltas[cmd.TargetId][newField] += 1
	} // 其他类型可扩展

	// 3. 原子更新 Redis 计数
	if len(deltas) > 0 {
		if err := uc.videoCounter.BatchIncrFields(ctx, deltas); err != nil {
			uc.log.Errorf("批量更新视频计数失败: %v", err)
			return err
		}
	}

	// 4. 发送 Kafka 事件（Action=1 表示添加，消费者会处理切换逻辑）
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
	err = uc.kafkaProducer.SendMessage(uc.favoriteTopic, []byte(key), data)

	// 5. 处理发送失败：回滚计数
	if err != nil {
		uc.log.Errorf("发送点赞事件失败，准备回滚计数: %v, userId=%d, targetId=%d", err, cmd.UserId, cmd.TargetId)
		// 异步回滚计数（反向操作）
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			rollbackDeltas := make(map[int64]map[string]int64)
			if cmd.TargetType == 0 {
				// 反向：增加原类型（如果有），减少新类型
				rollback := make(map[string]int64)
				if current.IsFavorite {
					rollback[uc.getCountField(current.FavoriteType)] = 1
				}
				rollback[uc.getCountField(cmd.FavoriteType)] = -1
				rollbackDeltas[cmd.TargetId] = rollback
			}
			if err := uc.videoCounter.BatchIncrFields(bgCtx, rollbackDeltas); err != nil {
				uc.log.Errorf("回滚计数失败: %v", err)
			} else {
				uc.log.Infof("计数回滚完成: targetId=%d", cmd.TargetId)
			}
		}()
		return fmt.Errorf("操作失败，请稍后重试")
	}

	// 6. Kafka 发送成功，更新状态缓存
	if uc.cache != nil {
		statusResult := &IsFavoriteResult{
			IsFavorite:   true,
			FavoriteType: cmd.FavoriteType,
		}
		if data, err := json.Marshal(statusResult); err == nil {
			key := uc.keyGen.UserStatusKey(cmd.UserId, cmd.TargetId, cmd.TargetType)
			if err := uc.cache.Set(ctx, key, string(data), time.Hour).Err(); err != nil {
				uc.log.Warnf("写入用户状态缓存失败: %v", err)
			}
		}
	}
	return nil
}

// RemoveFavorite 取消点赞/点踩（检查当前状态）
func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, cmd *RemoveFavoriteCommand) error {
	if cmd.UserId <= 0 || cmd.TargetId <= 0 {
		return fmt.Errorf("参数无效")
	}
	if cmd.TargetType != 0 && cmd.TargetType != 1 {
		return fmt.Errorf("目标类型无效")
	}
	if cmd.FavoriteType != 0 && cmd.FavoriteType != 1 {
		return fmt.Errorf("点赞类型无效")
	}

	// 1. 查询当前状态
	current, err := uc.IsFavorite(ctx, &IsFavoriteQuery{
		UserId:     cmd.UserId,
		TargetId:   cmd.TargetId,
		TargetType: cmd.TargetType,
	})
	if err != nil {
		return err
	}
	if !current.IsFavorite || current.FavoriteType != cmd.FavoriteType {
		return fmt.Errorf("未找到对应的点赞/点踩记录")
	}

	// 2. 构建计数变更（只处理视频）
	deltas := make(map[int64]map[string]int64)
	if cmd.TargetType == 0 {
		field := uc.getCountField(cmd.FavoriteType)
		deltas[cmd.TargetId] = map[string]int64{field: -1}
	}

	// 3. 原子更新计数
	if len(deltas) > 0 {
		if err := uc.videoCounter.BatchIncrFields(ctx, deltas); err != nil {
			uc.log.Errorf("批量更新视频计数失败: %v", err)
			return err
		}
	}

	// 4. 发送 Kafka 事件
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
	err = uc.kafkaProducer.SendMessage(uc.favoriteTopic, []byte(key), data)

	// 5. 处理发送失败：回滚计数
	if err != nil {
		uc.log.Errorf("发送取消事件失败，准备回滚计数: %v, userId=%d, targetId=%d", err, cmd.UserId, cmd.TargetId)
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			rollback := map[int64]map[string]int64{
				cmd.TargetId: {uc.getCountField(cmd.FavoriteType): 1},
			}
			if err := uc.videoCounter.BatchIncrFields(bgCtx, rollback); err != nil {
				uc.log.Errorf("回滚计数失败: %v", err)
			} else {
				uc.log.Infof("计数回滚完成: targetId=%d", cmd.TargetId)
			}
		}()
		return fmt.Errorf("操作失败，请稍后重试")
	}

	// 6. 删除状态缓存
	if uc.cache != nil {
		key := uc.keyGen.UserStatusKey(cmd.UserId, cmd.TargetId, cmd.TargetType)
		if err := uc.cache.Del(ctx, key).Err(); err != nil {
			uc.log.Warnf("删除用户状态缓存失败: %v", err)
		}
	}
	return nil
}

// ListFavorite 保持不变
func (uc *FavoriteUsecase) ListFavorite(ctx context.Context, query *ListFavoriteQuery) (*ListFavoriteResult, error) {
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

// CountFavorite 批量统计点赞/点踩数量（优先从Redis读取视频计数）
func (uc *FavoriteUsecase) CountFavorite(ctx context.Context, query *CountFavoriteQuery) (*CountFavoriteResult, error) {
	if len(query.Ids) == 0 {
		return &CountFavoriteResult{Items: []CountFavoriteResultItem{}}, nil
	}
	if len(query.Ids) > uc.maxBatchSize {
		return nil, fmt.Errorf("ID列表过长，最多支持%d个ID", uc.maxBatchSize)
	}

	var items []CountFavoriteResultItem
	var err error

	switch query.AggregateType {
	case 0: // BY_VIDEO
		// 从 Redis 获取视频计数
		var countersMap map[int64]map[string]int64
		countersMap, err = uc.videoCounter.BatchGetVideoCounters(ctx, query.Ids, "like_count", "dislike_count")
		if err != nil {
			uc.log.Warnf("批量获取视频计数器失败: %v，回退到数据库", err)
			// 降级：查数据库
			var dbMap map[int64]FavoriteCount
			dbMap, err = uc.repo.CountFavoritesByTargetIds(ctx, query.Ids, 0)
			if err != nil {
				return nil, fmt.Errorf("统计视频点赞数量失败: %w", err)
			}
			for _, id := range query.Ids {
				counts := dbMap[id]
				items = append(items, CountFavoriteResultItem{
					BizId:        id,
					LikeCount:    counts.LikeCount,
					DislikeCount: counts.DislikeCount,
					TotalCount:   counts.TotalCount,
				})
			}
		} else {
			for _, id := range query.Ids {
				counters, ok := countersMap[id]
				if !ok || counters == nil {
					items = append(items, CountFavoriteResultItem{
						BizId:        id,
						LikeCount:    0,
						DislikeCount: 0,
						TotalCount:   0,
					})
				} else {
					like := counters["like_count"]
					dislike := counters["dislike_count"]
					items = append(items, CountFavoriteResultItem{
						BizId:        id,
						LikeCount:    like,
						DislikeCount: dislike,
						TotalCount:   like + dislike,
					})
				}
			}
		}
	case 1: // BY_COMMENT（评论，目前未在Redis维护，直接查数据库）
		var dbMap map[int64]FavoriteCount
		dbMap, err = uc.repo.CountFavoritesByTargetIds(ctx, query.Ids, 1)
		if err != nil {
			return nil, fmt.Errorf("统计评论点赞数量失败: %w", err)
		}
		for _, id := range query.Ids {
			counts := dbMap[id]
			items = append(items, CountFavoriteResultItem{
				BizId:        id,
				LikeCount:    counts.LikeCount,
				DislikeCount: counts.DislikeCount,
				TotalCount:   counts.TotalCount,
			})
		}
	case 2: // BY_USER（统计用户点赞的视频数，查数据库）
		var dbMap map[int64]FavoriteCount
		dbMap, err = uc.repo.CountFavoritesByUserIds(ctx, query.Ids, 0) // targetType 0 表示视频
		if err != nil {
			return nil, fmt.Errorf("统计用户点赞数量失败: %w", err)
		}
		for _, id := range query.Ids {
			counts := dbMap[id]
			items = append(items, CountFavoriteResultItem{
				BizId:        id,
				LikeCount:    counts.LikeCount,
				DislikeCount: counts.DislikeCount,
				TotalCount:   counts.TotalCount,
			})
		}
	default:
		return nil, fmt.Errorf("聚合类型无效: %d", query.AggregateType)
	}
	return &CountFavoriteResult{Items: items}, nil
}

// IsFavorite 查询单个点赞状态（优先读缓存，不再包含计数）
func (uc *FavoriteUsecase) IsFavorite(ctx context.Context, query *IsFavoriteQuery) (*IsFavoriteResult, error) {
	if query.UserId <= 0 || query.TargetId <= 0 {
		return nil, fmt.Errorf("参数无效")
	}

	// 1. 尝试从缓存读取
	if uc.cache != nil {
		key := uc.keyGen.UserStatusKey(query.UserId, query.TargetId, query.TargetType)
		cached, err := uc.cache.Get(ctx, key).Result()
		if err == nil && cached != "" {
			var result IsFavoriteResult
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
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

	isFavorite := favorite != nil && favorite.DeleteAt == 0
	favType := int32(-1)
	if isFavorite {
		favType = favorite.FavoriteType
	}

	result := &IsFavoriteResult{
		IsFavorite:   isFavorite,
		FavoriteType: favType,
	}

	// 3. 写入缓存（根据状态设置不同过期时间）
	if uc.cache != nil {
		if data, err := json.Marshal(result); err == nil {
			key := uc.keyGen.UserStatusKey(query.UserId, query.TargetId, query.TargetType)
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

// BatchIsFavorite 批量查询点赞状态（优化版，只返回状态，计数由 CountFavorite 处理）
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
			key := uc.keyGen.UserStatusKey(uid, tid, query.TargetType)
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

		// 构建 favorite 映射
		favMap := make(map[string]*Favorite)
		for _, f := range favorites {
			key := uc.keyGen.UserStatusKey(f.UserId, f.TargetId, query.TargetType)
			favMap[key] = f
		}

		// 为每个未命中组合构造结果
		for _, p := range missPairs {
			key := uc.keyGen.UserStatusKey(p.userId, p.targetId, query.TargetType)
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

			res := &IsFavoriteResult{
				IsFavorite:   isLiked || isDisliked,
				FavoriteType: -1,
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
			key := uc.keyGen.UserStatusKey(uid, tid, query.TargetType)
			var res *IsFavoriteResult
			if cached, ok := cachedResults[key]; ok {
				res = cached
			} else {
				res = dbResults[key]
			}

			item := BatchIsFavoriteResultItem{
				UserId:     uid,
				TargetId:   tid,
				IsLiked:    false,
				IsDisliked: false,
			}
			if res != nil && res.IsFavorite {
				if res.FavoriteType == 0 {
					item.IsLiked = true
				} else if res.FavoriteType == 1 {
					item.IsDisliked = true
				}
			}
			items = append(items, item)
		}
	}

	return &BatchIsFavoriteResult{Items: items}, nil
}
