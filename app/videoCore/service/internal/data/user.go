package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"time"
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
	userList := make([]model.User, len(idList))
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
