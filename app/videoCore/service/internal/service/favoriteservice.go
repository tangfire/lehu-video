package service

import (
	"context"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type FavoriteServiceService struct {
	pb.UnimplementedFavoriteServiceServer
	uc *biz.FavoriteUsecase
}

func NewFavoriteServiceService(uc *biz.FavoriteUsecase) *FavoriteServiceService {
	return &FavoriteServiceService{uc: uc}
}

func (s *FavoriteServiceService) AddFavorite(ctx context.Context, req *pb.AddFavoriteReq) (*pb.AddFavoriteResp, error) {
	cmd := &biz.AddFavoriteCommand{
		UserId:       req.UserId,
		TargetId:     req.Id,
		TargetType:   int32(req.Target),
		FavoriteType: int32(req.Type),
	}

	_, err := s.uc.AddFavorite(ctx, cmd)
	if err != nil {
		return &pb.AddFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.AddFavoriteResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FavoriteServiceService) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	cmd := &biz.RemoveFavoriteCommand{
		UserId:       req.UserId,
		TargetId:     req.Id,
		TargetType:   int32(req.Target),
		FavoriteType: int32(req.Type),
	}

	_, err := s.uc.RemoveFavorite(ctx, cmd)
	if err != nil {
		return &pb.RemoveFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RemoveFavoriteResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FavoriteServiceService) ListFavorite(ctx context.Context, req *pb.ListFavoriteReq) (*pb.ListFavoriteResp, error) {
	query := &biz.ListFavoriteQuery{
		Id:            req.Id,
		AggregateType: int32(req.AggregateType),
		FavoriteType:  int32(req.FavoriteType),
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListFavorite(ctx, query)
	if err != nil {
		return &pb.ListFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.ListFavoriteResp{
		Meta:      utils.GetSuccessMeta(),
		Id:        result.TargetIds,
		PageStats: &pb.PageStatsResp{Total: int32(result.Total)},
	}, nil
}

func (s *FavoriteServiceService) CountFavorite(ctx context.Context, req *pb.CountFavoriteReq) (*pb.CountFavoriteResp, error) {
	query := &biz.CountFavoriteQuery{
		Ids:           req.Id,
		AggregateType: int32(req.AggregateType),
		FavoriteType:  int32(req.FavoriteType),
	}

	result, err := s.uc.CountFavorite(ctx, query)
	if err != nil {
		return &pb.CountFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var pbItems []*pb.CountFavoriteRespItem
	for _, item := range result.Items {
		pbItems = append(pbItems, &pb.CountFavoriteRespItem{
			BizId: item.BizId,
			Count: item.Count,
		})
	}

	return &pb.CountFavoriteResp{
		Meta:  utils.GetSuccessMeta(),
		Items: pbItems,
	}, nil
}

func (s *FavoriteServiceService) IsFavorite(ctx context.Context, req *pb.IsFavoriteReq) (*pb.IsFavoriteResp, error) {
	var queryItems []biz.IsFavoriteQueryItem
	for _, item := range req.Items {
		queryItems = append(queryItems, biz.IsFavoriteQueryItem{
			BizId:  item.BizId,
			UserId: item.UserId,
		})
	}

	query := &biz.IsFavoriteQuery{
		TargetType:   int32(req.Target),
		FavoriteType: int32(req.Type),
		Items:        queryItems,
	}

	result, err := s.uc.IsFavorite(ctx, query)
	if err != nil {
		return &pb.IsFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var pbItems []*pb.IsFavoriteRespItem
	for _, item := range result.Items {
		pbItems = append(pbItems, &pb.IsFavoriteRespItem{
			BizId:  item.BizId,
			UserId: item.UserId,
		})
	}

	return &pb.IsFavoriteResp{
		Meta:  utils.GetSuccessMeta(),
		Items: pbItems,
	}, nil
}
