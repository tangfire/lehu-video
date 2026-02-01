package service

import (
	"context"
	"github.com/spf13/cast"
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
		UserId:       cast.ToInt64(req.UserId),
		TargetId:     cast.ToInt64(req.BizId),
		TargetType:   int32(req.Target),
		FavoriteType: int32(req.Type),
	}

	result, err := s.uc.AddFavorite(ctx, cmd)
	if err != nil {
		return &pb.AddFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.AddFavoriteResp{
		Meta:             utils.GetSuccessMeta(),
		AlreadyFavorited: result.AlreadyFavorited,
		TotalCount:       result.TotalCount,
	}, nil
}

func (s *FavoriteServiceService) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	cmd := &biz.RemoveFavoriteCommand{
		UserId:       cast.ToInt64(req.UserId),
		TargetId:     cast.ToInt64(req.BizId),
		TargetType:   int32(req.Target),
		FavoriteType: int32(req.Type),
	}

	result, err := s.uc.RemoveFavorite(ctx, cmd)
	if err != nil {
		return &pb.RemoveFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RemoveFavoriteResp{
		Meta:         utils.GetSuccessMeta(),
		NotFavorited: result.NotFavorited,
		TotalCount:   result.TotalCount,
	}, nil
}

func (s *FavoriteServiceService) ListFavorite(ctx context.Context, req *pb.ListFavoriteReq) (*pb.ListFavoriteResp, error) {
	// 处理分页参数
	var pageStatsReq *pb.PageStatsReq
	if req.PageStats != nil {
		pageStatsReq = req.PageStats
	} else {
		// 默认分页参数
		pageStatsReq = &pb.PageStatsReq{
			Page: 1,
			Size: 20,
		}
	}

	query := &biz.ListFavoriteQuery{
		Id:             cast.ToInt64(req.Id),
		AggregateType:  int32(req.AggregateType),
		FavoriteType:   int32(req.FavoriteType),
		PageStats:      biz.PageStats{Page: pageStatsReq.Page, PageSize: pageStatsReq.Size},
		IncludeDeleted: req.IncludeDeleted,
	}

	result, err := s.uc.ListFavorite(ctx, query)
	if err != nil {
		return &pb.ListFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 将int64类型的TargetIds转换为字符串数组
	ids := make([]string, 0, len(result.TargetIds))
	for _, id := range result.TargetIds {
		ids = append(ids, cast.ToString(id))
	}

	pageStatsResp := &pb.PageStatsResp{
		Total: int32(result.Total),
	}

	return &pb.ListFavoriteResp{
		Meta:       utils.GetSuccessMeta(),
		Ids:        ids,
		PageStats:  pageStatsResp,
		TotalCount: result.TotalCount,
	}, nil
}

func (s *FavoriteServiceService) CountFavorite(ctx context.Context, req *pb.CountFavoriteReq) (*pb.CountFavoriteResp, error) {
	query := &biz.CountFavoriteQuery{
		Ids:           make([]int64, 0, len(req.Ids)),
		AggregateType: int32(req.AggregateType),
		FavoriteType:  int32(req.FavoriteType),
	}

	// 转换IDs
	for _, id := range req.Ids {
		query.Ids = append(query.Ids, cast.ToInt64(id))
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
			BizId:        cast.ToString(item.BizId),
			LikeCount:    item.LikeCount,
			DislikeCount: item.DislikeCount,
			TotalCount:   item.TotalCount,
		})
	}

	return &pb.CountFavoriteResp{
		Meta:  utils.GetSuccessMeta(),
		Items: pbItems,
	}, nil
}

func (s *FavoriteServiceService) IsFavorite(ctx context.Context, req *pb.IsFavoriteReq) (*pb.IsFavoriteResp, error) {
	query := &biz.IsFavoriteQuery{
		UserId:       cast.ToInt64(req.UserId),
		TargetId:     cast.ToInt64(req.BizId),
		TargetType:   int32(req.Target),
		FavoriteType: int32(req.Type),
	}

	result, err := s.uc.IsFavorite(ctx, query)
	if err != nil {
		return &pb.IsFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var favoriteType pb.FavoriteType
	if result.FavoriteType == 0 {
		favoriteType = pb.FavoriteType_FAVORITE_TYPE_LIKE
	} else if result.FavoriteType == 1 {
		favoriteType = pb.FavoriteType_FAVORITE_TYPE_DISLIKE
	} else {
		// 未点赞状态
		favoriteType = pb.FavoriteType_FAVORITE_TYPE_LIKE // 默认值
	}

	return &pb.IsFavoriteResp{
		Meta:          utils.GetSuccessMeta(),
		IsFavorite:    result.IsFavorite,
		FavoriteType:  favoriteType,
		TotalLikes:    result.TotalLikes,
		TotalDislikes: result.TotalDislikes,
	}, nil
}

func (s *FavoriteServiceService) BatchIsFavorite(ctx context.Context, req *pb.BatchIsFavoriteReq) (*pb.BatchIsFavoriteResp, error) {
	// 将字符串IDs转换为int64
	targetIds := make([]int64, 0, len(req.BizIds))
	for _, id := range req.BizIds {
		targetIds = append(targetIds, cast.ToInt64(id))
	}

	// 注意：proto中BatchIsFavoriteReq只有UserId，但biz层需要UserIds数组
	// 这里假设是查询一个用户对多个目标的点赞状态
	userIds := []int64{cast.ToInt64(req.UserId)}

	query := &biz.BatchIsFavoriteQuery{
		UserIds:    userIds,
		TargetIds:  targetIds,
		TargetType: int32(req.Target),
	}

	result, err := s.uc.BatchIsFavorite(ctx, query)
	if err != nil {
		return &pb.BatchIsFavoriteResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var pbItems []*pb.BatchIsFavoriteItem
	for _, item := range result.Items {
		// 只返回请求的用户对应的记录
		if item.UserId == cast.ToInt64(req.UserId) {
			pbItems = append(pbItems, &pb.BatchIsFavoriteItem{
				BizId:        cast.ToString(item.TargetId),
				IsLiked:      item.IsLiked,
				IsDisliked:   item.IsDisliked,
				LikeCount:    item.LikeCount,
				DislikeCount: item.DislikeCount,
			})
		}
	}

	return &pb.BatchIsFavoriteResp{
		Meta:  utils.GetSuccessMeta(),
		Items: pbItems,
	}, nil
}
