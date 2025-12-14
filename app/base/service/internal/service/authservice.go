package service

import (
	"context"
	"lehu-video/app/base/service/internal/biz"

	pb "lehu-video/api/base/service/v1"
)

type AuthServiceService struct {
	pb.UnimplementedAuthServiceServer

	uc *biz.AuthUsecase
}

func NewAuthServiceService(uc *biz.AuthUsecase) *AuthServiceService {
	return &AuthServiceService{uc: uc}
}

func (s *AuthServiceService) CreateVerificationCode(ctx context.Context, req *pb.CreateVerificationCodeReq) (*pb.CreateVerificationCodeResp, error) {
	return s.uc.CreateVerificationCode(ctx, req)
}
func (s *AuthServiceService) ValidateVerificationCode(ctx context.Context, req *pb.ValidateVerificationCodeReq) (*pb.ValidateVerificationCodeResp, error) {
	return s.uc.ValidateVerificationCode(ctx, req)
}
