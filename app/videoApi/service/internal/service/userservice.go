package service

import (
	"context"
	"lehu-video/app/videoApi/service/internal/biz"
	"strconv"

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
	// ✅ 改为Input
	input := &biz.RegisterInput{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
		CodeId:   req.CodeId,
		Code:     req.Code,
	}
	output, err := s.uc.Register(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResp{
		UserId: output.UserID,
	}, nil
}

func (s *UserServiceService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	// ✅ 改为Input
	input := &biz.LoginInput{
		Mobile:   req.Mobile,
		Email:    req.Email,
		Password: req.Password,
	}
	output, err := s.uc.Login(ctx, input)
	if err != nil {
		return nil, err
	}
	user := output.User
	retUser := &pb.User{
		Id:              user.ID,
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

	input := &biz.GetUserInfoInput{
		UserID: req.UserId,
	}

	output, err := s.uc.GetCompleteUserInfo(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换为proto
	user := convertToProtoUser(output.User)

	return &pb.GetUserInfoResp{
		User: user,
	}, nil
}

func (s *UserServiceService) BatchGetUserInfo(ctx context.Context, req *pb.BatchGetUserInfoReq) (*pb.BatchGetUserInfoResp, error) {

	input := &biz.BatchGetUserInfoInput{
		UserIDs:         req.UserIds,
		IncludePrivate:  req.IncludePrivate,
		IncludeRelation: req.IncludeRelation,
	}

	output, err := s.uc.BatchGetUserInfo(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换为proto
	users := make([]*pb.User, 0, len(output.Users))
	for _, user := range output.Users {
		users = append(users, convertToProtoUser(user))
	}

	return &pb.BatchGetUserInfoResp{
		Users: users,
	}, nil
}

func (s *UserServiceService) SearchUsers(ctx context.Context, req *pb.SearchUsersReq) (*pb.SearchUsersResp, error) {

	input := &biz.SearchUsersInput{
		Keyword:  req.Keyword,
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
	}

	output, err := s.uc.SearchUsers(ctx, input)
	if err != nil {
		return nil, err
	}

	// 转换为proto
	users := make([]*pb.User, 0, len(output.Users))
	for _, user := range output.Users {
		users = append(users, convertToProtoUser(user))
	}

	return &pb.SearchUsersResp{
		Users: users,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

// 辅助函数：转换biz.UserInfo到pb.User
func convertToProtoUser(user *biz.UserInfo) *pb.User {
	if user == nil {
		return nil
	}

	return &pb.User{
		Id:              user.ID,
		Name:            user.Name,
		Nickname:        user.Nickname,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		Gender:          user.Gender,
		FollowCount:     user.FollowCount,
		FollowerCount:   user.FollowerCount,
		TotalFavorited:  user.TotalFavorited,
		WorkCount:       user.WorkCount,
		FavoriteCount:   user.FavoriteCount,
		CreatedAt:       user.CreatedAt,
		OnlineStatus:    user.OnlineStatus,
		LastOnlineTime:  strconv.FormatInt(user.LastOnlineTime.Unix(), 10),
		IsFollowing:     user.IsFollowing,
		IsFollower:      user.IsFollower,
		IsFriend:        user.IsFriend,
		FriendRemark:    user.FriendRemark,
		FriendGroup:     user.FriendGroup,
		Mobile:          user.Mobile,
		Email:           user.Email,
	}
}

func (s *UserServiceService) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	// ✅ 改为Input
	input := &biz.UpdateUserInfoInput{
		UserID:          req.UserId,
		Name:            req.Name,
		Nickname:        req.Nickname,
		Avatar:          req.Avatar,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
		Gender:          req.Gender,
	}
	_, err := s.uc.UpdateUserInfo(ctx, input)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserInfoResp{}, nil
}

// BindUserVoucher 修复方法
func (s *UserServiceService) BindUserVoucher(ctx context.Context, req *pb.BindUserVoucherReq) (*pb.BindUserVoucherResp, error) {
	// TODO: 实现绑定凭证逻辑
	return &pb.BindUserVoucherResp{}, nil
}

// UnbindUserVoucher 修复方法
func (s *UserServiceService) UnbindUserVoucher(ctx context.Context, req *pb.UnbindUserVoucherReq) (*pb.UnbindUserVoucherResp, error) {
	// TODO: 实现解绑凭证逻辑
	return &pb.UnbindUserVoucherResp{}, nil
}
