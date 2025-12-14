package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/data/model"
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
		Id:       in.Id,
		Mobile:   in.Mobile,
		Email:    in.Email,
		Password: in.Password,
		Salt:     in.Salt,
		CreateAt: time.Now(),
		UpdateAt: time.Now(),
	}
	err := a.data.db.Table(model.Account{}.TableName()).Create(&account).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *accountRepo) CheckAccountUnique(ctx context.Context, in *biz.Account) error {
	account := model.Account{}
	err := a.data.db.Table(model.Account{}.TableName()).Where("mobile = ? or email = ?", in.Mobile, in.Email).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("account not unique")
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
		"mobile": account.Mobile,
		"email":  account.Email,
	}).Error
	if err != nil {
		return err
	}
	return nil
}
