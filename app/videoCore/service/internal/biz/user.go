package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
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

// ✅ biz层自己的请求/响应结构体
type CreateUserRequest struct {
	AccountId int64
	Mobile    string
	Email     string
}

type CreateUserResponse struct {
	UserId int64
}

type UpdateUserInfoRequest struct {
	UserId          int64
	Name            string
	Avatar          string
	BackgroundImage string
	Signature       string
}

type UpdateUserInfoResponse struct {
	// 更新成功不需要额外数据
}

type GetUserInfoRequest struct {
	UserId    int64
	AccountId int64
}

type GetUserInfoResponse struct {
	User *User
}

type GetUserByIdListRequest struct {
	UserIdList []int64
}

type GetUserByIdListResponse struct {
	UserList []*User
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

func (uc *UserUsecase) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
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
	return &CreateUserResponse{
		UserId: user.Id,
	}, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, req *UpdateUserInfoRequest) (*UpdateUserInfoResponse, error) {
	exist, oldUser, err := uc.repo.GetUserById(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("用户不存在")
	}

	newUser := &User{
		Id:              req.UserId,
		AccountId:       oldUser.AccountId,
		Mobile:          oldUser.Mobile,
		Email:           oldUser.Email,
		Name:            req.Name,
		Avatar:          req.Avatar,
		BackgroundImage: req.BackgroundImage,
		Signature:       req.Signature,
		CreatedAt:       oldUser.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	err = uc.repo.UpdateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return &UpdateUserInfoResponse{}, nil
}

func (uc *UserUsecase) GetUserInfo(ctx context.Context, req *GetUserInfoRequest) (*GetUserInfoResponse, error) {
	// 参数验证
	if req.UserId == 0 && req.AccountId == 0 {
		return nil, errors.New("参数错误：必须提供UserId或AccountId")
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
		return nil, errors.New("用户不存在")
	}

	return &GetUserInfoResponse{
		User: existUser,
	}, nil
}

func (uc *UserUsecase) GetUserByIdList(ctx context.Context, req *GetUserByIdListRequest) (*GetUserByIdListResponse, error) {
	userList, err := uc.repo.GetUserByIdList(ctx, req.UserIdList)
	if err != nil {
		return nil, err
	}

	return &GetUserByIdListResponse{
		UserList: userList,
	}, nil
}
