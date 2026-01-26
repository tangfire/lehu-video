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

// ✅ 使用Command/Result模式，更符合业务语义
type RegisterCommand struct {
	Mobile   string
	Email    string
	Password string
}

type RegisterResult struct {
	AccountId int64
}

// ✅ 这是查询操作，使用Query/Result
type CheckAccountQuery struct {
	AccountId int64
	Mobile    string
	Email     string
	Password  string
}

type CheckAccountResult struct {
	AccountId int64
}

// ✅ 绑定是命令操作
type BindCommand struct {
	AccountId   int64
	VoucherType VoucherType
	Voucher     string
}

type BindResult struct{}

// ✅ 解绑也是命令操作
type UnbindCommand struct {
	AccountId   int64
	VoucherType VoucherType
}

type UnbindResult struct{}

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

// ✅ 方法签名改为使用Command/Result
func (uc *AccountUsecase) Register(ctx context.Context, cmd *RegisterCommand) (*RegisterResult, error) {
	account := &Account{
		Mobile:   cmd.Mobile,
		Email:    cmd.Email,
		Password: cmd.Password,
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

	return &RegisterResult{
		AccountId: account.Id,
	}, nil
}

// ✅ 查询方法使用Query/Result
func (uc *AccountUsecase) CheckAccount(ctx context.Context, query *CheckAccountQuery) (*CheckAccountResult, error) {
	var account *Account
	var err error
	var exist bool

	if query.AccountId != 0 {
		exist, account, err = uc.repo.GetAccountById(ctx, query.AccountId)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, errors.New("account not exist")
		}
	}

	if query.Mobile != "" {
		exist, account, err = uc.repo.GetAccountByMobile(ctx, query.Mobile)
		if err != nil {
			return nil, err
		}
		if !exist {
			return nil, errors.New("account not exist")
		}
	}

	if query.Email != "" {
		exist, account, err = uc.repo.GetAccountByEmail(ctx, query.Email)
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

	err = account.CheckPassword(query.Password)
	if err != nil {
		return nil, err
	}

	return &CheckAccountResult{
		AccountId: account.Id,
	}, nil
}

// ✅ 命令方法使用Command/Result
func (uc *AccountUsecase) Bind(ctx context.Context, cmd *BindCommand) (*BindResult, error) {
	exist, account, err := uc.repo.GetAccountById(ctx, cmd.AccountId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("account not exist")
	}

	switch cmd.VoucherType {
	case VoucherTypeEmail:
		account.Email = cmd.Voucher
	case VoucherTypePhone:
		account.Mobile = cmd.Voucher
	default:
		return nil, errors.New("invalid voucher type")
	}

	err = uc.repo.UpdateAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	return &BindResult{}, nil
}

// ✅ 命令方法使用Command/Result
func (uc *AccountUsecase) Unbind(ctx context.Context, cmd *UnbindCommand) (*UnbindResult, error) {
	exist, account, err := uc.repo.GetAccountById(ctx, cmd.AccountId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("account not exist")
	}

	switch cmd.VoucherType {
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

	return &UnbindResult{}, nil
}
