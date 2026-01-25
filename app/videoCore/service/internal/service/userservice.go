package service

import (
	"context"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
)

type UserServiceService struct {
	pb.UnimplementedUserServiceServer
	uc *biz.UserUsecase
}

func NewUserServiceService(uc *biz.UserUsecase) *UserServiceService {
	return &UserServiceService{uc: uc}
}

func (s *UserServiceService) CreateUser(ctx context.Context, req *pb.CreateUserReq) (*pb.CreateUserResp, error) {
	bizReq := &biz.CreateUserRequest{
		AccountId: req.AccountId,
		Mobile:    req.Mobile,
		Email:     req.Email,
	}

	resp, err := s.uc.CreateUser(ctx, bizReq)
	if err != nil {
		return &pb.CreateUserResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateUserResp{
		Meta:   utils.GetSuccessMeta(),
		UserId: resp.UserId,
	}, nil
}

func (s *UserServiceService) UpdateUser(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	bizReq := &biz.UpdateUserInfoRequest{
		UserId:          req.UserId,
		Name:            req.Name,
		Avatar:          req.Avatar,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
	}

	_, err := s.uc.UpdateUser(ctx, bizReq)
	if err != nil {
		return &pb.UpdateUserInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UpdateUserInfoResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *UserServiceService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	bizReq := &biz.GetUserInfoRequest{
		UserId:    req.UserId,
		AccountId: req.AccountId,
	}

	resp, err := s.uc.GetUserInfo(ctx, bizReq)
	if err != nil {
		return &pb.GetUserInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换biz.User到pb.User
	user := &pb.User{
		Id:              resp.User.Id,
		Name:            resp.User.Name,
		Avatar:          resp.User.Avatar,
		BackgroundImage: resp.User.BackgroundImage,
		Signature:       resp.User.Signature,
		Mobile:          resp.User.Mobile,
		Email:           resp.User.Email,
	}

	return &pb.GetUserInfoResp{
		Meta: utils.GetSuccessMeta(),
		User: user,
	}, nil
}

func (s *UserServiceService) GetUserByIdList(ctx context.Context, req *pb.GetUserByIdListReq) (*pb.GetUserByIdListResp, error) {
	bizReq := &biz.GetUserByIdListRequest{
		UserIdList: req.UserIdList,
	}

	resp, err := s.uc.GetUserByIdList(ctx, bizReq)
	if err != nil {
		return &pb.GetUserByIdListResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换[]*biz.User到[]*pb.User
	var retList []*pb.User
	for _, user := range resp.UserList {
		retList = append(retList, &pb.User{
			Id:              user.Id,
			Name:            user.Name,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
		})
	}

	return &pb.GetUserByIdListResp{
		Meta:     utils.GetSuccessMeta(),
		UserList: retList,
	}, nil
}
