package service

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"

	pb "lehu-video/api/videoApi/service/v1"
)

type FavoriteServiceService struct {
	pb.UnimplementedFavoriteServiceServer
	uc *biz.FavoriteUsecase
}

func NewFavoriteServiceService(uc *biz.FavoriteUsecase) *FavoriteServiceService {
	return &FavoriteServiceService{
		uc: uc,
	}
}

func (s *FavoriteServiceService) AddFavorite(ctx context.Context, req *pb.AddFavoriteReq) (*pb.AddFavoriteResp, error) {
	target := biz.FavoriteTarget(req.Target)
	_type := biz.FavoriteType(req.Type)

	input := &biz.AddFavoriteInput{
		Target: &target,
		Type:   &_type,
		Id:     req.Id,
	}

	err := s.uc.AddFavorite(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.AddFavoriteResp{}, nil
}

func (s *FavoriteServiceService) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	target := biz.FavoriteTarget(req.Target)
	_type := biz.FavoriteType(req.Type)

	input := &biz.RemoveFavoriteInput{
		Target: &target,
		Type:   &_type,
		Id:     req.Id,
	}

	err := s.uc.RemoveFavorite(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveFavoriteResp{}, nil
}

func (s *FavoriteServiceService) ListFavoriteVideo(ctx context.Context, req *pb.ListFavoriteVideoReq) (*pb.ListFavoriteVideoResp, error) {
	pageStats := &biz.PageStats{}
	if req.PageStats != nil {
		pageStats.Page = int(req.PageStats.Page)
		pageStats.PageSize = int(req.PageStats.Size)
	} else {
		pageStats.Page = 1
		pageStats.PageSize = 20
	}

	input := &biz.ListFavoriteVideoInput{
		UserId:       req.UserId,
		PageStats:    pageStats,
		IncludeStats: req.IncludeStats,
	}

	total, videos, err := s.uc.ListFavoriteVideo(ctx, input)
	if err != nil {
		return nil, err
	}

	var retVideos []*pb.Video
	for _, video := range videos {
		var author *pb.VideoAuthor
		if video.Author != nil {
			author = &pb.VideoAuthor{
				Id:          video.Author.ID,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing,
			}
		}

		retVideos = append(retVideos, &pb.Video{
			Id:             video.ID,
			Author:         author,
			PlayUrl:        video.PlayURL,
			CoverUrl:       video.CoverURL,
			FavoriteCount:  video.FavoriteCount,
			CommentCount:   video.CommentCount,
			IsFavorite:     video.IsFavorite,
			Title:          video.Title,
			IsCollected:    video.IsCollected,
			CollectedCount: video.CollectedCount,
		})
	}

	return &pb.ListFavoriteVideoResp{
		Videos:     retVideos,
		PageStats:  &pb.PageStatsResp{Total: int32(total)},
		TotalCount: total,
	}, nil
}

func (s *FavoriteServiceService) CheckFavoriteStatus(ctx context.Context, req *pb.CheckFavoriteStatusReq) (*pb.CheckFavoriteStatusResp, error) {
	target := biz.FavoriteTarget(req.Target)
	_type := biz.FavoriteType(req.Type)

	input := &biz.CheckFavoriteStatusInput{
		Target: &target,
		Type:   &_type,
		Id:     req.Id,
	}

	result, err := s.uc.CheckFavoriteStatus(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换枚举
	var pbFavoriteType pb.FavoriteType
	if result.FavoriteType == 0 {
		pbFavoriteType = pb.FavoriteType_FAVORITE_TYPE_LIKE
	} else if result.FavoriteType == 1 {
		pbFavoriteType = pb.FavoriteType_FAVORITE_TYPE_DISLIKE
	} else {
		pbFavoriteType = pb.FavoriteType_FAVORITE_TYPE_LIKE
	}

	return &pb.CheckFavoriteStatusResp{
		IsFavorite:    result.IsFavorite,
		FavoriteType:  pbFavoriteType,
		TotalLikes:    result.TotalLikes,
		TotalDislikes: result.TotalDislikes,
		TotalCount:    result.TotalCount,
	}, nil
}

func (s *FavoriteServiceService) BatchCheckFavoriteStatus(ctx context.Context, req *pb.BatchCheckFavoriteStatusReq) (*pb.BatchCheckFavoriteStatusResp, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		userId = "0"
	}

	coreReq := &core.BatchIsFavoriteReq{
		UserId: userId,
		BizIds: req.Ids,
		Target: core.FavoriteTarget(req.Target),
	}

	// 调用 biz 层批量查询（包含状态和计数）
	result, err := s.uc.BatchIsFavorite(ctx, coreReq)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.BatchCheckFavoriteStatusItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &pb.BatchCheckFavoriteStatusItem{
			Id:           item.BizId,
			IsLiked:      item.IsLiked,
			IsDisliked:   item.IsDisliked,
			LikeCount:    item.LikeCount,
			DislikeCount: item.DislikeCount,
		})
	}

	return &pb.BatchCheckFavoriteStatusResp{Items: items}, nil
}
