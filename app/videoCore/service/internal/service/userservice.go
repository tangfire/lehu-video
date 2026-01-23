package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	pb "lehu-video/api/videoCore/service/v1"
)

type UserServiceService struct {
	pb.UnimplementedUserServiceServer

	uc *biz.UserUsecase
}

func NewUserServiceService(uc *biz.UserUsecase) *UserServiceService {
	return &UserServiceService{uc: uc}
}

func (s *UserServiceService) CreateUser(ctx context.Context, req *pb.CreateUserReq) (*pb.CreateUserResp, error) {
	return s.uc.CreateUser(ctx, req)
}
func (s *UserServiceService) UpdateUser(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	return s.uc.UpdateUser(ctx, req)
}
func (s *UserServiceService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	return s.uc.GetUserInfo(ctx, req)
}
func (s *UserServiceService) GetUserByIdList(ctx context.Context, req *pb.GetUserByIdListReq) (*pb.GetUserByIdListResp, error) {
	return s.uc.GetUserByIdList(ctx, req)
}
