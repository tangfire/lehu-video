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
	cmd := &biz.CreateUserCommand{
		AccountId: req.AccountId,
		Mobile:    req.Mobile,
		Email:     req.Email,
	}

	result, err := s.uc.CreateUser(ctx, cmd)
	if err != nil {
		return &pb.CreateUserResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateUserResp{
		Meta:   utils.GetSuccessMeta(),
		UserId: result.UserId,
	}, nil
}

func (s *UserServiceService) UpdateUser(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	cmd := &biz.UpdateUserInfoCommand{
		UserId:          req.UserId,
		Name:            req.Name,
		Avatar:          req.Avatar,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
	}

	_, err := s.uc.UpdateUser(ctx, cmd)
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
	query := &biz.GetUserInfoQuery{
		UserId:    req.UserId,
		AccountId: req.AccountId,
	}

	result, err := s.uc.GetUserInfo(ctx, query)
	if err != nil {
		return &pb.GetUserInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	user := &pb.User{
		Id:              result.User.Id,
		Name:            result.User.Name,
		Avatar:          result.User.Avatar,
		BackgroundImage: result.User.BackgroundImage,
		Signature:       result.User.Signature,
		Mobile:          result.User.Mobile,
		Email:           result.User.Email,
	}

	return &pb.GetUserInfoResp{
		Meta: utils.GetSuccessMeta(),
		User: user,
	}, nil
}

func (s *UserServiceService) GetUserByIdList(ctx context.Context, req *pb.GetUserByIdListReq) (*pb.GetUserByIdListResp, error) {
	query := &biz.GetUserByIdListQuery{
		UserIdList: req.UserIdList,
	}

	result, err := s.uc.GetUserByIdList(ctx, query)
	if err != nil {
		return &pb.GetUserByIdListResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var retList []*pb.User
	for _, user := range result.UserList {
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

// 新增：搜索用户实现
func (s *UserServiceService) SearchUsers(ctx context.Context, req *pb.SearchUsersReq) (*pb.SearchUsersResp, error) {
	query := &biz.SearchUsersQuery{
		Keyword:  req.Keyword,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	result, err := s.uc.SearchUsers(ctx, query)
	if err != nil {
		return &pb.SearchUsersResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var users []*pb.User
	for _, u := range result.Users {
		users = append(users, &pb.User{
			Id:              u.Id,
			Name:            u.Name,
			Avatar:          u.Avatar,
			BackgroundImage: u.BackgroundImage,
			Signature:       u.Signature,
			Mobile:          u.Mobile,
			Email:           u.Email,
		})
	}

	return &pb.SearchUsersResp{
		Users: users,
		Total: int32(result.Total),
		Meta:  utils.GetSuccessMeta(),
	}, nil
}
