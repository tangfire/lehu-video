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

// ✅ 使用Command/Result模式
type CreateUserCommand struct {
	AccountId int64
	Mobile    string
	Email     string
}

type CreateUserResult struct {
	UserId int64
}

type UpdateUserInfoCommand struct {
	UserId          int64
	Name            string
	Avatar          string
	BackgroundImage string
	Signature       string
}

type UpdateUserInfoResult struct {
	// 更新成功不需要额外数据
}

// ✅ 查询操作使用Query/Result
type GetUserInfoQuery struct {
	UserId    int64
	AccountId int64
}

type GetUserInfoResult struct {
	User *User
}

type GetUserByIdListQuery struct {
	UserIdList []int64
}

type GetUserByIdListResult struct {
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

func (uc *UserUsecase) CreateUser(ctx context.Context, cmd *CreateUserCommand) (*CreateUserResult, error) {
	user := &User{
		Id:              0,
		AccountId:       cmd.AccountId,
		Mobile:          cmd.Mobile,
		Email:           cmd.Email,
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
	return &CreateUserResult{
		UserId: user.Id,
	}, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, cmd *UpdateUserInfoCommand) (*UpdateUserInfoResult, error) {
	exist, oldUser, err := uc.repo.GetUserById(ctx, cmd.UserId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("用户不存在")
	}

	newUser := &User{
		Id:              cmd.UserId,
		AccountId:       oldUser.AccountId,
		Mobile:          oldUser.Mobile,
		Email:           oldUser.Email,
		Name:            cmd.Name,
		Avatar:          cmd.Avatar,
		BackgroundImage: cmd.BackgroundImage,
		Signature:       cmd.Signature,
		CreatedAt:       oldUser.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	err = uc.repo.UpdateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return &UpdateUserInfoResult{}, nil
}

func (uc *UserUsecase) GetUserInfo(ctx context.Context, query *GetUserInfoQuery) (*GetUserInfoResult, error) {
	// 参数验证
	if query.UserId == 0 && query.AccountId == 0 {
		return nil, errors.New("参数错误：必须提供UserId或AccountId")
	}

	var existUser *User
	var exist bool
	var err error

	// 优先通过UserId查找
	if query.UserId != 0 {
		exist, existUser, err = uc.repo.GetUserById(ctx, query.UserId)
	} else {
		// 通过AccountId查找
		exist, existUser, err = uc.repo.GetUserByAccountId(ctx, query.AccountId)
	}

	// 处理错误
	if err != nil {
		// 记录错误日志
		uc.log.Error("获取用户信息失败", "error", err, "userId", query.UserId, "accountId", query.AccountId)
		return nil, err
	}

	// 用户不存在
	if !exist {
		return nil, errors.New("用户不存在")
	}

	return &GetUserInfoResult{
		User: existUser,
	}, nil
}

func (uc *UserUsecase) GetUserByIdList(ctx context.Context, query *GetUserByIdListQuery) (*GetUserByIdListResult, error) {
	userList, err := uc.repo.GetUserByIdList(ctx, query.UserIdList)
	if err != nil {
		return nil, err
	}

	return &GetUserByIdListResult{
		UserList: userList,
	}, nil
}
