package biz

import (
	"context"
	"testing"

	"lehu-video/app/base/service/internal/pkg/idgen"
	"lehu-video/pkg/password"

	"github.com/go-kratos/kratos/v2/log"
)

type memoryAccountRepo struct {
	accounts map[int64]*Account
}

func newMemoryAccountRepo() *memoryAccountRepo {
	return &memoryAccountRepo{accounts: make(map[int64]*Account)}
}

func (r *memoryAccountRepo) CreateAccount(ctx context.Context, account *Account) error {
	cp := *account
	r.accounts[account.Id] = &cp
	return nil
}

func (r *memoryAccountRepo) GetAccountById(ctx context.Context, id int64) (bool, *Account, error) {
	account, ok := r.accounts[id]
	if !ok {
		return false, nil, nil
	}
	cp := *account
	return true, &cp, nil
}

func (r *memoryAccountRepo) GetAccountByMobile(ctx context.Context, mobile string) (bool, *Account, error) {
	for _, account := range r.accounts {
		if account.Mobile == mobile {
			cp := *account
			return true, &cp, nil
		}
	}
	return false, nil, nil
}

func (r *memoryAccountRepo) GetAccountByEmail(ctx context.Context, email string) (bool, *Account, error) {
	for _, account := range r.accounts {
		if account.Email == email {
			cp := *account
			return true, &cp, nil
		}
	}
	return false, nil, nil
}

func (r *memoryAccountRepo) CheckAccountUnique(ctx context.Context, account *Account) error {
	return nil
}

func (r *memoryAccountRepo) UpdateAccount(ctx context.Context, account *Account) error {
	cp := *account
	r.accounts[account.Id] = &cp
	return nil
}

func TestRegisterStoresModernPassword(t *testing.T) {
	repo := newMemoryAccountRepo()
	uc := NewAccountUsecase(repo, idgen.NewGenerator(1), log.DefaultLogger)

	result, err := uc.Register(context.Background(), &RegisterCommand{
		Mobile:   "13800138000",
		Email:    "test@example.com",
		Password: "12345678",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	account := repo.accounts[result.AccountId]
	if account == nil || !password.IsModern(account.Password) {
		t.Fatalf("password should be modern, account=%#v", account)
	}
}

func TestCheckAccountUpgradesLegacyPassword(t *testing.T) {
	repo := newMemoryAccountRepo()
	repo.accounts[1] = &Account{
		Id:       1,
		Mobile:   "13800138000",
		Password: password.MD5WithSalt("12345678", "salt"),
		Salt:     "salt",
	}
	uc := NewAccountUsecase(repo, idgen.NewGenerator(1), log.DefaultLogger)

	if _, err := uc.CheckAccount(context.Background(), &CheckAccountQuery{
		Mobile:   "13800138000",
		Password: "12345678",
	}); err != nil {
		t.Fatalf("CheckAccount() error = %v", err)
	}
	if !password.IsModern(repo.accounts[1].Password) {
		t.Fatalf("password was not upgraded: %#v", repo.accounts[1])
	}
}
