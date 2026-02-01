package service

import (
	"context"
	"lehu-video/app/videoApi/service/internal/biz"

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
	// 类型转换
	target := biz.FavoriteTarget(req.Target)
	_type := biz.FavoriteType(req.Type)

	input := &biz.AddFavoriteInput{
		Target: &target,
		Type:   &_type,
		Id:     req.Id,
	}

	result, err := s.uc.AddFavorite(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.AddFavoriteResp{
		AlreadyFavorited: result.AlreadyFavorited,
		TotalCount:       result.TotalCount,
		TotalLikes:       result.TotalLikes,
		TotalDislikes:    result.TotalDislikes,
	}, nil
}

func (s *FavoriteServiceService) RemoveFavorite(ctx context.Context, req *pb.RemoveFavoriteReq) (*pb.RemoveFavoriteResp, error) {
	target := biz.FavoriteTarget(req.Target)
	_type := biz.FavoriteType(req.Type)

	input := &biz.RemoveFavoriteInput{
		Target: &target,
		Type:   &_type,
		Id:     req.Id,
	}

	result, err := s.uc.RemoveFavorite(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveFavoriteResp{
		NotFavorited:  result.NotFavorited,
		TotalCount:    result.TotalCount,
		TotalLikes:    result.TotalLikes,
		TotalDislikes: result.TotalDislikes,
	}, nil
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

	return &pb.CheckFavoriteStatusResp{
		IsFavorite:    result.IsFavorite,
		FavoriteType:  pb.FavoriteType(result.FavoriteType),
		TotalLikes:    result.TotalLikes,
		TotalDislikes: result.TotalDislikes,
		TotalCount:    result.TotalCount,
	}, nil
}

func (s *FavoriteServiceService) GetFavoriteStats(ctx context.Context, req *pb.GetFavoriteStatsReq) (*pb.GetFavoriteStatsResp, error) {
	target := biz.FavoriteTarget(req.Target)

	input := &biz.GetFavoriteStatsInput{
		Target: &target,
		Id:     req.Id,
	}

	stats, err := s.uc.GetFavoriteStats(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.GetFavoriteStatsResp{
		LikeCount:    stats.LikeCount,
		DislikeCount: stats.DislikeCount,
		TotalCount:   stats.TotalCount,
		HotScore:     float32(stats.HotScore),
	}, nil
}
