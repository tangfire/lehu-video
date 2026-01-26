package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"time"
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

// ✅ 使用Command/Query/Result模式
type AddFavoriteCommand struct {
	UserId       int64
	TargetId     int64
	TargetType   int32 // 0: video, 1: comment
	FavoriteType int32 // 0: like, 1: dislike
}

type AddFavoriteResult struct{}

type RemoveFavoriteCommand struct {
	UserId       int64
	TargetId     int64
	TargetType   int32 // 0: video, 1: comment
	FavoriteType int32 // 0: like, 1: dislike
}

type RemoveFavoriteResult struct{}

type ListFavoriteQuery struct {
	Id            int64
	AggregateType int32 // 0: by_video, 1: by_comment, 2: by_user
	FavoriteType  int32 // 0: like, 1: dislike
	PageStats     PageStats
}

type ListFavoriteResult struct {
	TargetIds []int64
	Total     int64
}

type CountFavoriteQuery struct {
	Ids           []int64
	AggregateType int32 // 0: by_video, 1: by_comment, 2: by_user
	FavoriteType  int32 // 0: like, 1: dislike
}

type CountFavoriteResultItem struct {
	BizId int64
	Count int64
}

type CountFavoriteResult struct {
	Items []CountFavoriteResultItem
}

type IsFavoriteQueryItem struct {
	BizId  int64
	UserId int64
}

type IsFavoriteQuery struct {
	TargetType   int32 // 0: video, 1: comment
	FavoriteType int32 // 0: like, 1: dislike
	Items        []IsFavoriteQueryItem
}

type IsFavoriteResultItem struct {
	BizId      int64
	UserId     int64
	IsFavorite bool
}

type IsFavoriteResult struct {
	Items []IsFavoriteResultItem
}

// 简化的仓储接口 - 只做数据访问
type FavoriteRepo interface {
	// 点赞操作
	CreateFavorite(ctx context.Context, favorite *Favorite) error
	UpdateFavorite(ctx context.Context, favorite *Favorite) error
	GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (*Favorite, error)
	SoftDeleteFavorite(ctx context.Context, favoriteId int64) error

	// 查询操作
	ListFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32, offset, limit int) ([]*Favorite, error)
	CountFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (int64, error)
	CountFavoritesByTargetIds(ctx context.Context, targetIds []int64, targetType, favoriteType int32) (map[int64]int64, error)
	CountFavoritesByUserIds(ctx context.Context, userIds []int64, targetType, favoriteType int32) (map[int64]int64, error)
	GetFavoritesByUserAndTargets(ctx context.Context, userId int64, targetIds []int64, targetType, favoriteType int32) ([]*Favorite, error)
}

type FavoriteUsecase struct {
	repo FavoriteRepo
	log  *log.Helper
}

func NewFavoriteUsecase(repo FavoriteRepo, logger log.Logger) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, cmd *AddFavoriteCommand) (*AddFavoriteResult, error) {
	// 业务验证
	if cmd.UserId <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}
	if cmd.TargetId <= 0 {
		return nil, fmt.Errorf("目标ID无效")
	}
	if cmd.TargetType != 0 && cmd.TargetType != 1 {
		return nil, fmt.Errorf("目标类型无效")
	}
	if cmd.FavoriteType != 0 && cmd.FavoriteType != 1 {
		return nil, fmt.Errorf("点赞类型无效")
	}

	// 业务逻辑：检查是否已经点赞
	existingFavorite, err := uc.repo.GetFavorite(ctx, cmd.UserId, cmd.TargetId, cmd.TargetType, cmd.FavoriteType)
	if err != nil {
		uc.log.Errorf("检查点赞状态失败: userId=%d, targetId=%d, err=%v", cmd.UserId, cmd.TargetId, err)
		return nil, fmt.Errorf("检查点赞状态失败")
	}

	if existingFavorite != nil {
		if existingFavorite.IsDeleted {
			// 曾经点赞过但取消了，现在重新点赞
			existingFavorite.IsDeleted = false
			existingFavorite.UpdatedAt = time.Now()
			err = uc.repo.UpdateFavorite(ctx, existingFavorite)
			if err != nil {
				uc.log.Errorf("恢复点赞失败: userId=%d, targetId=%d, err=%v", cmd.UserId, cmd.TargetId, err)
				return nil, fmt.Errorf("恢复点赞失败")
			}
		}
		// 已经点赞，直接返回成功（幂等性）
		return &AddFavoriteResult{}, nil
	}

	// 创建新的点赞记录
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

	err = uc.repo.CreateFavorite(ctx, favorite)
	if err != nil {
		uc.log.Errorf("创建点赞记录失败: userId=%d, targetId=%d, err=%v", cmd.UserId, cmd.TargetId, err)
		return nil, fmt.Errorf("创建点赞记录失败")
	}

	return &AddFavoriteResult{}, nil
}

