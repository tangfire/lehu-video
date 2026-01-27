package service

import (
	"context"
	pb "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

// VideoServiceService 视频服务实现
type VideoServiceService struct {
	pb.UnimplementedVideoServiceServer

	uc *biz.VideoUsecase
}

// NewVideoServiceService 创建视频服务
func NewVideoServiceService(uc *biz.VideoUsecase) *VideoServiceService {
	return &VideoServiceService{
		uc: uc,
	}
}

// PreSign4UploadVideo 视频预签名上传
func (s *VideoServiceService) PreSign4UploadVideo(ctx context.Context, req *pb.PreSign4UploadVideoReq) (*pb.PreSign4UploadVideoResp, error) {
	// 构建输入
	input := &biz.PreSignUploadInput{
		Hash:     req.Hash,
		FileType: req.FileType,
		Size:     req.Size,
		Filename: req.Filename,
	}

	// 调用业务层
	output, err := s.uc.PreSignUpload(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.PreSign4UploadVideoResp{
		Url:    output.URL,
		FileId: output.FileID,
	}, nil
}

// PreSign4UploadCover 封面预签名上传
func (s *VideoServiceService) PreSign4UploadCover(ctx context.Context, req *pb.PreSign4UploadCoverReq) (*pb.PreSign4UploadCoverResp, error) {
	// 构建输入
	input := &biz.PreSignUploadInput{
		Hash:     req.Hash,
		FileType: req.FileType,
		Size:     req.Size,
		Filename: req.Filename,
	}

	// 调用业务层
	output, err := s.uc.PreSignUpload(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.PreSign4UploadCoverResp{
		Url:    output.URL,
		FileId: output.FileID,
	}, nil
}

// ReportFinishUpload 报告上传完成
func (s *VideoServiceService) ReportFinishUpload(ctx context.Context, req *pb.ReportFinishUploadReq) (*pb.ReportFinishUploadResp, error) {
	// 构建输入
	input := &biz.ReportFinishUploadInput{
		FileID: req.FileId,
	}

	// 调用业务层
	output, err := s.uc.ReportFinishUpload(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.ReportFinishUploadResp{
		Url: output.URL,
	}, nil
}

// ReportVideoFinishUpload 报告视频上传完成
func (s *VideoServiceService) ReportVideoFinishUpload(ctx context.Context, req *pb.ReportVideoFinishUploadReq) (*pb.ReportVideoFinishUploadResp, error) {
	// 获取当前用户ID
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, err
	}

	// 构建输入
	input := &biz.ReportVideoFinishUploadInput{
		FileID:      req.FileId,
		Title:       req.Title,
		CoverURL:    req.CoverUrl,
		Description: req.Description,
		VideoURL:    req.VideoUrl,
		UserID:      userID,
	}

	// 调用业务层
	output, err := s.uc.ReportVideoFinishUpload(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.ReportVideoFinishUploadResp{
		VideoId: output.VideoID,
	}, nil
}

// FeedShortVideo 视频流
func (s *VideoServiceService) FeedShortVideo(ctx context.Context, req *pb.FeedShortVideoReq) (*pb.FeedShortVideoResp, error) {
	// 获取当前用户ID
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		userID = 0 // 未登录用户
	}

	// 构建输入
	input := &biz.FeedVideoInput{
		LatestTime: req.LatestTime,
		UserID:     userID,
		FeedNum:    req.FeedNum,
	}

	// 调用业务层
	output, err := s.uc.FeedVideo(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换为proto结构
	pbVideos := make([]*pb.Video, 0, len(output.Videos))
	for _, video := range output.Videos {
		pbVideos = append(pbVideos, s.convertToPBVideo(video))
	}

	return &pb.FeedShortVideoResp{
		Videos:   pbVideos,
		NextTime: output.NextTime,
	}, nil
}

// GetVideoById 获取视频详情
func (s *VideoServiceService) GetVideoById(ctx context.Context, req *pb.GetVideoByIdReq) (*pb.GetVideoByIdResp, error) {
	// 获取当前用户ID
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		userID = 0 // 未登录用户
	}

	// 构建输入
	input := &biz.GetVideoInput{
		VideoID: req.VideoId,
		UserID:  userID,
	}

	// 调用业务层
	output, err := s.uc.GetVideo(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.GetVideoByIdResp{
		Video: s.convertToPBVideo(output.Video),
	}, nil
}

// ListPublishedVideo 获取已发布视频列表
func (s *VideoServiceService) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	// 获取当前用户ID
	currentUserID, err := claims.GetUserId(ctx)
	if err != nil {
		currentUserID = 0
	}

	// 构建输入
	input := &biz.ListPublishedVideoInput{
		UserID:   currentUserID,
		Page:     int64(req.PageStats.Page),
		PageSize: int64(req.PageStats.Size),
	}

	// 调用业务层
	output, err := s.uc.ListPublishedVideo(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换为proto结构
	pbVideos := make([]*pb.Video, 0, len(output.Videos))
	for _, video := range output.Videos {
		pbVideos = append(pbVideos, s.convertToPBVideo(video))
	}

	return &pb.ListPublishedVideoResp{
		VideoList: pbVideos,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

// convertToPBVideo 将业务层Video转换为proto Video
func (s *VideoServiceService) convertToPBVideo(video *biz.Video) *pb.Video {
	if video == nil {
		return nil
	}

	pbVideo := &pb.Video{
		Id:             video.ID,
		PlayUrl:        video.PlayURL,
		CoverUrl:       video.CoverURL,
		FavoriteCount:  video.FavoriteCount,
		CommentCount:   video.CommentCount,
		IsFavorite:     video.IsFavorite,
		Title:          video.Title,
		IsCollected:    video.IsCollected,
		CollectedCount: video.CollectedCount,
	}

	if video.Author != nil {
		pbVideo.Author = &pb.VideoAuthor{
			Id:          video.Author.ID,
			Name:        video.Author.Name,
			Avatar:      video.Author.Avatar,
			IsFollowing: video.Author.IsFollowing,
		}
	}

	return pbVideo
}
