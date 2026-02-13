package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, in *biz.User) error {
	user := model.User{
		Id:              in.Id,
		AccountId:       in.AccountId,
		Mobile:          in.Mobile,
		Email:           in.Email,
		Name:            in.Name,
		Nickname:        in.Nickname,
		Avatar:          in.Avatar,
		BackgroundImage: in.BackgroundImage,
		Signature:       in.Signature,
		Gender:          in.Gender,
		FollowCount:     in.FollowCount,
		FollowerCount:   in.FollowerCount,
		TotalFavorited:  in.TotalFavorited,
		WorkCount:       in.WorkCount,
		FavoriteCount:   in.FavoriteCount,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := r.data.db.Table(model.User{}.TableName()).Create(&user).Error
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateUser(ctx context.Context, in *biz.User) error {
	user := model.User{
		Id:              in.Id,
		AccountId:       in.AccountId,
		Mobile:          in.Mobile,
		Email:           in.Email,
		Name:            in.Name,
		Nickname:        in.Nickname,
		Avatar:          in.Avatar,
		BackgroundImage: in.BackgroundImage,
		Signature:       in.Signature,
		Gender:          in.Gender,
		FollowCount:     in.FollowCount,
		FollowerCount:   in.FollowerCount,
		TotalFavorited:  in.TotalFavorited,
		WorkCount:       in.WorkCount,
		FavoriteCount:   in.FavoriteCount,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	err := r.data.db.Table(model.User{}.TableName()).Updates(&user).Error
	if err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateUserStats(ctx context.Context, userId int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	updates["updated_at"] = time.Now()

	err := r.data.db.Table(model.User{}.TableName()).
		Where("id = ?", userId).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("更新用户统计失败: %w", err)
	}
	return nil
}

func (r *userRepo) GetUserById(ctx context.Context, id int64) (bool, *biz.User, error) {
	user := model.User{}
	err := r.data.db.Table(model.User{}.TableName()).Where("id = ?", id).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return true, convertToBizUser(&user), nil
}

func (r *userRepo) GetUserByAccountId(ctx context.Context, accountId int64) (bool, *biz.User, error) {
	user := model.User{}
	err := r.data.db.Table(model.User{}.TableName()).Where("account_id = ?", accountId).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return true, convertToBizUser(&user), nil
}

func (r *userRepo) GetUserByIdList(ctx context.Context, idList []int64) ([]*biz.User, error) {
	if len(idList) == 0 {
		return []*biz.User{}, nil
	}

	userList := make([]model.User, 0)
	err := r.data.db.Table(model.User{}.TableName()).Where("id IN (?)", idList).Find(&userList).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询用户失败: %w", err)
	}

	result := make([]*biz.User, 0, len(userList))
	for _, user := range userList {
		result = append(result, convertToBizUser(&user))
	}
	return result, nil
}

func (r *userRepo) SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*biz.User, int64, error) {
	db := r.data.db.Table(model.User{}.TableName())

	// 构建搜索条件
	if keyword != "" {
		searchPattern := "%" + strings.TrimSpace(keyword) + "%"
		db = db.Where("name LIKE ? OR nickname LIKE ? OR mobile LIKE ? OR email LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户数量失败: %w", err)
	}

	// 分页查询
	userList := make([]model.User, 0)
	err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&userList).Error
	if err != nil {
		return nil, 0, fmt.Errorf("搜索用户失败: %w", err)
	}

	// 转换结果
	result := make([]*biz.User, 0, len(userList))
	for _, user := range userList {
		result = append(result, convertToBizUser(&user))
	}

	return result, total, nil
}

// 转换函数
func convertToBizUser(user *model.User) *biz.User {
	if user == nil {
		return nil
	}

	return &biz.User{
		Id:              user.Id,
		AccountId:       user.AccountId,
		Mobile:          user.Mobile,
		Email:           user.Email,
		Name:            user.Name,
		Nickname:        user.Nickname,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		Gender:          user.Gender,
		FollowCount:     user.FollowCount,
		FollowerCount:   user.FollowerCount,
		TotalFavorited:  user.TotalFavorited,
		WorkCount:       user.WorkCount,
		FavoriteCount:   user.FavoriteCount,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}