func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, cmd *RemoveFavoriteCommand) (*RemoveFavoriteResult, error) {
	// 业务验证
	if cmd.UserId <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}
	if cmd.TargetId <= 0 {
		return nil, fmt.Errorf("目标ID无效")
	}

	// 业务逻辑：检查是否有点赞记录
	existingFavorite, err := uc.repo.GetFavorite(ctx, cmd.UserId, cmd.TargetId, cmd.TargetType, cmd.FavoriteType)
	if err != nil {
		uc.log.Errorf("检查点赞状态失败: userId=%d, targetId=%d, err=%v", cmd.UserId, cmd.TargetId, err)
		return nil, fmt.Errorf("检查点赞状态失败")
	}

	if existingFavorite == nil || existingFavorite.IsDeleted {
		// 没有点赞记录或已经取消，直接返回成功（幂等性）
		return &RemoveFavoriteResult{}, nil
	}

	// 软删除点赞记录
	err = uc.repo.SoftDeleteFavorite(ctx, existingFavorite.Id)
	if err != nil {
		uc.log.Errorf("取消点赞失败: userId=%d, targetId=%d, err=%v", cmd.UserId, cmd.TargetId, err)
		return nil, fmt.Errorf("取消点赞失败")
	}

	return &RemoveFavoriteResult{}, nil
}

func (uc *FavoriteUsecase) ListFavorite(ctx context.Context, query *ListFavoriteQuery) (*ListFavoriteResult, error) {
	// 业务验证
	if query.Id <= 0 {
		return nil, fmt.Errorf("查询ID无效")
	}
	if query.FavoriteType != 0 && query.FavoriteType != 1 {
		return nil, fmt.Errorf("点赞类型无效")
	}

	// 分页参数验证
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}

	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize

	var userId, targetId int64
	var targetType int32

	// 根据聚合类型设置查询参数
	switch query.AggregateType {
	case 0: // BY_VIDEO
		targetId = query.Id
		targetType = 0 // VIDEO
		userId = -1
	case 1: // BY_COMMENT
		targetId = query.Id
		targetType = 1 // COMMENT
		userId = -1
	case 2: // BY_USER
		userId = query.Id
		targetType = 0 // VIDEO (用户维度只获取视频点赞)
		targetId = -1
	default:
		return nil, fmt.Errorf("聚合类型无效: %d", query.AggregateType)
	}

	// 查询点赞记录
	favorites, err := uc.repo.ListFavorites(ctx, userId, targetId, targetType, query.FavoriteType, int(offset), int(query.PageStats.PageSize))
	if err != nil {
		uc.log.Errorf("查询点赞列表失败: aggregateType=%d, id=%d, err=%v", query.AggregateType, query.Id, err)
		return nil, fmt.Errorf("查询点赞列表失败")
	}

	// 获取总数
	total, err := uc.repo.CountFavorites(ctx, userId, targetId, targetType, query.FavoriteType)
	if err != nil {
		uc.log.Errorf("统计点赞总数失败: aggregateType=%d, id=%d, err=%v", query.AggregateType, query.Id, err)
		return nil, fmt.Errorf("统计点赞总数失败")
	}

	// 提取目标ID
	targetIds := make([]int64, 0, len(favorites))
	for _, fav := range favorites {
		targetIds = append(targetIds, fav.TargetId)
	}

	return &ListFavoriteResult{
		TargetIds: targetIds,
		Total:     total,
	}, nil
}

