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
	// 1. 查询关注数量（用户关注的人）
	followingCondition := map[string]interface{}{
		"user_id":    query.UserId,
		"is_deleted": false,
	}
	followingCount, err := uc.repo.CountFollowsByCondition(ctx, followingCondition)
	if err != nil {
		return nil, err
	}

	// 2. 查询粉丝数量（关注用户的人）
	followerCondition := map[string]interface{}{
		"target_user_id": query.UserId,
		"is_deleted":     false,
	}
	followerCount, err := uc.repo.CountFollowsByCondition(ctx, followerCondition)
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

	// 构建查询条件
	condition := map[string]interface{}{
		"user_id":        query.UserId,
		"target_user_id": query.TargetUserIdList,
		"is_deleted":     false,
	}

	// 查询关注列表
	follows, err := uc.repo.GetFollowsByCondition(ctx, condition)
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
	// 1. 根据followType构建查询条件
	condition := uc.buildListCondition(query.UserId, query.FollowType)

	// 2. 查询总数
	total, err := uc.repo.CountFollowsByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}

	// 3. 添加分页条件
	condition["limit"] = query.PageStats.PageSize
	condition["offset"] = (query.PageStats.Page - 1) * query.PageStats.PageSize

	// 4. 查询分页数据
	follows, err := uc.repo.GetFollowsByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}

	// 5. 提取用户ID列表
	userIdList := uc.extractUserIds(follows, query.FollowType)

	return &ListFollowingResult{
		UserIdList: userIdList,
		Total:      total,
	}, nil
}

// buildListCondition 构建查询条件 - 业务逻辑
func (uc *FollowUsecase) buildListCondition(userId int64, followType int32) map[string]interface{} {
	condition := map[string]interface{}{
		"is_deleted": false,
	}

	switch followType {
	case 0: // 关注的人
		condition["user_id"] = userId
	case 1: // 粉丝
		condition["target_user_id"] = userId
	case 2: // 互相关注
		// 这个复杂逻辑应该放在biz层，但为了性能，这里简化处理
		// 实际业务中可能需要更复杂的处理
		condition["mutual_follow"] = userId
	default:
		// 默认查关注的人
		condition["user_id"] = userId
	}

	return condition
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
