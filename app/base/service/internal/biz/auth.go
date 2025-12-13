package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/base/service/v1"
	"lehu-video/app/base/service/internal/pkg/utils"
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

func (uc *AuthUsecase) CreateVerificationCode(ctx context.Context, req *pb.CreateVerificationCodeReq) (*pb.CreateVerificationCodeResp, error) {
	code, err := uc.repo.CreateVerificationCode(ctx, req.Bits, req.ExpireTime)
	if err != nil {
		return nil, err
	}
	return &pb.CreateVerificationCodeResp{
		VerificationCodeId: code.Id,
		Meta:               utils.GetSuccessMeta(),
	}, nil
}

func (uc *AuthUsecase) ValidateVerificationCode(ctx context.Context, req *pb.ValidateVerificationCodeReq) (*pb.ValidateVerificationCodeResp, error) {
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
	return &pb.ValidateVerificationCodeResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
