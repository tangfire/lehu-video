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
	// 类型转换：pb.枚举 -> biz.枚举
	target := biz.FavoriteTarget(req.Target)
	_type := biz.FavoriteType(req.Type)

	input := &biz.AddFavoriteInput{
		Target: &target, // 使用转换后的值
		Type:   &_type,  // 使用转换后的值
		Id:     req.Id,  // 注意：pb.AddFavoriteReq 中没有 Id 字段，只有 UserId
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
	input := &biz.ListFavoriteVideoInput{
		UserId: req.UserId,
		PageStats: &biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}
	total, videos, err := s.uc.ListFavoriteVideo(ctx, input)
	if err != nil {
		return nil, err
	}
	var retVideos []*pb.Video
	for _, video := range videos {
		retVideos = append(retVideos, &pb.Video{
			Id: video.ID,
			Author: &pb.VideoAuthor{
				Id:          video.Author.ID,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing,
			},
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
		Videos:    retVideos,
		PageStats: &pb.PageStatsResp{Total: int32(total)},
	}, nil
}
