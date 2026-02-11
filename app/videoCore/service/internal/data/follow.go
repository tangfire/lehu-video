package data

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"strconv"
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
			db = db.Limit(int(value.(int32)))
		case "offset":
			db = db.Offset(int(value.(int32)))
		default:
			r.log.Warnf("Unknown condition key: %s", key)
		}
	}

	return db
}

// 实现ListFollowing方法
func (r *followRepo) ListFollowing(ctx context.Context, userID string, followType int32, pageStats *biz.PageStats) ([]string, error) {
	userId, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("用户ID格式错误: %v", err)
	}

	// 构建查询条件
	condition := make(map[string]interface{})
	condition["is_deleted"] = false

	// 根据followType决定查询字段
	switch followType {
	case 0: // 关注的人（following）
		condition["user_id"] = userId
	case 1: // 粉丝（follower）
		condition["target_user_id"] = userId
	default:
		condition["user_id"] = userId
	}

	// 添加分页条件
	if pageStats != nil {
		condition["limit"] = pageStats.PageSize
		condition["offset"] = (pageStats.Page - 1) * pageStats.PageSize
	}

	// 查询数据
	followData, err := r.GetFollowsByCondition(ctx, condition)
	if err != nil {
		return nil, err
	}

	// 提取用户ID列表
	userIDs := make([]string, 0, len(followData))
	for _, data := range followData {
		if followType == 0 { // 关注的人 -> 提取target_user_id
			userIDs = append(userIDs, strconv.FormatInt(data.TargetUserId, 10))
		} else { // 粉丝 -> 提取user_id
			userIDs = append(userIDs, strconv.FormatInt(data.UserId, 10))
		}
	}

	return userIDs, nil
}

// GetFollowersPaginated 分页获取粉丝ID列表
func (r *followRepo) GetFollowersPaginated(ctx context.Context, userID string, offset, limit int) ([]string, int64, error) {
	userId, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("用户ID格式错误: %v", err)
	}

	condition := map[string]interface{}{
		"target_user_id": userId,
		"is_deleted":     false,
	}

	// 先获取总数
	db := r.data.db.WithContext(ctx).Table(model.Follow{}.TableName())
	db = db.Where("target_user_id = ? AND is_deleted = ?", userId, false)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 添加分页
	condition["limit"] = limit
	condition["offset"] = offset

	followData, err := r.GetFollowsByCondition(ctx, condition)
	if err != nil {
		return nil, 0, err
	}

	userIDs := make([]string, 0, len(followData))
	for _, data := range followData {
		userIDs = append(userIDs, strconv.FormatInt(data.UserId, 10))
	}
	return userIDs, total, nil
}

// GetFollowers 保留原有方法（但不推荐使用大V场景），内部调用分页，返回全部（警告）
func (r *followRepo) GetFollowers(ctx context.Context, userID string) ([]string, error) {
	// 为避免破坏现有调用，暂时返回全部（但建议后续迁移）
	// 这里实现为分批获取全部，避免OOM
	var all []string
	offset := 0
	limit := 1000
	for {
		page, total, err := r.GetFollowersPaginated(ctx, userID, offset, limit)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if int64(len(all)) >= total {
			break
		}
		offset += limit
	}
	return all, nil
}

// 实现CountFollowers方法
func (r *followRepo) CountFollowers(ctx context.Context, userID string) (int64, error) {
	userId, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("用户ID格式错误: %v", err)
	}

	condition := map[string]interface{}{
		"target_user_id": userId,
		"is_deleted":     false,
	}

	return r.CountFollowsByCondition(ctx, condition)
}
