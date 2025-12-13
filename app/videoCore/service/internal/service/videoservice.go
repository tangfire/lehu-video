package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	pb "lehu-video/api/videoCore/service/v1"
)

type VideoServiceService struct {
	pb.UnimplementedVideoServiceServer

	uc *biz.VideoUsecase
}

func NewVideoServiceService(uc *biz.VideoUsecase) *VideoServiceService {
	return &VideoServiceService{uc: uc}
}

func (s *VideoServiceService) FeedShortVideo(ctx context.Context, req *pb.FeedShortVideoReq) (*pb.FeedShortResp, error) {
	return &pb.FeedShortResp{}, nil
}
func (s *VideoServiceService) GetVideoById(ctx context.Context, req *pb.GetVideoByIdReq) (*pb.GetVideoByIdResp, error) {
	return s.uc.GetVideoById(ctx, req)
}
func (s *VideoServiceService) PublishVideo(ctx context.Context, req *pb.PublishVideoReq) (*pb.PublishVideoResp, error) {
	return s.uc.PublishVideo(ctx, req)
}
func (s *VideoServiceService) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	return s.uc.ListPublishedVideo(ctx, req)
}
func (s *VideoServiceService) GetVideoByIdList(ctx context.Context, req *pb.GetVideoByIdListReq) (*pb.GetVideoByIdListResp, error) {
	return &pb.GetVideoByIdListResp{}, nil
}
