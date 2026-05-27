package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

// ✅ Command/Query模式
type AddFollowCommand struct {
	UserId       int64
	TargetUserId int64
}

type AddFollowResult struct{}

type RemoveFollowCommand struct {
	UserId       int64
	TargetUserId int64
}

type RemoveFollowResult struct{}

type CountFollowQuery struct {
	UserId int64
}

type CountFollowResult struct {
	FollowingCount int64
	FollowerCount  int64
}

type IsFollowingQuery struct {
	UserId           int64
	TargetUserIdList []int64
}

type IsFollowingResult struct {
	FollowingList []int64
}

type ListFollowingQuery struct {
	UserId     int64
	FollowType int32 // 0: following, 1: follower, 2: both
	PageStats  PageStats
}

type ListFollowingResult struct {
	UserIdList []int64
	Total      int64
}

// FollowRepo 数据层接口 - 只做简单的CRUD
type FollowRepo interface {
	// 基础CRUD
	CreateFollow(ctx context.Context, userId, targetUserId int64) error
	GetFollow(ctx context.Context, userId, targetUserId int64) (bool, int64, bool, error)
	UpdateFollowStatus(ctx context.Context, followId int64, isDeleted bool) error

	// 简单的查询
	GetFollowsByCondition(ctx context.Context, condition map[string]interface{}) ([]FollowData, error)
	CountFollowsByCondition(ctx context.Context, condition map[string]interface{}) (int64, error)
	CountFollowing(ctx context.Context, userId int64) (int64, error)
	CountFollower(ctx context.Context, userId int64) (int64, error)
	BatchGetFollowing(ctx context.Context, userId int64, targetUserIds []int64) ([]FollowData, error)
	ListRelations(ctx context.Context, query FollowListQuery) ([]FollowData, int64, error)

	ListFollowing(ctx context.Context, userID string, followType int32, pageStats *PageStats) ([]string, error)
	GetFollowers(ctx context.Context, userID string) ([]string, error)
	GetFollowersPaginated(ctx context.Context, userID string, offset, limit int) ([]string, int64, error)
	CountFollowers(ctx context.Context, userID string) (int64, error)
}

// FollowData 从数据层返回的数据结构
type FollowData struct {
	ID           int64
	UserId       int64
	TargetUserId int64
	IsDeleted    bool
}

type FollowListQuery struct {
	UserId     int64
	FollowType int32
	Page       int32
	PageSize   int32
}

type FollowUsecase struct {
	repo FollowRepo
	log  *log.Helper
}

func NewFollowUsecase(repo FollowRepo, logger log.Logger) *FollowUsecase {
	return &FollowUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *FollowUsecase) AddFollow(ctx context.Context, cmd *AddFollowCommand) (*AddFollowResult, error) {
	// 1. 检查是否已存在关注关系
	exist, followId, isDeleted, err := uc.repo.GetFollow(ctx, cmd.UserId, cmd.TargetUserId)
	if err != nil {
		return nil, err
	}

	if !exist {
		// 2. 不存在则创建
		err := uc.repo.CreateFollow(ctx, cmd.UserId, cmd.TargetUserId)
		if err != nil {
			return nil, err
		}
		return &AddFollowResult{}, nil
	}

	// 3. 存在但已删除，则重新激活
	if isDeleted {
		err = uc.repo.UpdateFollowStatus(ctx, followId, false)
		if err != nil {
			return nil, err
		}
	}
	// 4. 已存在且未删除，什么都不做（幂等）

	return &AddFollowResult{}, nil
}

func (uc *FollowUsecase) RemoveFollow(ctx context.Context, cmd *RemoveFollowCommand) (*RemoveFollowResult, error) {
	// 1. 检查是否存在关注关系
	exist, followId, isDeleted, err := uc.repo.GetFollow(ctx, cmd.UserId, cmd.TargetUserId)
	if err != nil {
		return nil, err
	}

	if !exist || isDeleted {
		// 不存在或已删除，直接返回成功（幂等）
		return &RemoveFollowResult{}, nil
	}

	// 2. 标记为删除
	err = uc.repo.UpdateFollowStatus(ctx, followId, true)
	if err != nil {
		return nil, err
	}

	return &RemoveFollowResult{}, nil
}

func (uc *FollowUsecase) CountFollow(ctx context.Context, query *CountFollowQuery) (*CountFollowResult, error) {
	followingCount, err := uc.repo.CountFollowing(ctx, query.UserId)
	if err != nil {
		return nil, err
	}

	followerCount, err := uc.repo.CountFollower(ctx, query.UserId)
	if err != nil {
		return nil, err
	}

	return &CountFollowResult{
		FollowingCount: followingCount,
		FollowerCount:  followerCount,
	}, nil
}

func (uc *FollowUsecase) IsFollowing(ctx context.Context, query *IsFollowingQuery) (*IsFollowingResult, error) {
	if len(query.TargetUserIdList) == 0 {
		return &IsFollowingResult{FollowingList: []int64{}}, nil
	}

	follows, err := uc.repo.BatchGetFollowing(ctx, query.UserId, query.TargetUserIdList)
	if err != nil {
		return nil, err
	}

	// 提取已关注的用户ID
	followingList := make([]int64, 0, len(follows))
	for _, follow := range follows {
		followingList = append(followingList, follow.TargetUserId)
	}

	return &IsFollowingResult{
		FollowingList: followingList,
	}, nil
}

func (uc *FollowUsecase) ListFollowing(ctx context.Context, query *ListFollowingQuery) (*ListFollowingResult, error) {
	follows, total, err := uc.repo.ListRelations(ctx, FollowListQuery{
		UserId:     query.UserId,
		FollowType: query.FollowType,
		Page:       query.PageStats.Page,
		PageSize:   query.PageStats.PageSize,
	})
	if err != nil {
		return nil, err
	}

	userIdList := uc.extractUserIds(follows, query.FollowType)

	return &ListFollowingResult{
		UserIdList: userIdList,
		Total:      total,
	}, nil
}

// extractUserIds 提取用户ID - 业务逻辑
func (uc *FollowUsecase) extractUserIds(follows []FollowData, followType int32) []int64 {
	userIds := make([]int64, 0, len(follows))

	for _, follow := range follows {
		switch followType {
		case 0: // 关注的人 -> 取target_user_id
			userIds = append(userIds, follow.TargetUserId)
		case 1: // 粉丝 -> 取user_id
			userIds = append(userIds, follow.UserId)
		case 2: // 互相关注 -> 取user_id
			userIds = append(userIds, follow.UserId)
		}
	}

	return userIds
}
