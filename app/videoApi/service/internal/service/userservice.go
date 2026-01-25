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
	codeId, err := s.uc.GetVerificationCode(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetVerificationCodeResp{
		CodeId: codeId,
	}, nil
}
func (s *UserServiceService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	bizReq := &biz.RegisterReq{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
		CodeId:   req.CodeId,
		Code:     req.Code,
	}
	resp, err := s.uc.Register(ctx, bizReq)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResp{
		UserId: resp.UserId,
	}, nil
}
func (s *UserServiceService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	bizReq := &biz.LoginReq{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
	}
	resp, err := s.uc.Login(ctx, bizReq)
	if err != nil {
		return nil, err
	}
	user := resp.User
	retUser := &pb.User{
		Id:              user.Id,
		Name:            user.Name,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Mobile:          user.Mobile,
		Email:           user.Email,
	}
	return &pb.LoginResp{
		Token: resp.Token,
		User:  retUser,
	}, nil
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
