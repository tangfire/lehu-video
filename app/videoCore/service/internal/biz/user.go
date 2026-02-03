package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type User struct {
	Id              int64
	AccountId       int64
	Mobile          string
	Email           string
	Name            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	Gender          int32
	OnlineStatus    int32
	LastOnlineTime  time.Time

	// 统计信息
	FollowCount    int64
	FollowerCount  int64
	TotalFavorited int64
	WorkCount      int64
	FavoriteCount  int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) GenerateId() {
	// 使用雪花ID或UUID
	// 这里简化为使用UUID
	u.Id = int64(uuid.New().ID())
}

// Command 和 Query 结构体
type CreateUserCommand struct {
	AccountId int64
	Mobile    string
	Email     string
	Name      string
}

type CreateUserResult struct {
	UserId int64
}

type GetUserBaseInfoQuery struct {
	UserId    int64
	AccountId int64
}

type GetUserBaseInfoResult struct {
	User *User
}

type UpdateUserInfoCommand struct {
	UserId          int64
	Name            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	Gender          int32
}

type UpdateUserInfoResult struct{}

type BatchGetUserBaseInfoQuery struct {
	UserIds []int64
}

type BatchGetUserBaseInfoResult struct {
	Users []*User
}

type SearchUsersQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

type SearchUsersResult struct {
	Users []*User
	Total int64
}

type UpdateUserStatsCommand struct {
	UserId         int64
	FollowCount    *int64
	FollowerCount  *int64
	TotalFavorited *int64
	WorkCount      *int64
	FavoriteCount  *int64
}

type UpdateUserStatsResult struct{}

// 仓库接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	GetUserById(ctx context.Context, id int64) (bool, *User, error)
	// todo
	GetUserByAccountId(ctx context.Context, accountId int64) (bool, *User, error)
	GetUserByIdList(ctx context.Context, idList []int64) ([]*User, error)
	SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*User, int64, error)
	UpdateUserStats(ctx context.Context, userId int64, updates map[string]interface{}) error
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
		AccountId:       cmd.AccountId,
		Mobile:          cmd.Mobile,
		Email:           cmd.Email,
		Name:            cmd.Name,
		Nickname:        "",
		Avatar:          "",
		BackgroundImage: "",
		Signature:       "",
		Gender:          0,
		FollowCount:     0,
		FollowerCount:   0,
		TotalFavorited:  0,
		WorkCount:       0,
		FavoriteCount:   0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	user.GenerateId()

	err := uc.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &CreateUserResult{UserId: user.Id}, nil
}

func (uc *UserUsecase) GetUserBaseInfo(ctx context.Context, query *GetUserBaseInfoQuery) (*GetUserBaseInfoResult, error) {

	var (
		existUser *User
		exist     bool
		err       error
	)

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

	return &GetUserBaseInfoResult{User: existUser}, nil
}

func (uc *UserUsecase) UpdateUserInfo(ctx context.Context, cmd *UpdateUserInfoCommand) (*UpdateUserInfoResult, error) {
	var err error
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	exist, oldUser, err := uc.repo.GetUserById(ctx, cmd.UserId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("用户不存在")
	}

	// 更新字段
	if cmd.Name != "" {
		oldUser.Name = cmd.Name
	}
	if cmd.Nickname != "" {
		oldUser.Nickname = cmd.Nickname
	}
	if cmd.Avatar != "" {
		oldUser.Avatar = cmd.Avatar
	}
	if cmd.BackgroundImage != "" {
		oldUser.BackgroundImage = cmd.BackgroundImage
	}
	if cmd.Signature != "" {
		oldUser.Signature = cmd.Signature
	}
	if cmd.Gender != 0 {
		oldUser.Gender = cmd.Gender
	}
	oldUser.UpdatedAt = time.Now()

	err = uc.repo.UpdateUser(ctx, oldUser)
	if err != nil {
		return nil, err
	}

	return &UpdateUserInfoResult{}, nil
}

func (uc *UserUsecase) BatchGetUserBaseInfo(ctx context.Context, query *BatchGetUserBaseInfoQuery) (*BatchGetUserBaseInfoResult, error) {

	if len(query.UserIds) == 0 {
		return &BatchGetUserBaseInfoResult{Users: []*User{}}, nil
	}

	users, err := uc.repo.GetUserByIdList(ctx, query.UserIds)
	if err != nil {
		return nil, err
	}

	return &BatchGetUserBaseInfoResult{Users: users}, nil
}

func (uc *UserUsecase) SearchUsers(ctx context.Context, query *SearchUsersQuery) (*SearchUsersResult, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	offset := (query.Page - 1) * query.PageSize

	if strings.TrimSpace(query.Keyword) == "" {
		return &SearchUsersResult{
			Users: []*User{},
			Total: 0,
		}, nil
	}

	users, total, err := uc.repo.SearchUsers(ctx, query.Keyword, offset, query.PageSize)
	if err != nil {
		return nil, err
	}

	return &SearchUsersResult{
		Users: users,
		Total: total,
	}, nil
}

func (uc *UserUsecase) UpdateUserStats(ctx context.Context, cmd *UpdateUserStatsCommand) (*UpdateUserStatsResult, error) {

	updates := make(map[string]interface{})
	if cmd.FollowCount != nil {
		updates["follow_count"] = *cmd.FollowCount
	}
	if cmd.FollowerCount != nil {
		updates["follower_count"] = *cmd.FollowerCount
	}
	if cmd.TotalFavorited != nil {
		updates["total_favorited"] = *cmd.TotalFavorited
	}
	if cmd.WorkCount != nil {
		updates["work_count"] = *cmd.WorkCount
	}
	if cmd.FavoriteCount != nil {
		updates["favorite_count"] = *cmd.FavoriteCount
	}

	if len(updates) == 0 {
		return &UpdateUserStatsResult{}, nil
	}

	err := uc.repo.UpdateUserStats(ctx, cmd.UserId, updates)
	if err != nil {
		return nil, err
	}

	return &UpdateUserStatsResult{}, nil
}
