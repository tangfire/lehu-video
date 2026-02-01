package biz

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
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

func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, input *AddFavoriteInput) (*AddFavoriteResult, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	// 参数验证
	if input.Target == nil || input.Type == nil {
		return nil, errors.New("参数不完整")
	}
	if input.Id == "" {
		return nil, errors.New("目标ID不能为空")
	}

	result, err := uc.core.AddFavorite(ctx, input.Id, userId, input.Target, input.Type)
	if err != nil {
		uc.log.Errorf("添加点赞失败: userId=%s, targetId=%s, err=%v", userId, input.Id, err)
		return nil, fmt.Errorf("添加点赞失败: %w", err)
	}

	return result, nil
}

func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, input *RemoveFavoriteInput) (*RemoveFavoriteResult, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	// 参数验证
	if input.Target == nil || input.Type == nil {
		return nil, errors.New("参数不完整")
	}
	if input.Id == "" {
		return nil, errors.New("目标ID不能为空")
	}

	result, err := uc.core.RemoveFavorite(ctx, input.Id, userId, input.Target, input.Type)
	if err != nil {
		uc.log.Errorf("取消点赞失败: userId=%s, targetId=%s, err=%v", userId, input.Id, err)
		return nil, fmt.Errorf("取消点赞失败: %w", err)
	}

	return result, nil
}

func (uc *FavoriteUsecase) ListFavoriteVideo(ctx context.Context, input *ListFavoriteVideoInput) (int64, []*Video, error) {
	// 获取用户ID
	currentUserId, err := claims.GetUserId(ctx)
	if err != nil {
		currentUserId = "0"
	}

	// 确定查询哪个用户的点赞视频
	queryUserId := input.UserId
	if cast.ToInt64(queryUserId) <= 0 {
		queryUserId = currentUserId
	}

	if cast.ToInt64(queryUserId) <= 0 {
		return 0, nil, errors.New("用户ID无效")
	}

	// 获取点赞视频ID列表
	total, videoIds, err := uc.core.ListUserFavoriteVideo(ctx, queryUserId, input.PageStats)
	if err != nil {
		uc.log.Errorf("获取点赞视频列表失败: userId=%s, err=%v", queryUserId, err)
		return 0, nil, fmt.Errorf("获取点赞视频列表失败: %w", err)
	}

	if len(videoIds) == 0 {
		return total, []*Video{}, nil
	}

	// 获取视频详情
	videos, err := uc.core.GetVideoByIdList(ctx, videoIds)
	if err != nil {
		uc.log.Errorf("获取视频详情失败: videoIds=%v, err=%v", videoIds, err)
		return 0, nil, fmt.Errorf("获取视频详情失败: %w", err)
	}

	// 组装视频信息
	var result []*Video
	if input.IncludeStats {
		// 如果需要包含统计信息，进行完整组装
		result, err = uc.assembler.AssembleVideos(ctx, videos, currentUserId)
		if err != nil {
			uc.log.Warnf("组装视频信息失败: err=%v", err)
			// 不返回错误，只记录日志
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
		// 未登录用户也可以查询，但不能查询自己的点赞状态
		userId = "0"
	}

	// 参数验证
	if input.Target == nil || input.Type == nil {
		return nil, errors.New("参数不完整")
	}
	if input.Id == "" {
		return nil, errors.New("目标ID不能为空")
	}

	result, err := uc.core.CheckFavoriteStatus(ctx, userId, input.Id, input.Target, input.Type)
	if err != nil {
		uc.log.Errorf("查询点赞状态失败: userId=%s, targetId=%s, err=%v", userId, input.Id, err)
		return nil, fmt.Errorf("查询点赞状态失败: %w", err)
	}

	return result, nil
}

func (uc *FavoriteUsecase) GetFavoriteStats(ctx context.Context, input *GetFavoriteStatsInput) (*FavoriteStats, error) {
	// 参数验证
	if input.Target == nil {
		return nil, errors.New("目标类型不能为空")
	}
	if input.Id == "" {
		return nil, errors.New("目标ID不能为空")
	}

	stats, err := uc.core.GetFavoriteStats(ctx, input.Id, input.Target)
	if err != nil {
		uc.log.Errorf("获取点赞统计失败: targetId=%s, err=%v", input.Id, err)
		return nil, fmt.Errorf("获取点赞统计失败: %w", err)
	}

	return stats, nil
}
