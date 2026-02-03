package service

import (
	"context"
	"github.com/spf13/cast"
	"strconv"

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
	accountId, _ := strconv.ParseInt(req.AccountId, 10, 64)

	cmd := &biz.CreateUserCommand{
		AccountId: accountId,
		Mobile:    req.Mobile,
		Email:     req.Email,
		Name:      req.Name,
	}

	result, err := s.uc.CreateUser(ctx, cmd)
	if err != nil {
		return &pb.CreateUserResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateUserResp{
		Meta:   utils.GetSuccessMeta(),
		UserId: strconv.FormatInt(result.UserId, 10),
	}, nil
}

func (s *UserServiceService) GetUserBaseInfo(ctx context.Context, req *pb.GetUserBaseInfoReq) (*pb.GetUserBaseInfoResp, error) {
	query := &biz.GetUserBaseInfoQuery{
		UserId:    cast.ToInt64(req.UserId),
		AccountId: cast.ToInt64(req.AccountId),
	}

	result, err := s.uc.GetUserBaseInfo(ctx, query)
	if err != nil {
		return &pb.GetUserBaseInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	user := &pb.UserBaseInfo{
		Id:              strconv.FormatInt(result.User.Id, 10),
		Name:            result.User.Name,
		Nickname:        result.User.Nickname,
		Avatar:          result.User.Avatar,
		BackgroundImage: result.User.BackgroundImage,
		Signature:       result.User.Signature,
		Mobile:          result.User.Mobile,
		Email:           result.User.Email,
		Gender:          result.User.Gender,
		FollowCount:     result.User.FollowCount,
		FollowerCount:   result.User.FollowerCount,
		TotalFavorited:  result.User.TotalFavorited,
		WorkCount:       result.User.WorkCount,
		FavoriteCount:   result.User.FavoriteCount,
		CreatedAt:       result.User.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       result.User.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return &pb.GetUserBaseInfoResp{
		Meta: utils.GetSuccessMeta(),
		User: user,
	}, nil
}

func (s *UserServiceService) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	cmd := &biz.UpdateUserInfoCommand{
		UserId:          cast.ToInt64(req.UserId),
		Name:            req.Name,
		Nickname:        req.Nickname,
		Avatar:          req.Avatar,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
		Gender:          req.Gender,
	}

	_, err := s.uc.UpdateUserInfo(ctx, cmd)
	if err != nil {
		return &pb.UpdateUserInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UpdateUserInfoResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *UserServiceService) BatchGetUserBaseInfo(ctx context.Context, req *pb.BatchGetUserBaseInfoReq) (*pb.BatchGetUserBaseInfoResp, error) {
	query := &biz.BatchGetUserBaseInfoQuery{
		UserIds: cast.ToInt64Slice(req.UserIds),
	}

	result, err := s.uc.BatchGetUserBaseInfo(ctx, query)
	if err != nil {
		return &pb.BatchGetUserBaseInfoResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	users := make([]*pb.UserBaseInfo, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, &pb.UserBaseInfo{
			Id:              strconv.FormatInt(user.Id, 10),
			Name:            user.Name,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Gender:          user.Gender,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorited,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
			CreatedAt:       user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.BatchGetUserBaseInfoResp{
		Meta:  utils.GetSuccessMeta(),
		Users: users,
	}, nil
}

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

	users := make([]*pb.UserBaseInfo, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, &pb.UserBaseInfo{
			Id:              strconv.FormatInt(user.Id, 10),
			Name:            user.Name,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Gender:          user.Gender,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorited,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
			CreatedAt:       user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.SearchUsersResp{
		Meta:  utils.GetSuccessMeta(),
		Users: users,
		Total: int32(result.Total),
	}, nil
}

func (s *UserServiceService) UpdateUserStats(ctx context.Context, req *pb.UpdateUserStatsReq) (*pb.UpdateUserStatsResp, error) {
	var followCount, followerCount, totalFavorited, workCount, favoriteCount *int64

	if req.FollowCount != nil {
		followCount = req.FollowCount
	}
	if req.FollowerCount != nil {
		followerCount = req.FollowerCount
	}
	if req.TotalFavorited != nil {
		totalFavorited = req.TotalFavorited
	}
	if req.WorkCount != nil {
		workCount = req.WorkCount
	}
	if req.FavoriteCount != nil {
		favoriteCount = req.FavoriteCount
	}

	cmd := &biz.UpdateUserStatsCommand{
		UserId:         cast.ToInt64(req.UserId),
		FollowCount:    followCount,
		FollowerCount:  followerCount,
		TotalFavorited: totalFavorited,
		WorkCount:      workCount,
		FavoriteCount:  favoriteCount,
	}

	_, err := s.uc.UpdateUserStats(ctx, cmd)
	if err != nil {
		return &pb.UpdateUserStatsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UpdateUserStatsResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
