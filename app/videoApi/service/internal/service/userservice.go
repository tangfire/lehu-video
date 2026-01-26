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
	codeId, err := s.uc.GenerateVerificationCode(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetVerificationCodeResp{
		CodeId: codeId,
	}, nil
}

func (s *UserServiceService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	// ✅ 改为Input
	input := &biz.RegisterInput{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
		CodeId:   req.CodeId,
		Code:     req.Code,
	}
	output, err := s.uc.ProcessRegistration(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResp{
		UserId: output.UserId,
	}, nil
}

func (s *UserServiceService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	// ✅ 改为Input
	input := &biz.LoginInput{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
	}
	output, err := s.uc.AuthenticateUser(ctx, input)
	if err != nil {
		return nil, err
	}
	user := output.User
	retUser := &pb.User{
		Id:              user.Id,
		Name:            user.Name,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Mobile:          user.Mobile,
		Email:           user.Email,
	}
	return &pb.LoginResp{
		Token: output.Token,
		User:  retUser,
	}, nil
}

func (s *UserServiceService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	// ✅ 改为Input
	input := &biz.GetUserInfoInput{
		UserId: req.UserId,
	}
	output, err := s.uc.RetrieveUserInfo(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换biz.UserInfo到pb.User
	user := &pb.User{
		Id:              output.User.Id,
		Name:            output.User.Name,
		Avatar:          output.User.Avatar,
		BackgroundImage: output.User.BackgroundImage,
		Mobile:          output.User.Mobile,
		Email:           output.User.Email,
	}

	return &pb.GetUserInfoResp{
		User: user,
	}, nil
}

func (s *UserServiceService) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	// ✅ 改为Input
	input := &biz.UpdateUserInfoInput{
		UserId:          req.UserId,
		Name:            req.Name,
		Avatar:          req.Avatar,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
	}
	_, err := s.uc.UpdateUserProfile(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserInfoResp{}, nil
}

func (s *UserServiceService) BindUserVoucher(ctx context.Context, req *pb.BindUserVoucherReq) (*pb.BindUserVoucherResp, error) {
	// ✅ 改为Input
	input := &biz.BindUserVoucherInput{
		// TODO: 从context中获取UserId
		VoucherType: req.VoucherType.String(),
		Voucher:     req.Voucher,
		// TODO: 需要获取CodeId和Code
	}
	_, err := s.uc.BindUserVoucher(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.BindUserVoucherResp{}, nil
}

func (s *UserServiceService) UnbindUserVoucher(ctx context.Context, req *pb.UnbindUserVoucherReq) (*pb.UnbindUserVoucherResp, error) {
	// ✅ 改为Input
	input := &biz.UnbindUserVoucherInput{
		// TODO: 从context中获取UserId
		VoucherType: req.VoucherType.String(),
	}
	_, err := s.uc.UnbindUserVoucher(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.UnbindUserVoucherResp{}, nil
}
