package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type FollowRepo interface {
	AddFollow(ctx context.Context, userId, targetUserId int64) error
	GetFollow(ctx context.Context, userId, targetUserId int64) (bool, int64, error)
	ReFollow(ctx context.Context, followId int64) error
	RemoveFollow(ctx context.Context, userId, targetUserId int64) error
	CountFollow(ctx context.Context, userId int64, followType int32) (int64, error)
	GetFollowingListById(ctx context.Context, userId int64, followingIdList []int64) ([]int64, error)
	ListFollowing(ctx context.Context, userId int64, followType int32, pageStats *pb.PageStatsReq) ([]int64, error)
}

type FollowUsecase struct {
	repo FollowRepo
	log  *log.Helper
}

func NewFollowUsecase(repo FollowRepo, logger log.Logger) *FollowUsecase {
	return &FollowUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *FollowUsecase) AddFollow(ctx context.Context, req *pb.AddFollowReq) (*pb.AddFollowResp, error) {
	exist, followId, err := uc.repo.GetFollow(ctx, req.UserId, req.TargetUserId)
	if err != nil {
		return nil, err
	}
	if !exist {
		err := uc.repo.AddFollow(ctx, followId, req.TargetUserId)
		if err != nil {
			return nil, err
		}
		return &pb.AddFollowResp{Meta: utils.GetSuccessMeta()}, nil
	}
	err = uc.repo.ReFollow(ctx, followId)
	if err != nil {
		return nil, err
	}
	return &pb.AddFollowResp{Meta: utils.GetSuccessMeta()}, nil
}

func (uc *FollowUsecase) RemoveFollow(ctx context.Context, req *pb.RemoveFollowReq) (*pb.RemoveFollowResp, error) {
	err := uc.repo.RemoveFollow(ctx, req.UserId, req.TargetUserId)
	if err != nil {
		return nil, err
	}
	return &pb.RemoveFollowResp{Meta: utils.GetSuccessMeta()}, nil
}

func (uc *FollowUsecase) CountFollow(ctx context.Context, req *pb.CountFollowReq) (*pb.CountFollowResp, error) {
	followingNum, err := uc.repo.CountFollow(ctx, req.UserId, int32(pb.FollowType_FOLLOWING))
	if err != nil {
		return nil, err
	}
	followerNum, err := uc.repo.CountFollow(ctx, req.UserId, int32(pb.FollowType_FOLLOWER))
	if err != nil {
		return nil, err
	}
	return &pb.CountFollowResp{
		Meta:           utils.GetSuccessMeta(),
		FollowingCount: followingNum,
		FollowerCount:  followerNum,
	}, nil
}

func (uc *FollowUsecase) IsFollowing(ctx context.Context, req *pb.IsFollowingReq) (*pb.IsFollowingResp, error) {
	followingIdList, err := uc.repo.GetFollowingListById(ctx, req.UserId, req.TargetUserIdList)
	if err != nil {
		return nil, err
	}
	return &pb.IsFollowingResp{
		Meta:          utils.GetSuccessMeta(),
		FollowingList: followingIdList,
	}, nil
}

func (uc *FollowUsecase) ListFollowing(ctx context.Context, req *pb.ListFollowingReq) (*pb.ListFollowingResp, error) {
	userIdList, err := uc.repo.ListFollowing(ctx, req.UserId, int32(req.FollowType), req.PageStats)
	if err != nil {
		return nil, err
	}
	total, err := uc.repo.CountFollow(ctx, req.UserId, int32(req.FollowType))
	if err != nil {
		return nil, err
	}
	return &pb.ListFollowingResp{
		Meta:       utils.GetSuccessMeta(),
		UserIdList: userIdList,
		PageStats:  &pb.PageStatsResp{Total: int32(total)},
	}, nil

}
