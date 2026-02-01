package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type FollowUser struct {
	Id          string
	Name        string
	Avatar      string
	IsFollowing bool
}

type ListFollowingInput struct {
	UserId    string // 暂时不传
	Type      *FollowType
	PageStats *PageStats
}

type ListFollowingOutput struct {
	Users []*FollowUser
	Total int64
}

type FollowUsecase struct {
	core CoreAdapter
	log  *log.Helper
}

func NewFollowUsecase(core CoreAdapter, logger log.Logger) *FollowUsecase {
	return &FollowUsecase{
		core: core,
		log:  log.NewHelper(logger),
	}
}

func (uc *FollowUsecase) AddFollow(ctx context.Context, targetUserId string) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}
	err = uc.core.AddFollow(ctx, userId, targetUserId)
	if err != nil {
		return errors.New("操作失败")
	}
	return nil
}

func (uc *FollowUsecase) RemoveFollow(ctx context.Context, targetUserId string) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}
	err = uc.core.RemoveFollow(ctx, userId, targetUserId)
	if err != nil {
		return errors.New("操作失败")
	}
	return nil
}

func (uc *FollowUsecase) ListFollowing(ctx context.Context, input *ListFollowingInput) (*ListFollowingOutput, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	total, followUserIds, err := uc.core.ListFollow(ctx, userId, input.Type, input.PageStats)
	if err != nil {
		return nil, errors.New("获取列表失败")
	}

	userInfos, err := uc.core.GetUserInfoByIdList(ctx, followUserIds)
	if err != nil {
		log.Context(ctx).Warnf("failed to get user info by id list: %v", err)
	}
	userInfoMap := make(map[string]*UserInfo)
	for _, userInfo := range userInfos {
		userInfoMap[userInfo.Id] = userInfo
	}

	var result []*FollowUser
	for _, id := range followUserIds {
		userInfo := userInfoMap[id]
		if userInfo == nil {
			continue
		}
		result = append(result, &FollowUser{
			Id:          userInfo.Id,
			Name:        userInfo.Name,
			Avatar:      userInfo.Avatar,
			IsFollowing: true,
		})
	}

	return &ListFollowingOutput{
		Users: result,
		Total: total,
	}, nil

}
