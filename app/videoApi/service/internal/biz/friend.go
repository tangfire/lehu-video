package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

// 好友信息
type FriendInfo struct {
	ID        int64     `json:"id"`
	Friend    *UserInfo `json:"friend"`
	Remark    string    `json:"remark"`
	GroupName string    `json:"group_name"`
	Status    int32     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// 好友申请
type FriendApply struct {
	ID          int64      `json:"id"`
	ApplicantID int64      `json:"applicant_id"`
	ReceiverID  int64      `json:"receiver_id"`
	ApplyReason string     `json:"apply_reason"`
	Status      int32      `json:"status"`
	HandledAt   *time.Time `json:"handled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// 输入输出结构体
type SearchUsersInput struct {
	Keyword  string
	Page     int32
	PageSize int32
}

type SearchUsersOutput struct {
	Users []*UserInfo
	Total int64
}

type SendFriendApplyInput struct {
	ReceiverID  int64
	ApplyReason string
}

type SendFriendApplyOutput struct {
	ApplyID int64
}

type HandleFriendApplyInput struct {
	ApplyID int64
	Accept  bool
}

type ListFriendAppliesInput struct {
	Page     int32
	PageSize int32
	Status   *int32
}

type ListFriendAppliesOutput struct {
	Applies []*FriendApply
	Total   int64
}

type ListFriendsInput struct {
	Page      int32
	PageSize  int32
	GroupName *string
}

type ListFriendsOutput struct {
	Friends []*FriendInfo
	Total   int64
}

type DeleteFriendInput struct {
	FriendID int64
}

type UpdateFriendRemarkInput struct {
	FriendID int64
	Remark   string
}

type SetFriendGroupInput struct {
	FriendID  int64
	GroupName string
}

type CheckFriendRelationInput struct {
	TargetID int64
}

type CheckFriendRelationOutput struct {
	IsFriend bool
	Status   int32
}

type GetUserOnlineStatusInput struct {
	UserID int64
}

type GetUserOnlineStatusOutput struct {
	OnlineStatus   int32
	LastOnlineTime string
}

type BatchGetUserOnlineStatusInput struct {
	UserIDs []int64
}

type BatchGetUserOnlineStatusOutput struct {
	OnlineStatus map[int64]int32
}

// FriendUsecase 好友用例
type FriendUsecase struct {
	chat ChatAdapter
	core CoreAdapter
	log  *log.Helper
}

func NewFriendUsecase(chat ChatAdapter, core CoreAdapter, logger log.Logger) *FriendUsecase {
	return &FriendUsecase{
		chat: chat,
		core: core,
		log:  log.NewHelper(logger),
	}
}

// SearchUsers 搜索用户
func (uc *FriendUsecase) SearchUsers(ctx context.Context, input *SearchUsersInput) (*SearchUsersOutput, error) {
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

	total, users, err := uc.chat.SearchUsers(ctx, input.Keyword, input.Page, input.PageSize)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("搜索用户失败: %v", err)
		return nil, errors.New("搜索用户失败")
	}

	return &SearchUsersOutput{
		Users: users,
		Total: total,
	}, nil
}

// SendFriendApply 发送好友申请
func (uc *FriendUsecase) SendFriendApply(ctx context.Context, input *SendFriendApplyInput) (*SendFriendApplyOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	applyID, err := uc.chat.SendFriendApply(ctx, userID, input.ReceiverID, input.ApplyReason)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("发送好友申请失败: %v", err)
		return nil, err
	}

	return &SendFriendApplyOutput{
		ApplyID: applyID,
	}, nil
}

// HandleFriendApply 处理好友申请
func (uc *FriendUsecase) HandleFriendApply(ctx context.Context, input *HandleFriendApplyInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.HandleFriendApply(ctx, input.ApplyID, userID, input.Accept)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("处理好友申请失败: %v", err)
		return err
	}

	return nil
}

// ListFriendApplies 获取好友申请列表
func (uc *FriendUsecase) ListFriendApplies(ctx context.Context, input *ListFriendAppliesInput) (*ListFriendAppliesOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

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

	total, applies, err := uc.chat.ListFriendApplies(ctx, userID, input.Page, input.PageSize, input.Status)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取好友申请列表失败: %v", err)
		return nil, errors.New("获取好友申请列表失败")
	}

	return &ListFriendAppliesOutput{
		Applies: applies,
		Total:   total,
	}, nil
}

// ListFriends 获取好友列表
func (uc *FriendUsecase) ListFriends(ctx context.Context, input *ListFriendsInput) (*ListFriendsOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

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

	// 1. 从chat服务获取好友列表
	total, chatFriends, err := uc.chat.ListFriends(ctx, userID, input.Page, input.PageSize, input.GroupName)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取好友列表失败: %v", err)
		return nil, errors.New("获取好友列表失败")
	}

	// 2. 提取好友ID列表
	var friendIDs []int64
	for _, friend := range chatFriends {
		friendIDs = append(friendIDs, friend.Friend.Id)
	}

	// 3. 从core服务获取好友详细信息
	userInfoList, err := uc.core.GetUserInfoByIdList(ctx, friendIDs)
	userInfos := make(map[int64]*UserInfo)
	for _, userInfo := range userInfoList {
		userInfos[userInfo.Id] = userInfo
	}
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取好友详细信息失败: %v", err)
		// 不返回错误，只记录日志
		uc.log.WithContext(ctx).Warnf("无法获取好友详细信息，将使用基础信息")
		userInfos = make(map[int64]*UserInfo)
	}

	// 4. 合并数据
	friends := make([]*FriendInfo, 0, len(chatFriends))
	for _, chatFriend := range chatFriends {
		friendInfo := &FriendInfo{
			ID:        chatFriend.ID,
			Remark:    chatFriend.Remark,
			GroupName: chatFriend.GroupName,
			Status:    chatFriend.Status,
			CreatedAt: chatFriend.CreatedAt,
		}

		// 填充好友信息
		if userInfo, ok := userInfos[chatFriend.Friend.Id]; ok {
			friendInfo.Friend = userInfo
		} else {
			// 如果没有获取到详细信息，使用基础信息
			friendInfo.Friend = &UserInfo{
				Id: chatFriend.Friend.Id,
			}
		}

		friends = append(friends, friendInfo)
	}

	return &ListFriendsOutput{
		Friends: friends,
		Total:   total,
	}, nil
}

// DeleteFriend 删除好友
func (uc *FriendUsecase) DeleteFriend(ctx context.Context, input *DeleteFriendInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.DeleteFriend(ctx, userID, input.FriendID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("删除好友失败: %v", err)
		return errors.New("删除好友失败")
	}

	return nil
}

// UpdateFriendRemark 更新好友备注
func (uc *FriendUsecase) UpdateFriendRemark(ctx context.Context, input *UpdateFriendRemarkInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.UpdateFriendRemark(ctx, userID, input.FriendID, input.Remark)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("更新好友备注失败: %v", err)
		return errors.New("更新好友备注失败")
	}

	return nil
}

// SetFriendGroup 设置好友分组
func (uc *FriendUsecase) SetFriendGroup(ctx context.Context, input *SetFriendGroupInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.SetFriendGroup(ctx, userID, input.FriendID, input.GroupName)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("设置好友分组失败: %v", err)
		return errors.New("设置好友分组失败")
	}

	return nil
}

// CheckFriendRelation 检查好友关系
func (uc *FriendUsecase) CheckFriendRelation(ctx context.Context, input *CheckFriendRelationInput) (*CheckFriendRelationOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	isFriend, status, err := uc.chat.CheckFriendRelation(ctx, userID, input.TargetID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("检查好友关系失败: %v", err)
		return nil, errors.New("检查好友关系失败")
	}

	return &CheckFriendRelationOutput{
		IsFriend: isFriend,
		Status:   status,
	}, nil
}

// GetUserOnlineStatus 获取用户在线状态
func (uc *FriendUsecase) GetUserOnlineStatus(ctx context.Context, input *GetUserOnlineStatusInput) (*GetUserOnlineStatusOutput, error) {
	status, lastOnlineTime, err := uc.chat.GetUserOnlineStatus(ctx, input.UserID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取用户在线状态失败: %v", err)
		return nil, errors.New("获取用户在线状态失败")
	}

	return &GetUserOnlineStatusOutput{
		OnlineStatus:   status,
		LastOnlineTime: lastOnlineTime,
	}, nil
}

// BatchGetUserOnlineStatus 批量获取用户在线状态
func (uc *FriendUsecase) BatchGetUserOnlineStatus(ctx context.Context, input *BatchGetUserOnlineStatusInput) (*BatchGetUserOnlineStatusOutput, error) {
	onlineStatus, err := uc.chat.BatchGetUserOnlineStatus(ctx, input.UserIDs)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("批量获取用户在线状态失败: %v", err)
		return nil, errors.New("批量获取用户在线状态失败")
	}

	return &BatchGetUserOnlineStatusOutput{
		OnlineStatus: onlineStatus,
	}, nil
}