func (uc *FavoriteUsecase) CountFavorite(ctx context.Context, query *CountFavoriteQuery) (*CountFavoriteResult, error) {
	// 业务验证
	if len(query.Ids) == 0 {
		return &CountFavoriteResult{Items: []CountFavoriteResultItem{}}, nil
	}

	if len(query.Ids) > 1000 {
		return nil, fmt.Errorf("ID列表过长，最多支持1000个ID")
	}

	if query.FavoriteType != 0 && query.FavoriteType != 1 {
		return nil, fmt.Errorf("点赞类型无效")
	}

	var resultMap map[int64]int64
	var err error

	// 根据聚合类型统计
	switch query.AggregateType {
	case 0: // BY_VIDEO
		resultMap, err = uc.repo.CountFavoritesByTargetIds(ctx, query.Ids, 0, query.FavoriteType)
	case 1: // BY_COMMENT
		resultMap, err = uc.repo.CountFavoritesByTargetIds(ctx, query.Ids, 1, query.FavoriteType)
	case 2: // BY_USER
		resultMap, err = uc.repo.CountFavoritesByUserIds(ctx, query.Ids, 0, query.FavoriteType)
	default:
		return nil, fmt.Errorf("聚合类型无效: %d", query.AggregateType)
	}

	if err != nil {
		uc.log.Errorf("统计点赞数量失败: aggregateType=%d, ids=%v, err=%v", query.AggregateType, query.Ids, err)
		return nil, fmt.Errorf("统计点赞数量失败")
	}

	// 构建结果
	items := make([]CountFavoriteResultItem, 0, len(resultMap))
	for id, count := range resultMap {
		items = append(items, CountFavoriteResultItem{
			BizId: id,
			Count: count,
		})
	}

	return &CountFavoriteResult{
		Items: items,
	}, nil
}

func (uc *FavoriteUsecase) IsFavorite(ctx context.Context, query *IsFavoriteQuery) (*IsFavoriteResult, error) {
	// 业务验证
	if len(query.Items) == 0 {
		return &IsFavoriteResult{Items: []IsFavoriteResultItem{}}, nil
	}

	if len(query.Items) > 1000 {
		return nil, fmt.Errorf("查询项过多，最多支持1000个")
	}

	// 提取用户ID和目标ID
	userId := int64(0)
	targetIds := make([]int64, 0, len(query.Items))

	// 检查是否所有项都是同一个用户（通常是这样）
	for i, item := range query.Items {
		if i == 0 {
			userId = item.UserId
		} else if item.UserId != userId {
			return nil, fmt.Errorf("只能查询同一用户的点赞状态")
		}

		if item.UserId <= 0 || item.BizId <= 0 {
			continue
		}
		targetIds = append(targetIds, item.BizId)
	}

	if userId <= 0 || len(targetIds) == 0 {
		// 返回所有项，但都设置为未点赞
		items := make([]IsFavoriteResultItem, 0, len(query.Items))
		for _, item := range query.Items {
			items = append(items, IsFavoriteResultItem{
				BizId:      item.BizId,
				UserId:     item.UserId,
				IsFavorite: false, // 明确设置为 false
			})
		}
		return &IsFavoriteResult{Items: items}, nil
	}

	// 查询点赞记录
	favorites, err := uc.repo.GetFavoritesByUserAndTargets(ctx, userId, targetIds, query.TargetType, query.FavoriteType)
	if err != nil {
		uc.log.Errorf("查询点赞状态失败: userId=%d, targetIds=%v, err=%v", userId, targetIds, err)
		return nil, fmt.Errorf("查询点赞状态失败")
	}

	// 构建已点赞的映射
	favoritedMap := make(map[int64]bool)
	for _, fav := range favorites {
		favoritedMap[fav.TargetId] = true
	}

	// 构建结果 - 为每个查询项都创建结果
	items := make([]IsFavoriteResultItem, 0, len(query.Items))
	for _, item := range query.Items {
		isFavorite := false
		if favoritedMap[item.BizId] {
			isFavorite = true
		}
		items = append(items, IsFavoriteResultItem{
			BizId:      item.BizId,
			UserId:     item.UserId,
			IsFavorite: isFavorite, // 设置点赞状态
		})
	}

	return &IsFavoriteResult{
		Items: items,
	}, nil
}
