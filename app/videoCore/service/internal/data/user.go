package data

import (
	"context"
	"errors"
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
		Avatar:          in.Avatar,
		BackgroundImage: in.BackgroundImage,
		Signature:       in.Signature,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err := r.data.db.Table(model.User{}.TableName()).Create(&user).Error
	if err != nil {
		return err
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
		Avatar:          in.Avatar,
		BackgroundImage: in.BackgroundImage,
		Signature:       in.Signature,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       time.Now(),
	}
	err := r.data.db.Table(model.User{}.TableName()).Updates(&user).Error
	if err != nil {
		return err
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
		return false, nil, err
	}
	ret := &biz.User{
		Id:              user.Id,
		AccountId:       user.AccountId,
		Mobile:          user.Mobile,
		Email:           user.Email,
		Name:            user.Name,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
	return true, ret, nil
}

func (r *userRepo) GetUserByAccountId(ctx context.Context, accountId int64) (bool, *biz.User, error) {
	user := model.User{}
	err := r.data.db.Table(model.User{}.TableName()).Where("account_id = ?", accountId).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	ret := &biz.User{
		Id:              user.Id,
		AccountId:       user.AccountId,
		Mobile:          user.Mobile,
		Email:           user.Email,
		Name:            user.Name,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
	return true, ret, nil
}

func (r *userRepo) GetUserByIdList(ctx context.Context, idList []int64) ([]*biz.User, error) {
	userList := make([]model.User, 0, len(idList))
	err := r.data.db.Table(model.User{}.TableName()).Where("id in (?)", idList).Find(&userList).Error
	if err != nil {
		return nil, err
	}
	var retList []*biz.User
	for _, user := range userList {
		retList = append(retList, &biz.User{
			Id:              user.Id,
			AccountId:       user.AccountId,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
		})
	}
	return retList, nil
}

// 新增：搜索用户实现
func (r *userRepo) SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*biz.User, int64, error) {
	var users []model.User
	var total int64

	db := r.data.db.Table(model.User{}.TableName())

	// 构建搜索条件：搜索用户名、昵称、手机号、邮箱
	if keyword != "" {
		searchPattern := "%" + strings.TrimSpace(keyword) + "%"
		db = db.Where("name LIKE ? OR nickname LIKE ? OR mobile LIKE ? OR email LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		r.log.Errorf("统计用户数量失败: %v", err)
		return nil, 0, err
	}

	// 分页查询
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&users).Error; err != nil {
		r.log.Errorf("搜索用户失败: %v", err)
		return nil, 0, err
	}

	// 转换结果
	result := make([]*biz.User, 0, len(users))
	for _, user := range users {
		result = append(result, &biz.User{
			Id:              user.Id,
			AccountId:       user.AccountId,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
		})
	}

	return result, total, nil
}
