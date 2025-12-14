package service

import (
	"context"

	pb "lehu-video/api/videoApi/service/v1"
)

type UserServiceService struct {
	pb.UnimplementedUserServiceServer
}

func NewUserServiceService() *UserServiceService {
	return &UserServiceService{}
}

func (s *UserServiceService) GetVerificationCode(ctx context.Context, req *pb.GetVerificationCodeReq) (*pb.GetVerificationCodeResp, error) {
	return &pb.GetVerificationCodeResp{}, nil
}
func (s *UserServiceService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	return &pb.RegisterResp{}, nil
}
func (s *UserServiceService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	return &pb.LoginResp{}, nil
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
