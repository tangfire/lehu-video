package service

import (
	"context"

	pb "lehu-video/api/videoCore/service/v1"
)

type FollowServiceService struct {
	pb.UnimplementedFollowServiceServer
}

func NewFollowServiceService() *FollowServiceService {
	return &FollowServiceService{}
}

func (s *FollowServiceService) AddFollow(ctx context.Context, req *pb.AddFollowReq) (*pb.AddFollowResp, error) {
	return &pb.AddFollowResp{}, nil
}
func (s *FollowServiceService) RemoveFollow(ctx context.Context, req *pb.RemoveFollowReq) (*pb.RemoveFollowResp, error) {
	return &pb.RemoveFollowResp{}, nil
}
func (s *FollowServiceService) ListFollowing(ctx context.Context, req *pb.ListFollowingReq) (*pb.ListFollowingResp, error) {
	return &pb.ListFollowingResp{}, nil
}
func (s *FollowServiceService) IsFollowing(ctx context.Context, req *pb.IsFollowingReq) (*pb.IsFollowingResp, error) {
	return &pb.IsFollowingResp{}, nil
}
func (s *FollowServiceService) CountFollow(ctx context.Context, req *pb.CountFollowReq) (*pb.CountFollowResp, error) {
	return &pb.CountFollowResp{}, nil
}
