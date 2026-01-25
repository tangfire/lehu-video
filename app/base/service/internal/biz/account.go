package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"lehu-video/app/base/service/internal/pkg/utils"
)

const (
	AccountPasswordPattern = "^[A-Za-z\\d\\S]{8,}"
	ErrInvalidPassword     = "密码由大小写字母、数字、符号组成，且至少需要8位"
)

type VoucherType int32

const (
	VoucherTypeEmail VoucherType = 0
	VoucherTypePhone VoucherType = 1
)

// ✅ 保持Request/Response结构，但移除Success字段
// （因为err == nil本身就代表success）

type RegisterRequest struct {
	Mobile   string
	Email    string
	Password string
}

type RegisterResponse struct {
	AccountId int64
}

type CheckAccountRequest struct {
	AccountId int64
	Mobile    string
	Email     string
	Password  string
}

type CheckAccountResponse struct {
	AccountId int64
}

type BindRequest struct {
	AccountId   int64
	VoucherType VoucherType
	Voucher     string
}

type BindResponse struct{}

type UnbindRequest struct {
	AccountId   int64
	VoucherType VoucherType
}

type UnbindResponse struct{}

type Account struct {
	Id       int64
	Mobile   string
	Email    string
	Password string
	Salt     string
}

func (a *Account) IsPasswordValid(patterns ...string) bool {
	if len(patterns) > 0 {
		pattern := patterns[0]
		return utils.IsValidWithRegex(pattern, a.Password)
	}
	return utils.IsValidWithRegex(AccountPasswordPattern, a.Password)
}

func (a *Account) ModifyPassword(password string) error {
	a.Password = password

	isValid := a.IsPasswordValid()
	if !isValid {
		return errors.New(ErrInvalidPassword)
	}

	if err := a.EncryptPassword(); err != nil {
		return err
	}

	return nil
}

func (a *Account) EncryptPassword() error {
	if err := a.generateSalt(); err != nil {
		return err
	}

	a.Password = utils.GenerateMd5WithSalt(a.Password, a.Salt)
	return nil
}

func (a *Account) generateSalt() error {
	salt, err := utils.GetPasswordSalt()
	if err != nil {
		return err
	}

	a.Salt = salt
	return nil
}

func (a *Account) CheckPassword(password string) error {
	passwordMd5 := utils.GenerateMd5WithSalt(password, a.Salt)
	if passwordMd5 != a.Password {
		return errors.New("wrong password")
	}
	return nil
}

func (a *Account) GenerateId() {
	a.Id = int64(uuid.New().ID())
}

type AccountRepo interface {
	CreateAccount(ctx context.Context, account *Account) error
	GetAccountById(ctx context.Context, id int64) (bool, *Account, error)
	GetAccountByMobile(ctx context.Context, mobile string) (bool, *Account, error)
	GetAccountByEmail(ctx context.Context, email string) (bool, *Account, error)
	CheckAccountUnique(ctx context.Context, account *Account) error
	UpdateAccount(ctx context.Context, account *Account) error
}

type AccountUsecase struct {
	repo AccountRepo
	log  *log.Helper
}

func NewAccountUsecase(repo AccountRepo, logger log.Logger) *AccountUsecase {
	return &AccountUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *AccountUsecase) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	account := &Account{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
	}

	err := uc.repo.CheckAccountUnique(ctx, account)
	if err != nil {
		return nil, err
	}

	valid := account.IsPasswordValid()
	if !valid {
		return nil, errors.New(ErrInvalidPassword)
	}

	err = account.EncryptPassword()
	if err != nil {
		return nil, err
	}

	account.GenerateId()
	err = uc.repo.CreateAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	return &RegisterResponse{
		AccountId: account.Id,
	}, nil
}

func (uc *AccountUsecase) CheckAccount(ctx context.Context, req *CheckAccountRequest) (*CheckAccountResponse, error) {
	var account *Account
	var err error
	var exist bool

	if req.AccountId != 0 {
		exist, account, err = uc.repo.GetAccountById(ctx, req.AccountId)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, errors.New("account not exist")
		}
	}

	if req.Mobile != "" {
		exist, account, err = uc.repo.GetAccountByMobile(ctx, req.Mobile)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, errors.New("account not exist")
		}
	}

	if req.Email != "" {
		exist, account, err = uc.repo.GetAccountByEmail(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, errors.New("account not exist")
		}
	}

	if account == nil {
		return nil, errors.New("account not exist")
	}

	err = account.CheckPassword(req.Password)
	if err != nil {
		return nil, err
	}

	return &CheckAccountResponse{
		AccountId: account.Id,
	}, nil
}

func (uc *AccountUsecase) Bind(ctx context.Context, req *BindRequest) (*BindResponse, error) {
	exist, account, err := uc.repo.GetAccountById(ctx, req.AccountId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("account not exist")
	}

	switch req.VoucherType {
	case VoucherTypeEmail:
		account.Email = req.Voucher
	case VoucherTypePhone:
		account.Mobile = req.Voucher
	default:
		return nil, errors.New("invalid voucher type")
	}

	err = uc.repo.UpdateAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	return &BindResponse{}, nil
}

func (uc *AccountUsecase) Unbind(ctx context.Context, req *UnbindRequest) (*UnbindResponse, error) {
	exist, account, err := uc.repo.GetAccountById(ctx, req.AccountId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("account not exist")
	}

	switch req.VoucherType {
	case VoucherTypeEmail:
		account.Email = ""
	case VoucherTypePhone:
		account.Mobile = ""
	default:
		return nil, errors.New("invalid voucher type")
	}

	err = uc.repo.UpdateAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	return &UnbindResponse{}, nil
}
