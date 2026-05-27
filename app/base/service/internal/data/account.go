package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/data/model"
	"lehu-video/pkg/apperror"
	"strings"
	"time"
)

type accountRepo struct {
	data *Data
	log  *log.Helper
}

func NewAccountRepo(data *Data, logger log.Logger) biz.AccountRepo {
	return &accountRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (a *accountRepo) CreateAccount(ctx context.Context, in *biz.Account) error {
	account := model.Account{
		Id:        in.Id,
		Mobile:    in.Mobile,
		Email:     in.Email,
		Password:  in.Password,
		Salt:      in.Salt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := a.data.db.Table(model.Account{}.TableName()).Create(&account).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *accountRepo) CheckAccountUnique(ctx context.Context, in *biz.Account) error {
	// 判空检查
	if in.Mobile == "" && in.Email == "" {
		return apperror.InvalidArgument("手机号和邮箱不能同时为空")
	}

	account := model.Account{}
	query := a.data.db.Table(model.Account{}.TableName())

	// 构建查询条件
	var conditions []string
	var args []interface{}

	if in.Mobile != "" {
		conditions = append(conditions, "mobile = ?")
		args = append(args, in.Mobile)
	}

	if in.Email != "" {
		conditions = append(conditions, "email = ?")
		args = append(args, in.Email)
	}

	// 构建 WHERE 子句
	if len(conditions) == 0 {
		return nil
	}

	whereClause := strings.Join(conditions, " OR ")
	err := query.Where(whereClause, args...).First(&account).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	if err != nil {
		return apperror.Internal(err, "查询账号失败")
	}

	// 明确告知是手机号还是邮箱重复
	if in.Mobile != "" && account.Mobile == in.Mobile {
		return apperror.Conflict("手机号已被注册")
	}

	if in.Email != "" && account.Email == in.Email {
		return apperror.Conflict("邮箱已被注册")
	}

	return apperror.Conflict("账号已存在")
}

func (a *accountRepo) GetAccountById(ctx context.Context, id int64) (bool, *biz.Account, error) {
	account := model.Account{}
	err := a.data.db.Table(model.Account{}.TableName()).Where("id = ?", id).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, &biz.Account{
		Id:       account.Id,
		Mobile:   account.Mobile,
		Email:    account.Email,
		Password: account.Password,
		Salt:     account.Salt,
	}, nil
}

func (a *accountRepo) GetAccountByMobile(ctx context.Context, mobile string) (bool, *biz.Account, error) {
	account := model.Account{}
	err := a.data.db.Table(model.Account{}.TableName()).Where("mobile = ?", mobile).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, &biz.Account{
		Id:       account.Id,
		Mobile:   account.Mobile,
		Email:    account.Email,
		Password: account.Password,
		Salt:     account.Salt,
	}, nil
}

func (a *accountRepo) GetAccountByEmail(ctx context.Context, email string) (bool, *biz.Account, error) {
	account := model.Account{}
	err := a.data.db.Table(model.Account{}.TableName()).Where("email = ?", email).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, &biz.Account{
		Id:       account.Id,
		Mobile:   account.Mobile,
		Email:    account.Email,
		Password: account.Password,
		Salt:     account.Salt,
	}, nil
}

func (a *accountRepo) UpdateAccount(ctx context.Context, in *biz.Account) error {
	account := model.Account{
		Id:       in.Id,
		Mobile:   in.Mobile,
		Email:    in.Email,
		Password: in.Password,
		Salt:     in.Salt,
	}
	err := a.data.db.Table(model.Account{}.TableName()).Where("id = ?", account.Id).UpdateColumns(map[string]interface{}{
		"mobile":     account.Mobile,
		"email":      account.Email,
		"password":   account.Password,
		"salt":       account.Salt,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		return err
	}
	return nil
}
