package biz

import (
	"context"
	"errors"
	"lehu-video/app/campusUser/service/internal/pkg/idgen"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
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
	FollowCount     int64
	FollowerCount   int64
	BeLikedCount    int64 // 原 TotalFavorited
	WorkCount       int64
	CollectionCount int64 // 原 FavoriteCount

	CreatedAt time.Time
	UpdatedAt time.Time
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
	UserId          int64
	FollowCount     *int64
	FollowerCount   *int64
	BeLikedCount    *int64 // 原 TotalFavorited
	WorkCount       *int64
	CollectionCount *int64 // 原 FavoriteCount
}

type UpdateUserStatsResult struct{}

// 新增更新最后上线时间的 Command 和 Result
type UpdateUserLastOnlineTimeCommand struct {
	UserId         int64
	LastOnlineTime string // 格式：2006-01-02 15:04:05
}

type UpdateUserLastOnlineTimeResult struct{}

// 仓库接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	GetUserById(ctx context.Context, id int64) (bool, *User, error)
	GetUserByAccountId(ctx context.Context, accountId int64) (bool, *User, error)
	GetUserByIdList(ctx context.Context, idList []int64) ([]*User, error)
	SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*User, int64, error)
	UpdateUserStats(ctx context.Context, userId int64, updates map[string]interface{}) error
	UpdateUserLastOnlineTime(ctx context.Context, userId int64, lastOnlineTime time.Time) error
}

type UserUsecase struct {
	repo  UserRepo
	idGen idgen.Generator
	log   *log.Helper
}

func NewUserUsecase(repo UserRepo, idGen idgen.Generator, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, idGen: idGen, log: log.NewHelper(logger)}
}

func (uc *UserUsecase) CreateUser(ctx context.Context, cmd *CreateUserCommand) (*CreateUserResult, error) {
	user := &User{
		Id:              uc.idGen.NextID(),
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
		BeLikedCount:    0,
		WorkCount:       0,
		CollectionCount: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

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

	if query.UserId != 0 {
		exist, existUser, err = uc.repo.GetUserById(ctx, query.UserId)
	} else {
		exist, existUser, err = uc.repo.GetUserByAccountId(ctx, query.AccountId)
	}
	if err != nil {
		uc.log.Error("获取用户信息失败", "error", err, "userId", query.UserId, "accountId", query.AccountId)
		return nil, err
	}
	if !exist {
		return nil, errors.New("用户不存在")
	}

	return &GetUserBaseInfoResult{User: existUser}, nil
}

func (uc *UserUsecase) UpdateUserInfo(ctx context.Context, cmd *UpdateUserInfoCommand) (*UpdateUserInfoResult, error) {
	var err error
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

// UpdateUserStats 更新用户统计信息，MySQL 是唯一事实来源。
func (uc *UserUsecase) UpdateUserStats(ctx context.Context, cmd *UpdateUserStatsCommand) (*UpdateUserStatsResult, error) {
	if cmd.UserId <= 0 {
		return nil, errors.New("用户ID无效")
	}
	updates := make(map[string]interface{})
	if cmd.FollowCount != nil {
		updates["follow_count"] = *cmd.FollowCount
	}
	if cmd.FollowerCount != nil {
		updates["follower_count"] = *cmd.FollowerCount
	}
	if cmd.BeLikedCount != nil {
		updates["be_liked_count"] = *cmd.BeLikedCount
	}
	if cmd.WorkCount != nil {
		updates["work_count"] = *cmd.WorkCount
	}
	if cmd.CollectionCount != nil {
		updates["collection_count"] = *cmd.CollectionCount
	}

	if len(updates) == 0 {
		return &UpdateUserStatsResult{}, nil
	}

	if err := uc.repo.UpdateUserStats(ctx, cmd.UserId, updates); err != nil {
		return nil, err
	}
	return &UpdateUserStatsResult{}, nil
}

// UpdateUserLastOnlineTime 更新用户最后上线时间
func (uc *UserUsecase) UpdateUserLastOnlineTime(ctx context.Context, cmd *UpdateUserLastOnlineTimeCommand) (*UpdateUserLastOnlineTimeResult, error) {
	lastOnlineTime, err := time.Parse("2006-01-02 15:04:05", cmd.LastOnlineTime)
	if err != nil {
		return nil, err
	}

	err = uc.repo.UpdateUserLastOnlineTime(ctx, cmd.UserId, lastOnlineTime)
	if err != nil {
		return nil, err
	}

	return &UpdateUserLastOnlineTimeResult{}, nil
}
