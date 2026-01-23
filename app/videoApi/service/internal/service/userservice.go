package service

import (
	"context"
	"lehu-video/app/videoApi/service/internal/biz"

	pb "lehu-video/api/videoApi/service/v1"
)

type UserServiceService struct {
	pb.UnimplementedUserServiceServer

	uc *biz.UserUsecase
}

func NewUserServiceService(uc *biz.UserUsecase) *UserServiceService {
	return &UserServiceService{uc: uc}
}

func (s *UserServiceService) GetVerificationCode(ctx context.Context, req *pb.GetVerificationCodeReq) (*pb.GetVerificationCodeResp, error) {
	return s.uc.GetVerificationCode(ctx, req)
}
func (s *UserServiceService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	return s.uc.Register(ctx, req)
}
func (s *UserServiceService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	return s.uc.Login(ctx, req)
}
func (s *UserServiceService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	return &pb.GetUserInfoResp{}, nil
}
func (s *UserServiceService) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	return &pb.UpdateUserInfoResp{}, nil
}
func (s *UserServiceService) BindUserVoucher(ctx context.Context, req *pb.BindUserVoucherReq) (*pb.BindUserVoucherResp, error) {
	return &pb.BindUserVoucherResp{}, nil
}
func (s *UserServiceService) UnbindUserVoucher(ctx context.Context, req *pb.UnbindUserVoucherReq) (*pb.UnbindUserVoucherResp, error) {
	return &pb.UnbindUserVoucherResp{}, nil
}
