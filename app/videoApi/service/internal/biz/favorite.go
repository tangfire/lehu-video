package biz

import (
	"context"
	"errors"
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
	UserId    string
	PageStats *PageStats
}

type ListFavoriteVideoOutput struct {
	Videos []*Video
	Total  int64
}

type FavoriteUsecase struct {
	core      CoreAdapter
	assembler *VideoAssembler
	log       *log.Helper
}

func NewFavoirteUsecase(core CoreAdapter, assembler *VideoAssembler, logger log.Logger) *FavoriteUsecase {
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

	err = uc.core.AddFavorite(ctx, input.Id, userId, input.Target, input.Type)
	if err != nil {
		return errors.New("failed to add favorite")
	}
	return nil
}

func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, input *RemoveFavoriteInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}
	err = uc.core.RemoveFavorite(ctx, input.Id, userId, input.Target, input.Type)
	if err != nil {
		return errors.New("failed to remove favorite")
	}
	return nil
}

func (uc *FavoriteUsecase) ListFavoriteVideo(ctx context.Context, input *ListFavoriteVideoInput) (int64, []*Video, error) {
	if cast.ToInt64(input.UserId) <= 0 {
		userId, err := claims.GetUserId(ctx)
		if err != nil {
			return 0, nil, errors.New("获取用户信息失败")
		}
		input.UserId = userId
	}
	total, videoIds, err := uc.core.ListUserFavoriteVideo(ctx, input.UserId, input.PageStats)
	if err != nil {
		return 0, nil, errors.New("failed to list favorite video")
	}
	if len(videoIds) == 0 {
		return 0, nil, nil
	}
	videos, err := uc.core.GetVideoByIdList(ctx, videoIds)
	if err != nil {
		return 0, nil, errors.New("failed to list favorite video")
	}
	result, err := uc.assembler.AssembleVideos(ctx, videos, input.UserId)
	if err != nil {
		log.Context(ctx).Warnf("something wrong in assembling videos: %v", err)
	}
	return total, result, nil

}
