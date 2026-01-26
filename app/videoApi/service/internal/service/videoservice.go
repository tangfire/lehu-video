package service

import (
	"context"

	pb "lehu-video/api/videoApi/service/v1"
)

type VideoServiceService struct {
	pb.UnimplementedVideoServiceServer
}

func NewVideoServiceService() *VideoServiceService {
	return &VideoServiceService{}
}

func (s *VideoServiceService) PreSign4UploadVideo(ctx context.Context, req *pb.PreSign4UploadVideoReq) (*pb.PreSign4UploadVideoResp, error) {
	return &pb.PreSign4UploadVideoResp{}, nil
}
func (s *VideoServiceService) PreSign4UploadCover(ctx context.Context, req *pb.PreSign4UploadCoverReq) (*pb.PreSign4UploadCoverResp, error) {
	return &pb.PreSign4UploadCoverResp{}, nil
}
func (s *VideoServiceService) ReportFinishUpload(ctx context.Context, req *pb.ReportFinishUploadReq) (*pb.ReportFinishUploadResp, error) {
	return &pb.ReportFinishUploadResp{}, nil
}
func (s *VideoServiceService) ReportVideoFinishUpload(ctx context.Context, req *pb.ReportVideoFinishUploadReq) (*pb.ReportVideoFinishUploadResp, error) {
	return &pb.ReportVideoFinishUploadResp{}, nil
}
func (s *VideoServiceService) FeedShortVideo(ctx context.Context, req *pb.FeedShortVideoReq) (*pb.FeedShortVideoResp, error) {
	return &pb.FeedShortVideoResp{}, nil
}
func (s *VideoServiceService) GetVideoById(ctx context.Context, req *pb.GetVideoByIdReq) (*pb.GetVideoByIdResp, error) {
	return &pb.GetVideoByIdResp{}, nil
}
func (s *VideoServiceService) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	return &pb.ListPublishedVideoResp{}, nil
}
