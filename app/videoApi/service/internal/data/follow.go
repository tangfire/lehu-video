package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *CoreAdapterImpl) CountFollow4User(ctx context.Context, userId string) ([]int64, error) {
	resp, err := r.follow.CountFollow(ctx, &core.CountFollowReq{UserId: userId})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	return []int64{resp.FollowingCount, resp.FollowerCount}, nil
}

func (r *CoreAdapterImpl) IsFollowing(ctx context.Context, userId string, targetUserIdList []string) (map[string]bool, error) {
	resp, err := r.follow.IsFollowing(ctx, &core.IsFollowingReq{
		UserId:           userId,
		TargetUserIdList: targetUserIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool)
	if len(resp.FollowingList) == 0 {
		return result, nil
	}

	for _, item := range resp.FollowingList {
		result[item] = true
	}
	return result, nil
}

func (r *CoreAdapterImpl) AddFollow(ctx context.Context, userId, targetUserId string) error {
	resp, err := r.follow.AddFollow(ctx, &core.AddFollowReq{
		UserId:       userId,
		TargetUserId: targetUserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
func (r *CoreAdapterImpl) RemoveFollow(ctx context.Context, userId, targetUserId string) error {
	resp, err := r.follow.RemoveFollow(ctx, &core.RemoveFollowReq{
		UserId:       userId,
		TargetUserId: targetUserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) ListFollow(ctx context.Context, userId string, _type *biz.FollowType, pageStats *biz.PageStats) (int64, []string, error) {
	followType := core.FollowType(*_type)
	resp, err := r.follow.ListFollowing(ctx, &core.ListFollowingReq{
		UserId:     userId,
		FollowType: followType,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	return int64(resp.PageStats.Total), resp.UserIdList, nil
}
