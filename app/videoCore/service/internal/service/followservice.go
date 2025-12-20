package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	pb "lehu-video/api/videoCore/service/v1"
)

type FollowServiceService struct {
	pb.UnimplementedFollowServiceServer

	uc *biz.FollowUsecase
}

func NewFollowServiceService(uc *biz.FollowUsecase) *FollowServiceService {
	return &FollowServiceService{
		uc: uc,
	}
}

func (s *FollowServiceService) AddFollow(ctx context.Context, req *pb.AddFollowReq) (*pb.AddFollowResp, error) {
	return s.uc.AddFollow(ctx, req)
}
func (s *FollowServiceService) RemoveFollow(ctx context.Context, req *pb.RemoveFollowReq) (*pb.RemoveFollowResp, error) {
	return s.uc.RemoveFollow(ctx, req)
}
func (s *FollowServiceService) ListFollowing(ctx context.Context, req *pb.ListFollowingReq) (*pb.ListFollowingResp, error) {
	return s.uc.ListFollowing(ctx, req)
}
func (s *FollowServiceService) IsFollowing(ctx context.Context, req *pb.IsFollowingReq) (*pb.IsFollowingResp, error) {
	return s.uc.IsFollowing(ctx, req)
}
func (s *FollowServiceService) CountFollow(ctx context.Context, req *pb.CountFollowReq) (*pb.CountFollowResp, error) {
	return s.uc.CountFollow(ctx, req)
}
