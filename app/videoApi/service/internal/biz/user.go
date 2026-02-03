package biz

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ==================== 基础结构体 ====================

type UserBaseInfo struct {
	ID              string
	Name            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	Mobile          string
	Email           string
	Gender          int32
	FollowCount     int64
	FollowerCount   int64
	TotalFavorited  int64
	WorkCount       int64
	FavoriteCount   int64
	CreatedAt       string
	UpdatedAt       string
}

type UserSocialInfo struct {
	UserID         string
	OnlineStatus   int32
	LastOnlineTime string
	DeviceType     string
	SessionInfo    map[string]string
}

type UserRelationInfo struct {
	UserID       string
	TargetUserID string
	IsFollowing  bool
	IsFollower   bool
	IsFriend     bool
	FriendStatus int32
	Remark       string
	GroupName    string
	CreatedAt    string
}

type UserInfo struct {
	ID              string
	Name            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	Gender          int32
	FollowCount     int64
	FollowerCount   int64
	TotalFavorited  int64
	WorkCount       int64
	FavoriteCount   int64
	CreatedAt       string
	OnlineStatus    int32
	LastOnlineTime  time.Time // 修正：保持 time.Time 类型
	IsFollowing     bool
	IsFollower      bool
	IsFriend        bool
	FriendRemark    string
	FriendGroup     string
	Mobile          string
	Email           string
}

// ==================== 输入输出结构 ====================

type GetUserInfoInput struct {
	UserID         string
	AccountID      string
	IncludePrivate bool
}

type GetUserInfoOutput struct {
	User *UserInfo
}

type BatchGetUserInfoInput struct {
	UserIDs         []string
	CurrentUserID   string
	IncludePrivate  bool
	IncludeRelation bool
}

type BatchGetUserInfoOutput struct {
	Users map[string]*UserInfo
}

type SearchUsersInput struct {
	Keyword  string
	Page     int32
	PageSize int32
}

type SearchUsersOutput struct {
	Users []*UserInfo
	Total int64
}

type RegisterInput struct {
	Mobile   string
	Email    string
	Password string
	CodeId   int64
	Code     string
}

type RegisterOutput struct {
	UserID string
}

type LoginInput struct {
	Mobile   string
	Email    string
	Password string
}

type LoginOutput struct {
	Token string
	User  *UserBaseInfo
}

type UpdateUserInfoInput struct {
	UserID          string
	Name            string
	Nickname        string
	Avatar          string
	BackgroundImage string
	Signature       string
	Gender          int32
}

type UpdateUserInfoOutput struct{}

type BindUserVoucherInput struct {
	UserID      string
	VoucherType string
	Voucher     string
	CodeId      int64
	Code        string
}

type BindUserVoucherOutput struct{}

type UnbindUserVoucherInput struct {
	UserID      string
	VoucherType string
}

type UnbindUserVoucherOutput struct{}

// ==================== Usecase ====================

type UserUsecase struct {
	base BaseAdapter
	core CoreAdapter
	chat ChatAdapter
	log  *log.Helper
}

func NewUserUsecase(base BaseAdapter, core CoreAdapter, chat ChatAdapter, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		base: base,
		core: core,
		chat: chat,
		log:  log.NewHelper(logger),
	}
}

// GetVerificationCode 生成验证码
func (uc *UserUsecase) GetVerificationCode(ctx context.Context) (int64, error) {
	// 默认生成6位数字验证码，10分钟过期
	codeId, err := uc.base.CreateVerificationCode(ctx, 6, 60*10)
	if err != nil {
		return 0, err
	}
	return codeId, nil
}

// Register 用户注册
func (uc *UserUsecase) Register(ctx context.Context, input *RegisterInput) (*RegisterOutput, error) {
	// 1. 验证验证码
	err := uc.base.ValidateVerificationCode(ctx, input.CodeId, input.Code)
	if err != nil {
		return nil, errors.New("验证码错误")
	}

	// 2. 注册账户
	accountID, err := uc.base.Register(ctx, input.Mobile, input.Email, input.Password)
	if err != nil {
		return nil, fmt.Errorf("注册账户失败: %w", err)
	}

	// 3. 创建用户信息
	userID, err := uc.core.CreateUser(ctx, input.Mobile, input.Email, accountID)
	if err != nil {
		return nil, fmt.Errorf("创建用户信息失败: %w", err)
	}

	return &RegisterOutput{UserID: userID}, nil
}

