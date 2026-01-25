package service

import (
	"context"
	pb "lehu-video/api/base/service/v1"
	"lehu-video/app/base/service/internal/biz"
	"lehu-video/app/base/service/internal/pkg/utils"
)

type AuthServiceService struct {
	pb.UnimplementedAuthServiceServer
	uc *biz.AuthUsecase
}

func NewAuthServiceService(uc *biz.AuthUsecase) *AuthServiceService {
	return &AuthServiceService{uc: uc}
}

func (s *AuthServiceService) CreateVerificationCode(ctx context.Context, req *pb.CreateVerificationCodeReq) (*pb.CreateVerificationCodeResp, error) {
	bizReq := &biz.CreateVerificationCodeRequest{
		Bits:       req.Bits,
		ExpireTime: req.ExpireTime,
	}

	resp, err := s.uc.CreateVerificationCode(ctx, bizReq)
	if err != nil {
		return &pb.CreateVerificationCodeResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateVerificationCodeResp{
		VerificationCodeId: resp.VerificationCodeId,
		Meta:               utils.GetSuccessMeta(),
	}, nil
}

func (s *AuthServiceService) ValidateVerificationCode(ctx context.Context, req *pb.ValidateVerificationCodeReq) (*pb.ValidateVerificationCodeResp, error) {
	bizReq := &biz.ValidateVerificationCodeRequest{
		VerificationCodeId: req.VerificationCodeId,
		Code:               req.Code,
	}

	_, err := s.uc.ValidateVerificationCode(ctx, bizReq)
	if err != nil {
		return &pb.ValidateVerificationCodeResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.ValidateVerificationCodeResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
