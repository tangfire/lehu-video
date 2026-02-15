package biz

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// 点赞实体
type Favorite struct {
	Id           int64
	UserId       int64
	TargetType   int32 // 0: video, 1: comment
	TargetId     int64
	FavoriteType int32 // 0: like, 1: dislike
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsDeleted    bool
}

func (f *Favorite) SetId() {
	f.Id = int64(uuid.New().ID())
}

// 缓存键生成器
type CacheKeyGenerator interface {
	UserFavoriteKey(userId, targetId int64, targetType, favoriteType int32) string
	TargetCountKey(targetId int64, targetType, favoriteType int32) string
	UserTargetKey(userId, targetId int64, targetType int32) string
}

type defaultCacheKeyGenerator struct{}

func (g *defaultCacheKeyGenerator) UserFavoriteKey(userId, targetId int64, targetType, favoriteType int32) string {
	return fmt.Sprintf("fav:u%d:t%d:ty%d:ft%d", userId, targetId, targetType, favoriteType)
}

func (g *defaultCacheKeyGenerator) TargetCountKey(targetId int64, targetType, favoriteType int32) string {
	return fmt.Sprintf("fav:count:t%d:ty%d:ft%d", targetId, targetType, favoriteType)
}

func (g *defaultCacheKeyGenerator) UserTargetKey(userId, targetId int64, targetType int32) string {
	return fmt.Sprintf("fav:u%d:t%d:ty%d", userId, targetId, targetType)
}

// Command/Query/Result模式
type AddFavoriteCommand struct {
	UserId       int64
	TargetId     int64
	TargetType   int32
	FavoriteType int32
	ClientIP     string // 客户端IP，用于限流
	Timestamp    int64  // 时间戳
}

type AddFavoriteResult struct {
	AlreadyFavorited bool
	TotalCount       int64
	PreviousType     int32 // 之前的点赞类型
}

type RemoveFavoriteCommand struct {
	UserId       int64
	TargetId     int64
	TargetType   int32
	FavoriteType int32
}

type RemoveFavoriteResult struct {
	NotFavorited bool
	TotalCount   int64
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
	NeedDetail    bool // 是否需要详细数据（点赞/点踩分开）
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

// 简化的仓储接口
type FavoriteRepo interface {
	// 基础操作
	CreateFavorite(ctx context.Context, favorite *Favorite) error
	UpdateFavorite(ctx context.Context, favorite *Favorite) error
	GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (*Favorite, error)
	GetFavoriteByUserTarget(ctx context.Context, userId, targetId int64, targetType int32) (*Favorite, error)
	SoftDeleteFavorite(ctx context.Context, favoriteId int64) error
	HardDeleteFavorite(ctx context.Context, favoriteId int64) error

	// 查询操作
	ListFavorites(ctx context.Context, query *ListFavoriteQuery) ([]*Favorite, int64, error)
	CountFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (int64, error)
	CountFavoritesByTargetIds(ctx context.Context, targetIds []int64, targetType int32) (map[int64]FavoriteCount, error)
	CountFavoritesByUserIds(ctx context.Context, userIds []int64, targetType int32) (map[int64]FavoriteCount, error)
	GetFavoritesByUserAndTargets(ctx context.Context, userId int64, targetIds []int64, targetType int32) ([]*Favorite, error)
	BatchGetFavorites(ctx context.Context, userIds, targetIds []int64, targetType int32) ([]*Favorite, error)

	// 统计操作
	GetFavoriteStats(ctx context.Context, targetId int64, targetType int32) (*FavoriteStats, error)
	BatchGetFavoriteStats(ctx context.Context, targetIds []int64, targetType int32) (map[int64]*FavoriteStats, error)

	// 事务操作
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
	HotScore     float64 // 热度分数（可用于排序）
}

type FavoriteUsecase struct {
	repo        FavoriteRepo
	videoRepo   VideoRepo   // 用于获取视频作者ID
	counterRepo CounterRepo // 用于更新用户获赞数
	cache       *redis.Client
	log         *log.Helper
	limiter     *rate.Limiter
	keyGen      CacheKeyGenerator

	// 配置
	maxBatchSize     int
	favoriteCooldown time.Duration // 点赞冷却时间
	mutex            sync.RWMutex
	rateLimiters     map[string]*rate.Limiter // 用户级限流器
}

// NewFavoriteUsecase 需同时注入 videoRepo 和 counterRepo
func NewFavoriteUsecase(
	repo FavoriteRepo,
	videoRepo VideoRepo,
	counterRepo CounterRepo,
	cache *redis.Client,
	logger log.Logger,
) *FavoriteUsecase {
	return &FavoriteUsecase{
		repo:             repo,
		videoRepo:        videoRepo,
		counterRepo:      counterRepo,
		cache:            cache,
		log:              log.NewHelper(logger),
		limiter:          rate.NewLimiter(rate.Limit(1000), 100),
		keyGen:           &defaultCacheKeyGenerator{},
		maxBatchSize:     1000,
		favoriteCooldown: time.Second,
		rateLimiters:     make(map[string]*rate.Limiter),
	}
}

// 获取用户限流器
func (uc *FavoriteUsecase) getUserLimiter(userId int64) *rate.Limiter {
	key := fmt.Sprintf("user_%d", userId)

	uc.mutex.RLock()
	limiter, exists := uc.rateLimiters[key]
	uc.mutex.RUnlock()

	if !exists {
		uc.mutex.Lock()
		limiter, exists = uc.rateLimiters[key]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(10), 5) // 每用户10QPS，突发5
			uc.rateLimiters[key] = limiter
		}
		uc.mutex.Unlock()
	}

	return limiter
}