// Login 用户登录
func (uc *UserUsecase) Login(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
	// 1. 验证账户
	accountID, err := uc.base.CheckAccount(ctx, input.Mobile, input.Email, input.Password)
	if err != nil {
		return nil, errors.New("账户或密码错误")
	}

	// 2. 通过账户ID查找用户ID（这里需要调用core服务，但缺少相关方法）

	baseUser, err := uc.core.GetUserBaseInfo(ctx, "0", accountID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 4. 生成token
	token, err := uc.setToken2Header(ctx, claims.New(baseUser.ID))
	fmt.Println("token = " + token)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		Token: token,
		User:  baseUser,
	}, nil
}

func (uc *UserUsecase) setToken2Header(ctx context.Context, claim *claims.Claims) (string, error) {
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte("fireshine"))
	if err != nil {
		return "", err
	}

	if header, ok := transport.FromServerContext(ctx); ok {
		header.ReplyHeader().Set("Authorization", "Bearer "+tokenString)
		return tokenString, nil
	}

	return "", jwt.ErrWrongContext
}

// GetCompleteUserInfo 获取用户完整信息（聚合）
func (uc *UserUsecase) GetCompleteUserInfo(ctx context.Context, input *GetUserInfoInput) (*GetUserInfoOutput, error) {
	// 1. 获取基础信息
	baseInfo, err := uc.core.GetUserBaseInfo(ctx, input.UserID, input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("获取用户基础信息失败: %w", err)
	}

	// 2. 获取社交信息
	var socialInfo *UserSocialInfo
	if uc.chat != nil {
		socialInfo, _ = uc.chat.GetUserOnlineStatus(ctx, input.UserID)
	}

	currentUserID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 获取关系信息
	var relationInfo *UserRelationInfo
	if currentUserID != "" && currentUserID != input.UserID && uc.chat != nil {
		relationInfo, _ = uc.chat.GetUserRelation(ctx, currentUserID, input.UserID)
	}

	// 4. 聚合数据
	user := uc.aggregateUserInfo(baseInfo, socialInfo, relationInfo)

	// 5. 处理隐私字段
	if !input.IncludePrivate && currentUserID != input.UserID {
		user.Mobile = ""
		user.Email = ""
	}

	return &GetUserInfoOutput{User: user}, nil
}

// BatchGetUserInfo 批量获取用户信息
func (uc *UserUsecase) BatchGetUserInfo(ctx context.Context, input *BatchGetUserInfoInput) (*BatchGetUserInfoOutput, error) {
	if len(input.UserIDs) == 0 {
		return &BatchGetUserInfoOutput{Users: make(map[string]*UserInfo)}, nil
	}

	// 限制批量查询数量
	if len(input.UserIDs) > 100 {
		return nil, errors.New("单次批量查询最多100个用户")
	}

	// 1. 批量获取基础信息
	baseInfos, err := uc.core.BatchGetUserBaseInfo(ctx, input.UserIDs)
	if err != nil {
		return nil, fmt.Errorf("批量获取用户基础信息失败: %w", err)
	}

	// 2. 批量获取社交信息
	var socialInfos map[string]*UserSocialInfo
	if uc.chat != nil {
		socialInfos, _ = uc.chat.BatchGetUserOnlineStatus(ctx, input.UserIDs)
	}

	// 3. 批量获取关系信息
	var relationInfos map[string]*UserRelationInfo
	if input.IncludeRelation && input.CurrentUserID != "" && uc.chat != nil {
		relationInfos, _ = uc.chat.BatchGetUserRelations(ctx, input.CurrentUserID, input.UserIDs)
	}

	// 4. 聚合数据
	result := make(map[string]*UserInfo)
	for _, baseInfo := range baseInfos {
		if baseInfo == nil {
			continue
		}

		userID := baseInfo.ID
		socialInfo := socialInfos[userID]
		var relationInfo *UserRelationInfo
		if relationInfos != nil {
			relationInfo = relationInfos[userID]
		}

		user := uc.aggregateUserInfo(baseInfo, socialInfo, relationInfo)

		// 处理隐私字段
		if !input.IncludePrivate && input.CurrentUserID != userID {
			user.Mobile = ""
			user.Email = ""
		}

		result[userID] = user
	}

	return &BatchGetUserInfoOutput{Users: result}, nil
}

// UpdateUserInfo 更新用户信息
func (uc *UserUsecase) UpdateUserInfo(ctx context.Context, input *UpdateUserInfoInput) (*UpdateUserInfoOutput, error) {
	err := uc.core.UpdateUserInfo(ctx,
		input.UserID,
		input.Name,
		input.Nickname,
		input.Avatar,
		input.BackgroundImage,
		input.Signature,
		input.Gender)
	if err != nil {
		return nil, err
	}

	return &UpdateUserInfoOutput{}, nil
}

