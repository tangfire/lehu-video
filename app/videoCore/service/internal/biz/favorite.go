package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type FavoriteRepo interface {
	AddFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) error
	GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (bool, int64, error)
	DeleteFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) error
}

type FavoriteUsecase struct {
	repo FavoriteRepo
	log  *log.Helper
}

func NewFavoriteUsecase(repo FavoriteRepo, logger log.Logger) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *FavoriteUsecase) AddFavorite(ctx context.Context, req *pb.AddFavoriteReq) (*pb.AddFavoriteResp, error) {
	err := uc.repo.AddFavorite(ctx, req.UserId, req.Id, int32(req.Target), int32(req.Type))
	if err != nil {
		return nil, err
	}
	return &pb.AddFavoriteResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (uc *FavoriteUsecase) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	err := uc.repo.DeleteFavorite(ctx, req.UserId, req.Id, int32(req.Target), int32(req.Type))
	if err != nil {
		return nil, err
	}
	return &pb.RemoveFavoriteResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
