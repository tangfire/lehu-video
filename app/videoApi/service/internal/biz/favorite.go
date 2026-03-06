package biz

import (
	"context"
	"errors"
	"fmt"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
)

var (
	VIDEO   FavoriteTarget = 0
	COMMENT FavoriteTarget = 1
)

var (
	FAVORITE FavoriteType = 0
	UNLIKE   FavoriteType = 1
)

type AddFavoriteInput struct {
	Target *FavoriteTarget
	Type   *FavoriteType
	Id     string
}

type RemoveFavoriteInput struct {
	Target *FavoriteTarget
	Type   *FavoriteType
	Id     string
}

type ListFavoriteVideoInput struct {
	UserId       string
	PageStats    *PageStats
	IncludeStats bool // 是否包含统计信息
}

type CheckFavoriteStatusInput struct {
	Target *FavoriteTarget
	Type   *FavoriteType
	Id     string
}

type BatchCheckFavoriteInput struct {
	Ids    []string
	Target *FavoriteTarget
}

type GetFavoriteStatsInput struct {
	Target *FavoriteTarget
	Id     string
}

// BatchFavoriteStatus 批量查询结果项（包含状态和计数）
type BatchFavoriteStatus struct {
	BizId        string
	IsLiked      bool
	IsDisliked   bool
	LikeCount    int64
	DislikeCount int64
}

type BatchFavoriteStatusResult struct {
	Items []*BatchFavoriteStatus
}

// FavoriteCount 点赞计数（用于批量返回）
type FavoriteCount struct {
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
}

// IsFavoriteResult 单个点赞状态结果（只包含状态）
type IsFavoriteResult struct {
	IsFavorite   bool
	FavoriteType int32 // 0:点赞, 1:点踩, -1:无
}

// FavoriteStats 点赞统计（包含计数）
type FavoriteStats struct {
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
	HotScore     float64
}

// CheckFavoriteResult 组合状态和计数（用于 CheckFavoriteStatus）
type CheckFavoriteResult struct {
	IsFavorite    bool
	FavoriteType  int32
	TotalLikes    int64
	TotalDislikes int64
	TotalCount    int64
}

type FavoriteUsecase struct {
	core      CoreAdapter
	assembler *VideoAssembler
	log       *log.Helper
}

func NewFavoriteUsecase(core CoreAdapter, assembler *VideoAssembler, logger log.Logger) *FavoriteUsecase {
	return &FavoriteUsecase{
		core:      core,
		assembler: assembler,
		log:       log.NewHelper(logger),
	}
}

func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, input *AddFavoriteInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}
	if input.Target == nil || input.Type == nil {
		return errors.New("参数不完整")
	}
	if input.Id == "" {
		return errors.New("目标ID不能为空")
	}

	err = uc.core.AddFavorite(ctx, input.Id, userId, input.Target, input.Type)
	if err != nil {
		uc.log.Errorf("添加点赞失败: userId=%s, targetId=%s, err=%v", userId, input.Id, err)
		return fmt.Errorf("添加点赞失败: %w", err)
	}
	return nil
}

func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, input *RemoveFavoriteInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}
	if input.Target == nil || input.Type == nil {
		return errors.New("参数不完整")
	}
	if input.Id == "" {
		return errors.New("目标ID不能为空")
	}

	err = uc.core.RemoveFavorite(ctx, input.Id, userId, input.Target, input.Type)
	if err != nil {
		uc.log.Errorf("取消点赞失败: userId=%s, targetId=%s, err=%v", userId, input.Id, err)
		return fmt.Errorf("取消点赞失败: %w", err)
	}
	return nil
}

// ListFavoriteVideo 获取用户点赞视频列表
func (uc *FavoriteUsecase) ListFavoriteVideo(ctx context.Context, input *ListFavoriteVideoInput) (int64, []*Video, error) {
	currentUserId, err := claims.GetUserId(ctx)
	if err != nil {
		currentUserId = "0"
	}
	queryUserId := input.UserId
	if cast.ToInt64(queryUserId) <= 0 {
		queryUserId = currentUserId
	}
	if cast.ToInt64(queryUserId) <= 0 {
		return 0, nil, errors.New("用户ID无效")
	}

	total, videoIds, err := uc.core.ListUserFavoriteVideo(ctx, queryUserId, input.PageStats)
	if err != nil {
		uc.log.Errorf("获取点赞视频列表失败: userId=%s, err=%v", queryUserId, err)
		return 0, nil, fmt.Errorf("获取点赞视频列表失败: %w", err)
	}
	if len(videoIds) == 0 {
		return total, []*Video{}, nil
	}

	videos, err := uc.core.GetVideoByIdList(ctx, videoIds)
	if err != nil {
		uc.log.Errorf("获取视频详情失败: videoIds=%v, err=%v", videoIds, err)
		return 0, nil, fmt.Errorf("获取视频详情失败: %w", err)
	}

	var result []*Video
	if input.IncludeStats {
		result, err = uc.assembler.AssembleVideos(ctx, videos, currentUserId)
		if err != nil {
			uc.log.Warnf("组装视频信息失败: err=%v", err)
			result = videos
		}
	} else {
		result = videos
	}
	return total, result, nil
}

