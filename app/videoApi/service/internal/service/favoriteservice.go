package service

import (
	"context"
	"github.com/spf13/cast"
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

	// 将 int32 转换为 proto 枚举，处理 -1 的情况
	var pbFavoriteType pb.FavoriteType
	if result.FavoriteType == 0 {
		pbFavoriteType = pb.FavoriteType_FAVORITE_TYPE_LIKE
	} else if result.FavoriteType == 1 {
		pbFavoriteType = pb.FavoriteType_FAVORITE_TYPE_DISLIKE
	} else {
		// 未点赞状态，默认设置为 LIKE（前端应优先判断 IsFavorite）
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

func (s *FavoriteServiceService) BatchCheckFavoriteStatus(ctx context.Context, req *pb.BatchCheckFavoriteStatusReq) (*pb.BatchCheckFavoriteStatusResp, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		userId = "0"
	}

	// 调用 core 的 BatchIsFavorite
	targetIds := make([]string, 0, len(req.Ids))
	for _, id := range req.Ids {
		targetIds = append(targetIds, id)
	}

	coreReq := &core.BatchIsFavoriteReq{
		UserId: userId,
		BizIds: targetIds,
		Target: core.FavoriteTarget(req.Target),
	}

	resp, err := s.uc.BatchIsFavorite(ctx, coreReq)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.BatchCheckFavoriteStatusItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &pb.BatchCheckFavoriteStatusItem{
			Id:           cast.ToString(item.BizId),
			IsLiked:      item.IsLiked,
			IsDisliked:   item.IsDisliked,
			LikeCount:    item.LikeCount,
			DislikeCount: item.DislikeCount,
		})
	}

	return &pb.BatchCheckFavoriteStatusResp{Items: items}, nil
}
