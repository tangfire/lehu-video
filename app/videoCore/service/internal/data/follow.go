package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type followRepo struct {
	data *Data
	log  *log.Helper
}

func NewFollowRepo(data *Data, logger log.Logger) biz.FollowRepo {
	return &followRepo{data: data, log: log.NewHelper(logger)}
}

func (r *followRepo) AddFollow(ctx context.Context, userId, targetUserId int64) error {

	follow := model.Follow{
		Id:           int64(uuid.New().ID()),
		UserId:       userId,
		TargetUserId: targetUserId,
	}
	err := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).Create(&follow).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *followRepo) GetFollow(ctx context.Context, userId, targetUserId int64) (bool, int64, error) {
	follow := model.Follow{}
	err := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).Where("user_id = ? AND target_user_id = ?", userId, targetUserId).First(&follow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, follow.Id, nil
}

func (r *followRepo) ReFollow(ctx context.Context, followId int64) error {
	err := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).
		Where("id = ?", followId).
		UpdateColumns(map[string]interface{}{
			"is_deleted": false,
		}).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *followRepo) RemoveFollow(ctx context.Context, userId, targetUserId int64) error {
	err := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).
		Where("user_id = ? AND target_user_id = ?", userId, targetUserId).
		UpdateColumns(map[string]interface{}{
			"is_deleted": true,
		}).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *followRepo) parseFollowType(db *gorm.DB, followType int32, userId int64) (*gorm.DB, error) {
	switch followType {
	case 0:
		// 用户关注的人（关注列表）
		return db.Where("user_id = ?", userId), nil
	case 1:
		// 关注用户的人（粉丝列表）
		return db.Where("target_user_id = ?", userId), nil
	case 2:
		// 互相关注（好友列表）
		// 先找到用户关注的人，再找出这些人中哪些也关注了用户
		subquery := db.Session(&gorm.Session{}).
			Table(model.Follow{}.TableName()).
			Select("target_user_id").
			Where("user_id = ?", userId)

		return db.Where("user_id IN (?) AND target_user_id = ?", subquery, userId), nil
	}
	return nil, errors.New("invalid follow type")
}

func (r *followRepo) ListFollowing(ctx context.Context, userId int64, followType int32, pageStats *pb.PageStatsReq) ([]int64, error) {
	db := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName())
	db, err := r.parseFollowType(db, followType, userId)
	if err != nil {
		return nil, err
	}

	var followList []model.Follow
	offset := int((pageStats.Page - 1) * pageStats.Size)
	limit := int(pageStats.Size)

	err = db.Offset(offset).Limit(limit).Find(&followList).Error
	if err != nil {
		return nil, err
	}

	// 根据 followType 提取对应的用户ID
	result := make([]int64, 0, len(followList))

	switch followType {
	case 0:
		// 用户关注的人（关注列表）- 提取 target_user_id
		for _, follow := range followList {
			result = append(result, follow.TargetUserId)
		}
	case 1:
		// 关注用户的人（粉丝列表）- 提取 user_id
		for _, follow := range followList {
			result = append(result, follow.UserId)
		}
	case 2:
		// 互相关注（好友列表）- 提取 user_id（即用户的粉丝中，用户也关注的人）
		for _, follow := range followList {
			result = append(result, follow.UserId)
		}
	default:
		return nil, errors.New("invalid follow type")
	}

	return result, nil
}

func (r *followRepo) CountFollow(ctx context.Context, userId int64, followType int32) (int64, error) {
	db := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName())
	db, err := r.parseFollowType(db, followType, userId)
	if err != nil {
		return 0, err
	}
	var total int64
	err = db.Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *followRepo) GetFollowingListById(ctx context.Context, userId int64, followingIdList []int64) ([]int64, error) {
	var followList []model.Follow
	err := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).Select("target_user_id").
		Where("user_id = ?", userId).
		Where("target_user_id in (?)", followingIdList).
		Find(&followList).Error
	if err != nil {
		return nil, err
	}
	followingIdList = utils.Slice2Slice(followList, func(follow model.Follow) int64 {
		return follow.TargetUserId
	})
	return followingIdList, nil
}
