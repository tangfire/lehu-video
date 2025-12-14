package service

import (
	"context"
	"lehu-video/app/base/service/internal/biz"

	pb "lehu-video/api/base/service/v1"
)

type AccountServiceService struct {
	pb.UnimplementedAccountServiceServer

	uc *biz.AccountUsecase
}

func NewAccountServiceService(uc *biz.AccountUsecase) *AccountServiceService {
	return &AccountServiceService{uc: uc}
}

func (s *AccountServiceService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	return s.uc.Register(ctx, req)
}
func (s *AccountServiceService) CheckAccount(ctx context.Context, req *pb.CheckAccountReq) (*pb.CheckAccountResp, error) {
	return s.uc.CheckAccount(ctx, req)
}
func (s *AccountServiceService) Bind(ctx context.Context, req *pb.BindReq) (*pb.BindResp, error) {
	return &pb.BindResp{}, nil
}
func (s *AccountServiceService) Unbind(ctx context.Context, req *pb.UnbindReq) (*pb.UnbindResp, error) {
	return &pb.UnbindResp{}, nil
}
