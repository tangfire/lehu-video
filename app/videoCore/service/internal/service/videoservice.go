package service

import (
	"context"
	"errors"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
	"time"
)

type VideoServiceService struct {
	pb.UnimplementedVideoServiceServer
	uc *biz.VideoUsecase
}

func NewVideoServiceService(uc *biz.VideoUsecase) *VideoServiceService {
	return &VideoServiceService{uc: uc}
}

func (s *VideoServiceService) FeedShortVideo(ctx context.Context, req *pb.FeedShortVideoReq) (*pb.FeedShortResp, error) {
	// 设置分页参数
	pageSize := int32(30) // 默认每页30条
	if req.FeedNum > 0 {
		pageSize = int32(req.FeedNum)
	}

	// ✅ 改为Query
	query := &biz.FeedShortVideoQuery{
		LatestTime: req.LatestTime,
		PageStats: biz.PageStats{
			Page:     1, // 第一页
			PageSize: pageSize,
		},
	}

	result, err := s.uc.FeedShortVideo(ctx, query)
	if err != nil {
		return &pb.FeedShortResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换biz.Video到pb.Video
	var pbVideos []*pb.Video
	for _, video := range result.Videos {
		pbVideos = append(pbVideos, convertBizVideoToPb(video))
	}

	return &pb.FeedShortResp{
		Meta:   utils.GetSuccessMeta(),
		Videos: pbVideos,
	}, nil
}

func (s *VideoServiceService) GetVideoById(ctx context.Context, req *pb.GetVideoByIdReq) (*pb.GetVideoByIdResp, error) {
	// ✅ 改为Query
	query := &biz.GetVideoByIdQuery{
		VideoId: req.VideoId,
	}

	result, err := s.uc.GetVideoById(ctx, query)
	if err != nil {
		return &pb.GetVideoByIdResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if result == nil || result.Video == nil {
		return &pb.GetVideoByIdResp{
			Meta: utils.GetMetaWithError(errors.New("video not found")),
		}, nil
	}

	return &pb.GetVideoByIdResp{
		Meta:  utils.GetSuccessMeta(),
		Video: convertBizVideoToPb(result.Video),
	}, nil
}

func (s *VideoServiceService) PublishVideo(ctx context.Context, req *pb.PublishVideoReq) (*pb.PublishVideoResp, error) {
	// ✅ 改为Command
	cmd := &biz.PublishVideoCommand{
		UserId:      req.UserId,
		Title:       req.Title,
		Description: req.Description,
		PlayUrl:     req.PlayUrl,
		CoverUrl:    req.CoverUrl,
	}

	result, err := s.uc.PublishVideo(ctx, cmd)
	if err != nil {
		return &pb.PublishVideoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.PublishVideoResp{
		Meta:    utils.GetSuccessMeta(),
		VideoId: result.VideoId,
	}, nil
}

func (s *VideoServiceService) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	// 设置分页参数
	pageStats := biz.PageStats{
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
	}

	// ✅ 改为Query
	query := &biz.ListPublishedVideoQuery{
		UserId:     req.UserId,
		LatestTime: req.LatestTime,
		PageStats:  pageStats,
	}

	result, err := s.uc.ListPublishedVideo(ctx, query)
	if err != nil {
		return &pb.ListPublishedVideoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换biz.Video到pb.Video
	var pbVideos []*pb.Video
	for _, video := range result.Videos {
		pbVideos = append(pbVideos, convertBizVideoToPb(video))
	}

	return &pb.ListPublishedVideoResp{
		Meta:      utils.GetSuccessMeta(),
		Videos:    pbVideos,
		PageStats: &pb.PageStatsResp{Total: int32(result.Total)},
	}, nil
}

func (s *VideoServiceService) GetVideoByIdList(ctx context.Context, req *pb.GetVideoByIdListReq) (*pb.GetVideoByIdListResp, error) {
	// ✅ 改为Query
	query := &biz.GetVideoByIdListQuery{
		VideoIdList: req.VideoIdList,
	}

	result, err := s.uc.GetVideoByIdList(ctx, query)
	if err != nil {
		return &pb.GetVideoByIdListResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换biz.Video到pb.Video
	var pbVideos []*pb.Video
	for _, video := range result.Videos {
		pbVideos = append(pbVideos, convertBizVideoToPb(video))
	}

	return &pb.GetVideoByIdListResp{
		Meta:   utils.GetSuccessMeta(),
		Videos: pbVideos,
	}, nil
}

// 转换函数：biz.Video -> pb.Video
func convertBizVideoToPb(video *biz.Video) *pb.Video {
	if video == nil {
		return nil
	}

	pbVideo := &pb.Video{
		Id:            video.Id,
		Title:         video.Title,
		Description:   video.Description,
		PlayUrl:       video.VideoUrl,
		CoverUrl:      video.CoverUrl,
		FavoriteCount: video.LikeCount,
		CommentCount:  video.CommentCount,
		UploadTime:    video.UploadTime.Format(time.DateTime),
	}

	if video.Author != nil {
		pbVideo.Author = &pb.Author{
			Id:     video.Author.Id,
			Name:   video.Author.Name,
			Avatar: video.Author.Avatar,
			// TODO: 需要从用户服务获取is_following信息
			IsFollowing: video.Author.IsFollowing,
		}
	}

	// TODO: 需要从点赞服务获取is_favorite信息
	// pbVideo.IsFavorite = 0

	return pbVideo
}