// BindUserVoucher 绑定凭证
func (uc *UserUsecase) BindUserVoucher(ctx context.Context, input *BindUserVoucherInput) (*BindUserVoucherOutput, error) {
	// 1. 验证验证码
	err := uc.base.ValidateVerificationCode(ctx, input.CodeId, input.Code)
	if err != nil {
		return nil, errors.New("验证码错误")
	}

	// 2. 调用base服务绑定凭证
	err = uc.base.BindVoucher(ctx, input.UserID, input.VoucherType, input.Voucher)
	if err != nil {
		return nil, fmt.Errorf("绑定凭证失败: %w", err)
	}

	return &BindUserVoucherOutput{}, nil
}

// UnbindUserVoucher 解绑凭证
func (uc *UserUsecase) UnbindUserVoucher(ctx context.Context, input *UnbindUserVoucherInput) (*UnbindUserVoucherOutput, error) {
	// 调用base服务解绑凭证
	err := uc.base.UnbindVoucher(ctx, input.UserID, input.VoucherType)
	if err != nil {
		return nil, fmt.Errorf("解绑凭证失败: %w", err)
	}

	return &UnbindUserVoucherOutput{}, nil
}

// SearchUsers 搜索用户
func (uc *UserUsecase) SearchUsers(ctx context.Context, input *SearchUsersInput) (*SearchUsersOutput, error) {
	// 参数验证
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}

	// 1. 搜索用户
	total, baseUsers, err := uc.core.SearchUsers(ctx, input.Keyword, input.Page, input.PageSize)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}

	if len(baseUsers) == 0 {
		return &SearchUsersOutput{Users: []*UserInfo{}, Total: total}, nil
	}

	// 2. 提取用户ID
	userIDs := make([]string, 0, len(baseUsers))
	for _, user := range baseUsers {
		userIDs = append(userIDs, user.ID)
	}

	// 3. 批量获取社交信息
	var socialInfos map[string]*UserSocialInfo
	if uc.chat != nil {
		socialInfos, _ = uc.chat.BatchGetUserOnlineStatus(ctx, userIDs)
	}

	currentUserID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, err
	}

	// 4. 批量获取关系信息
	var relationInfos map[string]*UserRelationInfo
	if currentUserID != "" && uc.chat != nil {
		relationInfos, _ = uc.chat.BatchGetUserRelations(ctx, currentUserID, userIDs)
	}

	// 5. 聚合数据
	users := make([]*UserInfo, 0, len(baseUsers))
	for _, baseUser := range baseUsers {
		userID := baseUser.ID
		socialInfo := socialInfos[userID]
		var relationInfo *UserRelationInfo
		if relationInfos != nil {
			relationInfo = relationInfos[userID]
		}

		user := uc.aggregateUserInfo(baseUser, socialInfo, relationInfo)

		// 搜索结果中隐藏隐私字段
		user.Mobile = ""
		user.Email = ""

		users = append(users, user)
	}

	return &SearchUsersOutput{
		Users: users,
		Total: total,
	}, nil
}

// ==================== 辅助函数 ====================

func (uc *UserUsecase) aggregateUserInfo(baseInfo *UserBaseInfo, socialInfo *UserSocialInfo, relationInfo *UserRelationInfo) *UserInfo {
	if baseInfo == nil {
		return nil
	}

	user := &UserInfo{
		// 基础信息
		ID:              baseInfo.ID,
		Name:            baseInfo.Name,
		Nickname:        baseInfo.Nickname,
		Avatar:          baseInfo.Avatar,
		BackgroundImage: baseInfo.BackgroundImage,
		Signature:       baseInfo.Signature,
		Gender:          baseInfo.Gender,

		// 统计信息
		FollowCount:    baseInfo.FollowCount,
		FollowerCount:  baseInfo.FollowerCount,
		TotalFavorited: baseInfo.TotalFavorited,
		WorkCount:      baseInfo.WorkCount,
		FavoriteCount:  baseInfo.FavoriteCount,
		CreatedAt:      baseInfo.CreatedAt,

		// 隐私字段
		Mobile: baseInfo.Mobile,
		Email:  baseInfo.Email,
	}

	// 社交信息
	if socialInfo != nil {
		user.OnlineStatus = socialInfo.OnlineStatus
		// 转换时间字符串为 time.Time
		if socialInfo.LastOnlineTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", socialInfo.LastOnlineTime); err == nil {
				user.LastOnlineTime = t
			}
		}
	}

	// 关系信息
	if relationInfo != nil {
		user.IsFollowing = relationInfo.IsFollowing
		user.IsFollower = relationInfo.IsFollower
		user.IsFriend = relationInfo.IsFriend
		user.FriendRemark = relationInfo.Remark
		user.FriendGroup = relationInfo.GroupName
	}

	return user
}
