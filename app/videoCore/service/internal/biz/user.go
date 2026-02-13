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
	repo        UserRepo
	counterRepo CounterRepo // 新增
	log         *log.Helper
}

func NewUserUsecase(repo UserRepo, counterRepo CounterRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{repo: repo, counterRepo: counterRepo, log: log.NewHelper(logger)}
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

// GetUserBaseInfo 优先从 Redis 获取计数器，降级从 DB 获取
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

	// todo 感觉这里是不是写的有点问题?因为这些count字段不是好像都已经在数据库了嘛？？好像不需要什么额外的查询
	// 从 Redis 获取计数器
	counters, err := uc.counterRepo.GetUserCounters(ctx, existUser.Id)
	if err != nil {
		uc.log.Warnf("从 Redis 获取用户计数器失败: %v, 使用 DB 值", err)
	} else if counters != nil {
		// 使用 Redis 中的值覆盖
		if val, ok := counters["follow_count"]; ok {
			existUser.FollowCount = val
		}
		if val, ok := counters["follower_count"]; ok {
			existUser.FollowerCount = val
		}
		if val, ok := counters["total_favorited"]; ok {
			existUser.TotalFavorited = val
		}
		if val, ok := counters["work_count"]; ok {
			existUser.WorkCount = val
		}
		if val, ok := counters["favorite_count"]; ok {
			existUser.FavoriteCount = val
		}
	} else {
		// Redis 中没有，则从 DB 加载并同步到 Redis
		go uc.syncUserCountersToRedis(context.Background(), existUser)
	}

	return &GetUserBaseInfoResult{User: existUser}, nil
}

// 将 DB 中的计数器同步到 Redis（异步）
func (uc *UserUsecase) syncUserCountersToRedis(ctx context.Context, user *User) {
	counters := map[string]int64{
		"follow_count":    user.FollowCount,
		"follower_count":  user.FollowerCount,
		"total_favorited": user.TotalFavorited,
		"work_count":      user.WorkCount,
		"favorite_count":  user.FavoriteCount,
	}
	err := uc.counterRepo.SetUserCounters(ctx, user.Id, counters)
	if err != nil {
		uc.log.Warnf("同步用户计数器到 Redis 失败: %v", err)
	}
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

// todo 是不是因为你点赞什么的，然后就去操作数据库，感觉压力太大了，所以这里有个先操作缓存的操作呢???
func (uc *UserUsecase) UpdateUserStats(ctx context.Context, cmd *UpdateUserStatsCommand) (*UpdateUserStatsResult, error) {
	updates := make(map[string]int64)
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

	// 直接更新 Redis（覆盖写）
	err := uc.counterRepo.SetUserCounters(ctx, cmd.UserId, updates)
	if err != nil {
		return nil, err
	}
	// 不再同步更新 DB，DB 由定时任务同步
	return &UpdateUserStatsResult{}, nil
}
