package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type Favorite struct {
	UserId       int64
	TargetType   int64
	TargetId     int64
	FavoriteType int64
}

type FavoriteRepo interface {
	AddFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) error
	GetFavoriteList(ctx context.Context, userId, targetId int64, targetType, favoriteType int32, pageStats *pb.PageStatsReq) (int64, []int64, error)
	DeleteFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) error
	CountFavorite(ctx context.Context, idList []int64, aggType int32, favoriteType int32) ([]*pb.CountFavoriteRespItem, error)
	GetFavoriteListByList(ctx context.Context, userIdList, targetIdList []int64, targetType, favoriteType int32) ([]Favorite, error)
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

// 获取点赞列表
func (uc *FavoriteUsecase) ListFavorite(ctx context.Context, req *pb.ListFavoriteReq) (*pb.ListFavoriteResp, error) {
	var (
		targetIdList []int64
		err          error
		total        int64
	)
	switch req.AggregateType {
	// 用户维度的时候，只获取视频的
	case pb.FavoriteAggregateType_BY_USER:
		total, targetIdList, err = uc.repo.GetFavoriteList(ctx, req.Id, -1, int32(pb.FavoriteTarget_VIDEO), int32(req.FavoriteType), req.PageStats)
		if err != nil {
			return nil, err
		}
	case pb.FavoriteAggregateType_BY_COMMENT:
		total, targetIdList, err = uc.repo.GetFavoriteList(ctx, -1, req.Id, int32(pb.FavoriteTarget_COMMENT), int32(req.FavoriteType), req.PageStats)
		if err != nil {
			return nil, err
		}
	case pb.FavoriteAggregateType_BY_VIDEO:
		total, targetIdList, err = uc.repo.GetFavoriteList(ctx, -1, req.Id, int32(pb.FavoriteTarget_VIDEO), int32(req.FavoriteType), req.PageStats)
		if err != nil {
			return nil, err
		}
	}
	return &pb.ListFavoriteResp{
		Meta:      utils.GetSuccessMeta(),
		Id:        targetIdList,
		PageStats: &pb.PageStatsResp{Total: int32(total)},
	}, nil
}

func (uc *FavoriteUsecase) CountFavorite(ctx context.Context, req *pb.CountFavoriteReq) (*pb.CountFavoriteResp, error) {
	favoriteCountList, err := uc.repo.CountFavorite(ctx, req.Id, int32(req.AggregateType), int32(req.FavoriteType))
	if err != nil {
		return nil, err
	}
	return &pb.CountFavoriteResp{
		Meta:  utils.GetSuccessMeta(),
		Items: favoriteCountList,
	}, nil
}

func (uc *FavoriteUsecase) IsFavorite(ctx context.Context, req *pb.IsFavoriteReq) (*pb.IsFavoriteResp, error) {
	var userIdList, targetIdList []int64
	for _, v := range req.Items {
		userIdList = append(userIdList, v.UserId)
		targetIdList = append(targetIdList, v.BizId)
	}
	list, err := uc.repo.GetFavoriteListByList(ctx, userIdList, targetIdList, int32(req.Target), int32(req.Target))
	if err != nil {
		return nil, err
	}
	favoriteMap := make(map[string]bool)
	for _, item := range list {
		key := fmt.Sprintf("%d_%d", item.UserId, item.TargetId)
		favoriteMap[key] = true
	}
	var retList []*pb.IsFavoriteRespItem
	for _, item := range req.Items {
		key := fmt.Sprintf("%d_%d", item.UserId, item.BizId)
		if _, ok := favoriteMap[key]; ok {
			tmp := &pb.IsFavoriteRespItem{
				BizId:  item.BizId,
				UserId: item.UserId,
			}
			retList = append(retList, tmp)
		}
	}
	return &pb.IsFavoriteResp{
		Meta:  utils.GetSuccessMeta(),
		Items: retList,
	}, nil
}
