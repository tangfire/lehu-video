package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
)

type followRepo struct {
	data *Data
	log  *log.Helper
}

func NewFollowRepo(data *Data, logger log.Logger) biz.FollowRepo {
	return &followRepo{data: data, log: log.NewHelper(logger)}
}

func (r *followRepo) CreateFollow(ctx context.Context, userId, targetUserId int64) error {
	follow := model.Follow{
		Id:           int64(uuid.New().ID()),
		UserId:       userId,
		TargetUserId: targetUserId,
		IsDeleted:    false,
	}

	return r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).Create(&follow).Error
}

func (r *followRepo) GetFollow(ctx context.Context, userId, targetUserId int64) (bool, int64, bool, error) {
	follow := model.Follow{}
	err := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).
		Where("user_id = ? AND target_user_id = ?", userId, targetUserId).
		First(&follow).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, 0, false, nil
	}
	if err != nil {
		return false, 0, false, err
	}

	return true, follow.Id, follow.IsDeleted, nil
}

func (r *followRepo) UpdateFollowStatus(ctx context.Context, followId int64, isDeleted bool) error {
	return r.data.db.WithContext(ctx).Table(model.Follow{}.TableName()).
		Where("id = ?", followId).
		Update("is_deleted", isDeleted).Error
}

func (r *followRepo) GetFollowsByCondition(ctx context.Context, condition map[string]interface{}) ([]biz.FollowData, error) {
	db := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName())
	db = r.applyConditions(db, condition)

	var follows []model.Follow
	if err := db.Find(&follows).Error; err != nil {
		return nil, err
	}

	// 转换为biz层结构
	result := make([]biz.FollowData, 0, len(follows))
	for _, f := range follows {
		result = append(result, biz.FollowData{
			ID:           f.Id,
			UserId:       f.UserId,
			TargetUserId: f.TargetUserId,
			IsDeleted:    f.IsDeleted,
		})
	}

	return result, nil
}

func (r *followRepo) CountFollowsByCondition(ctx context.Context, condition map[string]interface{}) (int64, error) {
	db := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName())
	db = r.applyConditions(db, condition)

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyConditions 应用查询条件 - 只做简单的条件转换
func (r *followRepo) applyConditions(db *gorm.DB, condition map[string]interface{}) *gorm.DB {
	for key, value := range condition {
		switch key {
		case "user_id":
			db = db.Where("user_id = ?", value)
		case "target_user_id":
			if ids, ok := value.([]int64); ok {
				db = db.Where("target_user_id IN (?)", ids)
			} else {
				db = db.Where("target_user_id = ?", value)
			}
		case "is_deleted":
			db = db.Where("is_deleted = ?", value)
		case "mutual_follow":
			// 这里实现互相关注的查询逻辑
			userId, _ := value.(int64)
			// 查询用户关注的人
			subquery := r.data.db.Session(&gorm.Session{}).
				Table(model.Follow{}.TableName()).
				Select("target_user_id").
				Where("user_id = ? AND is_deleted = ?", userId, false)

			// 查询这些人中哪些也关注了用户
			db = db.Where("user_id IN (?) AND target_user_id = ? AND is_deleted = ?", subquery, userId, false)
		case "limit":
			db = db.Limit(int(value.(int64)))
		case "offset":
			db = db.Offset(int(value.(int64)))
		default:
			r.log.Warnf("Unknown condition key: %s", key)
		}
	}

	return db
}
