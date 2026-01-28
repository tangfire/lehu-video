package service

import (
	"context"
	"lehu-video/app/videoApi/service/internal/biz"

	pb "lehu-video/api/videoApi/service/v1"
)

type FollowServiceService struct {
	pb.UnimplementedFollowServiceServer

	uc *biz.FollowUsecase
}

func NewFollowServiceService(uc *biz.FollowUsecase) *FollowServiceService {
	return &FollowServiceService{uc: uc}
}

func (s *FollowServiceService) AddFollow(ctx context.Context, req *pb.AddFollowReq) (*pb.AddFollowResp, error) {
	err := s.uc.AddFollow(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.AddFollowResp{}, nil
}
func (s *FollowServiceService) RemoveFollow(ctx context.Context, req *pb.RemoveFollowReq) (*pb.RemoveFollowResp, error) {
	err := s.uc.RemoveFollow(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.RemoveFollowResp{}, nil
}
func (s *FollowServiceService) ListFollowing(ctx context.Context, req *pb.ListFollowingReq) (*pb.ListFollowingResp, error) {
	followType := biz.FollowType(req.Type)
	input := &biz.ListFollowingInput{
		UserId: req.UserId,
		Type:   &followType,
		PageStats: &biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}
	output, err := s.uc.ListFollowing(ctx, input)
	if err != nil {
		return nil, err
	}
	var retFollowUsers []*pb.FollowUser
	for _, followUser := range output.Users {
		retFollowUsers = append(retFollowUsers, &pb.FollowUser{
			Id:          followUser.Id,
			Name:        followUser.Name,
			Avatar:      followUser.Avatar,
			IsFollowing: followUser.IsFollowing,
		})
	}
	return &pb.ListFollowingResp{
		Users: retFollowUsers,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}
