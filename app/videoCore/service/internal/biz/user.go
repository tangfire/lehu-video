package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
	"time"
)

type User struct {
	Id              int64
	AccountId       int64
	Mobile          string
	Email           string
	Name            string
	Avatar          string
	BackgroundImage string
	Signature       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (u *User) GenerateId() {
	u.Id = int64(uuid.New().ID())
}

type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	GetUserById(ctx context.Context, id int64) (bool, *User, error)
	GetUserByAccountId(ctx context.Context, accountId int64) (bool, *User, error)
	GetUserByIdList(ctx context.Context, idList []int64) ([]*User, error)
}

type UserUsecase struct {
	repo UserRepo
	log  *log.Helper
}

func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *UserUsecase) CreateUser(ctx context.Context, req *pb.CreateUserReq) (*pb.CreateUserResp, error) {
	user := &User{
		Id:              0,
		AccountId:       req.AccountId,
		Mobile:          req.Mobile,
		Email:           req.Email,
		Name:            "",
		Avatar:          "",
		BackgroundImage: "",
		Signature:       "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	user.GenerateId()
	err := uc.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return &pb.CreateUserResp{
		Meta:   utils.GetSuccessMeta(),
		UserId: user.Id,
	}, nil
}
func (uc *UserUsecase) UpdateUser(ctx context.Context, req *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResp, error) {
	exist, oldUser, err := uc.repo.GetUserById(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &pb.UpdateUserInfoResp{
			Meta: utils.GetMetaWithError(errors.New("用户不存在")),
		}, nil
	}
	newUser := &User{
		Id:              req.UserId,
		AccountId:       oldUser.AccountId,
		Mobile:          oldUser.Mobile,
		Email:           oldUser.Email,
		Name:            req.Name,
		Avatar:          req.BackgroundImage,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
		CreatedAt:       oldUser.CreatedAt,
		UpdatedAt:       time.Now(),
	}
	err = uc.repo.UpdateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserInfoResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (uc *UserUsecase) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	// 参数验证
	if req.UserId == 0 && req.AccountId == 0 {
		return &pb.GetUserInfoResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误：必须提供UserId或AccountId")),
		}, nil
	}

	var existUser *User
	var exist bool
	var err error

	// 优先通过UserId查找
	if req.UserId != 0 {
		exist, existUser, err = uc.repo.GetUserById(ctx, req.UserId)
	} else {
		// 通过AccountId查找
		exist, existUser, err = uc.repo.GetUserByAccountId(ctx, req.AccountId)
	}

	// 处理错误
	if err != nil {
		// 记录错误日志
		uc.log.Error("获取用户信息失败", "error", err, "userId", req.UserId, "accountId", req.AccountId)
		return nil, err
	}

	// 用户不存在
	if !exist {
		return &pb.GetUserInfoResp{
			Meta: utils.GetMetaWithError(errors.New("用户不存在")),
		}, nil
	}

	// 构建返回的用户信息
	user := &pb.User{
		Id:              existUser.Id,
		Name:            existUser.Name,
		Avatar:          existUser.Avatar,
		BackgroundImage: existUser.BackgroundImage,
		Signature:       existUser.Signature,
		Mobile:          existUser.Mobile,
		Email:           existUser.Email,
	}

	// 返回成功
	return &pb.GetUserInfoResp{
		Meta: utils.GetSuccessMeta(),
		User: user,
	}, nil
}

func (uc *UserUsecase) GetUserByIdList(ctx context.Context, req *pb.GetUserByIdListReq) (*pb.GetUserByIdListResp, error) {
	userList, err := uc.repo.GetUserByIdList(ctx, req.UserIdList)
	if err != nil {
		return nil, err
	}
	var retList []*pb.User
	for _, user := range userList {
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
