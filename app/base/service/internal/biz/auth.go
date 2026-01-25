package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
)

type VerificationCode struct {
	Id   int64
	Code string
}

func NewVerificationCode(id int64, code string) *VerificationCode {
	return &VerificationCode{
		Id:   id,
		Code: code,
	}
}

func (v *VerificationCode) IsReady() error {
	if v.Id == 0 {
		return errors.New("verification code id is required")
	}

	if v.Code == "" {
		return errors.New("code is required")
	}

	return nil
}

func (v *VerificationCode) Check(another *VerificationCode) (bool, error) {
	if v.Id != another.Id {
		return false, errors.New("verification code id is not match")
	}

	if v.Code != another.Code {
		return false, errors.New("code is not match")
	}

	return true, nil
}

// ✅ biz层自己的请求/响应结构体
type CreateVerificationCodeRequest struct {
	Bits       int64
	ExpireTime int64
}

type CreateVerificationCodeResponse struct {
	VerificationCodeId int64
}

type ValidateVerificationCodeRequest struct {
	VerificationCodeId int64
	Code               string
}

type ValidateVerificationCodeResponse struct {
	// 验证成功不需要额外数据，空结构体即可
}

type AuthRepo interface {
	CreateVerificationCode(ctx context.Context, bits, expireTime int64) (*VerificationCode, error)
	GetVerificationCode(ctx context.Context, id int64) (*VerificationCode, error)
	DelVerificationCode(ctx context.Context, id int64) error
}

type AuthUsecase struct {
	repo AuthRepo
	log  *log.Helper
}

func NewAuthUsecase(repo AuthRepo, logger log.Logger) *AuthUsecase {
	return &AuthUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *AuthUsecase) CreateVerificationCode(ctx context.Context, req *CreateVerificationCodeRequest) (*CreateVerificationCodeResponse, error) {
	code, err := uc.repo.CreateVerificationCode(ctx, req.Bits, req.ExpireTime)
	if err != nil {
		return nil, err
	}

	return &CreateVerificationCodeResponse{
		VerificationCodeId: code.Id,
	}, nil
}

func (uc *AuthUsecase) ValidateVerificationCode(ctx context.Context, req *ValidateVerificationCodeRequest) (*ValidateVerificationCodeResponse, error) {
	code := NewVerificationCode(req.VerificationCodeId, req.Code)
	err := code.IsReady()
	if err != nil {
		return nil, err
	}

	srcCode, err := uc.repo.GetVerificationCode(ctx, code.Id)
	if err != nil {
		return nil, err
	}

	check, err := code.Check(srcCode)
	if err != nil {
		return nil, err
	}

	if !check {
		return nil, errors.New("verification code check failed")
	}

	err = uc.repo.DelVerificationCode(ctx, code.Id)
	if err != nil {
		return nil, err
	}

	return &ValidateVerificationCodeResponse{}, nil
}
