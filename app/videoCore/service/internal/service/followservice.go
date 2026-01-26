package service

import (
	"context"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
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
	// ✅ 改为Command
	cmd := &biz.AddFollowCommand{
		UserId:       req.UserId,
		TargetUserId: req.TargetUserId,
	}

	_, err := s.uc.AddFollow(ctx, cmd)
	if err != nil {
		return &pb.AddFollowResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.AddFollowResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FollowServiceService) RemoveFollow(ctx context.Context, req *pb.RemoveFollowReq) (*pb.RemoveFollowResp, error) {
	// ✅ 改为Command
	cmd := &biz.RemoveFollowCommand{
		UserId:       req.UserId,
		TargetUserId: req.TargetUserId,
	}

	_, err := s.uc.RemoveFollow(ctx, cmd)
	if err != nil {
		return &pb.RemoveFollowResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RemoveFollowResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *FollowServiceService) ListFollowing(ctx context.Context, req *pb.ListFollowingReq) (*pb.ListFollowingResp, error) {
	// ✅ 改为Query
	query := &biz.ListFollowingQuery{
		UserId:     req.UserId,
		FollowType: int32(req.FollowType),
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListFollowing(ctx, query)
	if err != nil {
		return &pb.ListFollowingResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.ListFollowingResp{
		Meta:       utils.GetSuccessMeta(),
		UserIdList: result.UserIdList,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *FollowServiceService) IsFollowing(ctx context.Context, req *pb.IsFollowingReq) (*pb.IsFollowingResp, error) {
	// ✅ 改为Query
	query := &biz.IsFollowingQuery{
		UserId:           req.UserId,
		TargetUserIdList: req.TargetUserIdList,
	}

	result, err := s.uc.IsFollowing(ctx, query)
	if err != nil {
		return &pb.IsFollowingResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.IsFollowingResp{
		Meta:          utils.GetSuccessMeta(),
		FollowingList: result.FollowingList,
	}, nil
}

func (s *FollowServiceService) CountFollow(ctx context.Context, req *pb.CountFollowReq) (*pb.CountFollowResp, error) {
	// ✅ 改为Query
	query := &biz.CountFollowQuery{
		UserId: req.UserId,
	}

	result, err := s.uc.CountFollow(ctx, query)
	if err != nil {
		return &pb.CountFollowResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CountFollowResp{
		Meta:           utils.GetSuccessMeta(),
		FollowingCount: result.FollowingCount,
		FollowerCount:  result.FollowerCount,
	}, nil
}