func (uc *FavoriteUsecase) CheckFavoriteStatus(ctx context.Context, input *CheckFavoriteStatusInput) (*CheckFavoriteResult, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		userId = "0"
	}
	if input.Target == nil || input.Type == nil {
		return nil, errors.New("参数不完整")
	}
	if input.Id == "" {
		return nil, errors.New("目标ID不能为空")
	}

	// 调用 core 获取点赞状态
	statusResp, err := uc.core.IsFavorite(ctx, userId, input.Id, input.Target)
	if err != nil {
		uc.log.Errorf("查询点赞状态失败: userId=%s, targetId=%s, err=%v", userId, input.Id, err)
		return nil, fmt.Errorf("查询点赞状态失败: %w", err)
	}

	// 调用 core 获取统计信息
	stats, err := uc.core.GetFavoriteStats(ctx, input.Id, input.Target)
	if err != nil {
		uc.log.Warnf("获取点赞统计失败: targetId=%s, err=%v", input.Id, err)
		// 统计失败不影响状态返回
	}

	// 转换类型
	var favoriteTypeInt int32
	if statusResp.IsFavorite {
		switch statusResp.FavoriteType {
		case int32(core.FavoriteType_FAVORITE_TYPE_LIKE):
			favoriteTypeInt = 0
		case int32(core.FavoriteType_FAVORITE_TYPE_DISLIKE):
			favoriteTypeInt = 1
		default:
			favoriteTypeInt = -1
		}
	} else {
		favoriteTypeInt = -1
	}

	likeCount, dislikeCount := int64(0), int64(0)
	if stats != nil {
		likeCount = stats.LikeCount
		dislikeCount = stats.DislikeCount
	}

	return &CheckFavoriteResult{
		IsFavorite:    statusResp.IsFavorite,
		FavoriteType:  favoriteTypeInt,
		TotalLikes:    likeCount,
		TotalDislikes: dislikeCount,
		TotalCount:    likeCount + dislikeCount,
	}, nil
}

// BatchIsFavorite 批量查询点赞/点踩状态和计数
func (uc *FavoriteUsecase) BatchIsFavorite(ctx context.Context, req *core.BatchIsFavoriteReq) (*BatchFavoriteStatusResult, error) {
	if req == nil {
		return nil, errors.New("req is nil")
	}

	// 1. 调用 core 获取点赞状态
	statusResp, err := uc.core.BatchIsFavorite(ctx, req.UserId, req.BizIds, FavoriteTarget(req.Target))
	if err != nil {
		return nil, err
	}
	if err := respcheck.ValidateResponseMeta(statusResp.Meta); err != nil {
		return nil, err
	}

	// 2. 调用 core 获取计数（使用 CountFavorite4Video，根据目标类型选择聚合类型）
	var countMap map[string]FavoriteCount
	if req.Target == core.FavoriteTarget_FAVORITE_TARGET_VIDEO {
		countMap, err = uc.core.CountFavorite4Video(ctx, req.BizIds)
	} else {
		// 如果是评论，需要类似方法，这里假设 CountFavorite4Comment 存在，如果不存在需要扩展
		// 暂时用 CountFavorite4Video 占位，实际应根据业务实现
		countMap, err = uc.core.CountFavorite4Video(ctx, req.BizIds) // 需要根据评论调整
	}
	if err != nil {
		uc.log.Warnf("批量获取计数失败: %v", err)
		// 计数失败时，仍返回状态，计数为0
	}

	// 3. 合并结果
	items := make([]*BatchFavoriteStatus, 0, len(statusResp.Items))
	for _, item := range statusResp.Items {
		cnt := countMap[item.BizId]
		items = append(items, &BatchFavoriteStatus{
			BizId:        item.BizId,
			IsLiked:      item.IsLiked,
			IsDisliked:   item.IsDisliked,
			LikeCount:    cnt.LikeCount,
			DislikeCount: cnt.DislikeCount,
		})
	}

	return &BatchFavoriteStatusResult{Items: items}, nil
}