// 检查限流
func (uc *FavoriteUsecase) checkRateLimit(ctx context.Context, userId int64, clientIP string) error {
	// 全局限流
	if !uc.limiter.Allow() {
		return fmt.Errorf("系统繁忙，请稍后再试")
	}

	// 用户级限流
	userLimiter := uc.getUserLimiter(userId)
	if !userLimiter.Allow() {
		return fmt.Errorf("操作过于频繁，请稍后再试")
	}

	return nil
}

// AddFavorite 添加点赞/点踩
func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, cmd *AddFavoriteCommand) (*AddFavoriteResult, error) {
	if err := uc.validateCommand(cmd); err != nil {
		return nil, err
	}
	if err := uc.checkRateLimit(ctx, cmd.UserId, cmd.ClientIP); err != nil {
		return nil, err
	}

	var result *AddFavoriteResult
	var targetAuthorID int64 // 记录作者ID，用于后续更新获赞数
	var targetIsVideo bool   // 是否为视频点赞

	err := uc.repo.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := uc.repo.GetFavoriteByUserTarget(ctx, cmd.UserId, cmd.TargetId, cmd.TargetType)
		if err != nil {
			return fmt.Errorf("查询点赞状态失败: %w", err)
		}
		stats, err := uc.repo.GetFavoriteStats(ctx, cmd.TargetId, cmd.TargetType)
		if err != nil {
			return fmt.Errorf("获取统计数据失败: %w", err)
		}

		if existing != nil {
			if existing.FavoriteType == cmd.FavoriteType {
				// 相同类型：如果是已删除则恢复，否则已存在
				if existing.IsDeleted {
					existing.IsDeleted = false
					existing.UpdatedAt = time.Now()
					if err := uc.repo.UpdateFavorite(ctx, existing); err != nil {
						return fmt.Errorf("恢复点赞失败: %w", err)
					}
					// 恢复操作后，重新查询最新统计
					newStats, err := uc.repo.GetFavoriteStats(ctx, cmd.TargetId, cmd.TargetType)
					if err != nil {
						return fmt.Errorf("获取最新统计失败: %w", err)
					}
					stats = newStats
				}
				result = &AddFavoriteResult{
					AlreadyFavorited: true,
					TotalCount:       stats.TotalCount,
					PreviousType:     existing.FavoriteType,
				}
				return nil
			} else {
				// 切换点赞类型：先软删旧记录
				if !existing.IsDeleted {
					existing.IsDeleted = true
					existing.UpdatedAt = time.Now()
					if err := uc.repo.UpdateFavorite(ctx, existing); err != nil {
						return fmt.Errorf("取消原有操作失败: %w", err)
					}
					// 更新原有类型统计（后续会重新查询）
				}
			}
		}

		// 创建新记录
		favorite := &Favorite{
			UserId:       cmd.UserId,
			TargetId:     cmd.TargetId,
			TargetType:   cmd.TargetType,
			FavoriteType: cmd.FavoriteType,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		favorite.SetId()
		if err := uc.repo.CreateFavorite(ctx, favorite); err != nil {
			return fmt.Errorf("创建点赞记录失败: %w", err)
		}

		// 重新查询最新统计
		newStats, err := uc.repo.GetFavoriteStats(ctx, cmd.TargetId, cmd.TargetType)
		if err != nil {
			return fmt.Errorf("获取最新统计失败: %w", err)
		}
		stats = newStats

		result = &AddFavoriteResult{
			AlreadyFavorited: false,
			TotalCount:       stats.TotalCount,
			PreviousType:     -1,
		}

		// 如果是视频点赞，记录作者ID，用于事务成功后更新获赞数
		if cmd.TargetType == 0 && cmd.FavoriteType == 0 {
			targetIsVideo = true
			// 获取视频作者ID（这里可以尝试从缓存获取，但为了简单，直接查视频表，注意事务内可查）
			exist, video, err := uc.videoRepo.GetVideoById(ctx, cmd.TargetId)
			if err != nil {
				// 若获取作者失败，仅记录日志，不影响点赞主流程
				uc.log.Warnf("获取视频信息失败: videoId=%d, err=%v", cmd.TargetId, err)
			} else if exist && video != nil {
				targetAuthorID = video.Author.Id
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 事务成功后，异步更新作者总获赞数（最终一致，定时任务会兜底）
	if targetIsVideo && targetAuthorID > 0 {
		go func() {
			bgCtx := context.Background()
			// 增加作者获赞数
			if _, err := uc.counterRepo.IncrUserCounter(bgCtx, targetAuthorID, "total_favorited", 1); err != nil {
				uc.log.Errorf("异步更新作者 total_favorited 失败: userId=%d, err=%v", targetAuthorID, err)
			}
		}()
	}

	// 异步更新缓存
	go uc.updateCacheAsync(context.Background(), cmd, result.TotalCount)

	return result, nil
}

// RemoveFavorite 取消点赞/点踩
func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, cmd *RemoveFavoriteCommand) (*RemoveFavoriteResult, error) {
	if cmd.UserId <= 0 || cmd.TargetId <= 0 {
		return nil, fmt.Errorf("参数无效")
	}

	var result *RemoveFavoriteResult
	var targetAuthorID int64
	var targetIsVideo bool

	err := uc.repo.WithTransaction(ctx, func(ctx context.Context) error {
		favorite, err := uc.repo.GetFavorite(ctx, cmd.UserId, cmd.TargetId, cmd.TargetType, cmd.FavoriteType)
		if err != nil {
			return fmt.Errorf("查询点赞记录失败: %w", err)
		}
		if favorite == nil || favorite.IsDeleted {
			result = &RemoveFavoriteResult{
				NotFavorited: true,
				TotalCount:   0,
			}
			return nil
		}

		stats, err := uc.repo.GetFavoriteStats(ctx, cmd.TargetId, cmd.TargetType)
		if err != nil {
			return fmt.Errorf("获取统计数据失败: %w", err)
		}

		favorite.IsDeleted = true
		favorite.UpdatedAt = time.Now()
		if err := uc.repo.UpdateFavorite(ctx, favorite); err != nil {
			return fmt.Errorf("取消点赞失败: %w", err)
		}

		// 重新查询最新统计
		newStats, err := uc.repo.GetFavoriteStats(ctx, cmd.TargetId, cmd.TargetType)
		if err != nil {
			return fmt.Errorf("获取最新统计失败: %w", err)
		}
		stats = newStats

		result = &RemoveFavoriteResult{
			NotFavorited: false,
			TotalCount:   stats.TotalCount,
		}

		// 如果是视频点赞取消，记录作者ID
		if cmd.TargetType == 0 && cmd.FavoriteType == 0 {
			targetIsVideo = true
			exist, video, err := uc.videoRepo.GetVideoById(ctx, cmd.TargetId)
			if err != nil {
				uc.log.Warnf("获取视频信息失败: videoId=%d, err=%v", cmd.TargetId, err)
			} else if exist && video != nil {
				targetAuthorID = video.Author.Id
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 事务成功后，异步更新作者总获赞数（减1）
	if targetIsVideo && targetAuthorID > 0 {
		go func() {
			bgCtx := context.Background()
			if _, err := uc.counterRepo.IncrUserCounter(bgCtx, targetAuthorID, "total_favorited", -1); err != nil {
				uc.log.Errorf("异步更新作者 total_favorited 失败: userId=%d, err=%v", targetAuthorID, err)
			}
		}()
	}

	// 异步更新缓存
	go uc.updateCacheAsync(context.Background(), &AddFavoriteCommand{
		UserId:       cmd.UserId,
		TargetId:     cmd.TargetId,
		TargetType:   cmd.TargetType,
		FavoriteType: cmd.FavoriteType,
	}, result.TotalCount)

	return result, nil
}

// ListFavorite 查询点赞列表
func (uc *FavoriteUsecase) ListFavorite(ctx context.Context, query *ListFavoriteQuery) (*ListFavoriteResult, error) {
	// 参数验证
	if query.Id <= 0 {
		return nil, fmt.Errorf("查询ID无效")
	}

	// 分页参数
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}

	// 查询数据
	favorites, total, err := uc.repo.ListFavorites(ctx, query)
	if err != nil {
		uc.log.Errorf("查询点赞列表失败: aggregateType=%d, id=%d, err=%v", query.AggregateType, query.Id, err)
		return nil, fmt.Errorf("查询点赞列表失败: %w", err)
	}

	// 提取目标ID
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

// CountFavorite 统计点赞数量
func (uc *FavoriteUsecase) CountFavorite(ctx context.Context, query *CountFavoriteQuery) (*CountFavoriteResult, error) {
	if len(query.Ids) == 0 {
		return &CountFavoriteResult{Items: []CountFavoriteResultItem{}}, nil
	}

	if len(query.Ids) > uc.maxBatchSize {
		return nil, fmt.Errorf("ID列表过长，最多支持%d个ID", uc.maxBatchSize)
	}

	var resultMap map[int64]FavoriteCount
	var err error

	// 根据聚合类型查询
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

	// 构建结果
	items := make([]CountFavoriteResultItem, 0, len(resultMap))
	for id, counts := range resultMap {
		items = append(items, CountFavoriteResultItem{
			BizId:        id,
			LikeCount:    counts.LikeCount,
			DislikeCount: counts.DislikeCount,
			TotalCount:   counts.TotalCount,
		})
	}

	return &CountFavoriteResult{
		Items: items,
	}, nil
}

// IsFavorite 查询单个点赞状态
func (uc *FavoriteUsecase) IsFavorite(ctx context.Context, query *IsFavoriteQuery) (*IsFavoriteResult, error) {
	// 参数验证
	if query.UserId <= 0 || query.TargetId <= 0 {
		return nil, fmt.Errorf("参数无效")
	}

	// 尝试从缓存读取
	cacheKey := uc.keyGen.UserTargetKey(query.UserId, query.TargetId, query.TargetType)
	if uc.cache != nil {
		if _, err := uc.cache.Get(ctx, cacheKey).Result(); err == nil {
			// 解析缓存数据（简化处理，实际应反序列化）
			// 这里暂时不实现，仅作为示例
		}
	}

	// 查询数据库
	favorite, err := uc.repo.GetFavoriteByUserTarget(ctx, query.UserId, query.TargetId, query.TargetType)
	if err != nil {
		return nil, fmt.Errorf("查询点赞状态失败: %w", err)
	}

	// 获取统计数据
	stats, err := uc.repo.GetFavoriteStats(ctx, query.TargetId, query.TargetType)
	if err != nil {
		stats = &FavoriteStats{
			TargetId:     query.TargetId,
			TargetType:   query.TargetType,
			LikeCount:    0,
			DislikeCount: 0,
			TotalCount:   0,
		}
	}

	isFavorite := false
	favType := int32(-1)

	if favorite != nil && !favorite.IsDeleted {
		isFavorite = true
		favType = favorite.FavoriteType
	}

	result := &IsFavoriteResult{
		IsFavorite:    isFavorite,
		FavoriteType:  favType,
		TotalLikes:    stats.LikeCount,
		TotalDislikes: stats.DislikeCount,
	}

	// 更新缓存（异步）
	if uc.cache != nil {
		go func() {
			// 设置过期时间，如1小时
			uc.cache.Set(ctx, cacheKey, fmt.Sprintf("%v", result), time.Hour)
		}()
	}

	return result, nil
}

// BatchIsFavorite 批量查询点赞状态
func (uc *FavoriteUsecase) BatchIsFavorite(ctx context.Context, query *BatchIsFavoriteQuery) (*BatchIsFavoriteResult, error) {
	if len(query.TargetIds) == 0 || len(query.UserIds) == 0 {
		return &BatchIsFavoriteResult{Items: []BatchIsFavoriteResultItem{}}, nil
	}

	if len(query.TargetIds) > uc.maxBatchSize || len(query.UserIds) > uc.maxBatchSize {
		return nil, fmt.Errorf("批量查询数量过大，最多支持%d个", uc.maxBatchSize)
	}

	// 批量查询
	favorites, err := uc.repo.BatchGetFavorites(ctx, query.UserIds, query.TargetIds, query.TargetType)
	if err != nil {
		return nil, fmt.Errorf("批量查询失败: %w", err)
	}

	// 获取统计数据
	statsMap, err := uc.repo.BatchGetFavoriteStats(ctx, query.TargetIds, query.TargetType)
	if err != nil {
		// 即使统计失败也继续
		uc.log.Warnf("批量获取统计失败: %v", err)
	}

	// 构建映射
	favoriteMap := make(map[string]*Favorite)
	for _, fav := range favorites {
		key := fmt.Sprintf("%d_%d", fav.UserId, fav.TargetId)
		favoriteMap[key] = fav
	}

	// 构建结果
	items := make([]BatchIsFavoriteResultItem, 0)
	for _, userId := range query.UserIds {
		for _, targetId := range query.TargetIds {
			key := fmt.Sprintf("%d_%d", userId, targetId)
			fav := favoriteMap[key]

			var isLiked, isDisliked bool
			if fav != nil && !fav.IsDeleted {
				if fav.FavoriteType == 0 {
					isLiked = true
				} else {
					isDisliked = true
				}
			}

			stats := statsMap[targetId]
			if stats == nil {
				stats = &FavoriteStats{}
			}

			items = append(items, BatchIsFavoriteResultItem{
				UserId:       userId,
				TargetId:     targetId,
				IsLiked:      isLiked,
				IsDisliked:   isDisliked,
				LikeCount:    stats.LikeCount,
				DislikeCount: stats.DislikeCount,
			})
		}
	}

	return &BatchIsFavoriteResult{
		Items: items,
	}, nil
}

// 辅助方法
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

func (uc *FavoriteUsecase) updateCacheAsync(ctx context.Context, cmd *AddFavoriteCommand, totalCount int64) {
	if uc.cache == nil {
		return
	}

	// 更新用户-目标缓存
	userTargetKey := uc.keyGen.UserTargetKey(cmd.UserId, cmd.TargetId, cmd.TargetType)
	uc.cache.Del(ctx, userTargetKey)

	// 更新统计缓存
	countKey := uc.keyGen.TargetCountKey(cmd.TargetId, cmd.TargetType, cmd.FavoriteType)
	uc.cache.Set(ctx, countKey, totalCount, time.Hour*24)
}
