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
	//bizReq := &biz.FeedShortVideoRequest{
	//	LatestTime: req.LatestTime,
	//	PageStats: biz.PageStats{
	//		Page:     req.PageStats.Page,
	//		PageSize: req.PageStats.Size,
	//	},
	//}
	//
	//resp, err := s.uc.FeedShortVideo(ctx, bizReq)
	//if err != nil {
	//	return &pb.FeedShortResp{
	//		Meta: utils.GetMetaWithError(err),
	//	}, nil
	//}
	//
	//// 转换biz.Video到pb.Video
	//var pbVideos []*pb.Video
	//for _, video := range resp.Videos {
	//	pbVideos = append(pbVideos, convertBizVideoToPb(video))
	//}

	//return &pb.FeedShortResp{
	//	Meta:   utils.GetSuccessMeta(),
	//	Videos: pbVideos,
	//}, nil
	return nil, nil
}

func (s *VideoServiceService) GetVideoById(ctx context.Context, req *pb.GetVideoByIdReq) (*pb.GetVideoByIdResp, error) {
	bizReq := &biz.GetVideoByIdReq{
		VideoId: req.VideoId,
	}

	resp, err := s.uc.GetVideoById(ctx, bizReq)
	if err != nil {
		return &pb.GetVideoByIdResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if resp == nil || resp.Video == nil {
		return &pb.GetVideoByIdResp{
			Meta: utils.GetMetaWithError(errors.New("video not found")),
		}, nil
	}

	return &pb.GetVideoByIdResp{
		Meta:  utils.GetSuccessMeta(),
		Video: convertBizVideoToPb(resp.Video),
	}, nil
}

func (s *VideoServiceService) PublishVideo(ctx context.Context, req *pb.PublishVideoReq) (*pb.PublishVideoResp, error) {
	bizReq := &biz.PublishVideoReq{
		UserId:      req.UserId,
		Title:       req.Title,
		Description: req.Description,
		PlayUrl:     req.PlayUrl,
		CoverUrl:    req.CoverUrl,
	}

	resp, err := s.uc.PublishVideo(ctx, bizReq)
	if err != nil {
		return &pb.PublishVideoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.PublishVideoResp{
		Meta:    utils.GetSuccessMeta(),
		VideoId: resp.VideoId,
	}, nil
}

func (s *VideoServiceService) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	bizReq := &biz.ListPublishedVideoReq{
		UserId:     req.UserId,
		LatestTime: req.LatestTime,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	resp, err := s.uc.ListPublishedVideo(ctx, bizReq)
	if err != nil {
		return &pb.ListPublishedVideoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换biz.Video到pb.Video
	var pbVideos []*pb.Video
	for _, video := range resp.Videos {
		pbVideos = append(pbVideos, convertBizVideoToPb(video))
	}

	return &pb.ListPublishedVideoResp{
		Meta:   utils.GetSuccessMeta(),
		Videos: pbVideos,
	}, nil
}

func (s *VideoServiceService) GetVideoByIdList(ctx context.Context, req *pb.GetVideoByIdListReq) (*pb.GetVideoByIdListResp, error) {
	bizReq := &biz.GetVideoByIdListReq{
		VideoIdList: req.VideoIdList,
	}

	resp, err := s.uc.GetVideoByIdList(ctx, bizReq)
	if err != nil {
		return &pb.GetVideoByIdListResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换biz.Video到pb.Video
	var pbVideos []*pb.Video
	for _, video := range resp.Videos {
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
		}
	}

	return pbVideo
}
