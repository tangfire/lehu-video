package biz

import (
	"context"
	"errors"
	"github.com/spf13/cast"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

// 好友信息
type FriendInfo struct {
	ID        string    `json:"id"`
	Friend    *UserInfo `json:"friend"`
	Remark    string    `json:"remark"`
	GroupName string    `json:"group_name"`
	Status    int32     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// FriendRelation 好友关系（来自chat）
type FriendRelation struct {
	ID        string
	FriendID  string
	Remark    string
	GroupName string
	Status    int32
	CreatedAt string
}

// 好友申请
type FriendApply struct {
	ID          string     `json:"id"`
	ApplicantID string     `json:"applicant_id"`
	ReceiverID  string     `json:"receiver_id"`
	ApplyReason string     `json:"apply_reason"`
	Status      int32      `json:"status"`
	HandledAt   *time.Time `json:"handled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SendFriendApplyInput struct {
	ReceiverID  string
	ApplyReason string
}

type SendFriendApplyOutput struct {
	ApplyID string
}

type HandleFriendApplyInput struct {
	ApplyID string
	Accept  bool
}

type ListFriendAppliesInput struct {
	Page     int32
	PageSize int32
	Status   *int32
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
	FriendID string
}

type UpdateFriendRemarkInput struct {
	FriendID string
	Remark   string
}

type SetFriendGroupInput struct {
	FriendID  string
	GroupName string
}

type CheckFriendRelationInput struct {
	TargetID string
}

type CheckFriendRelationOutput struct {
	IsFriend bool
	Status   int32
}

type GetUserOnlineStatusInput struct {
	UserID string
}

type GetUserOnlineStatusOutput struct {
	OnlineStatus   int32
	LastOnlineTime string
}

type BatchGetUserOnlineStatusInput struct {
	UserIDs []string
}

type BatchGetUserOnlineStatusOutput struct {
	OnlineStatus map[string]int32
}

// 在 biz/friend.go 中添加
type FriendApplyDetail struct {
	ID          string
	Applicant   *UserInfo
	Receiver    *UserInfo
	ApplyReason string
	Status      int32
	HandledAt   *time.Time
	CreatedAt   time.Time
}

type ListFriendAppliesOutput struct {
	Applies []*FriendApplyDetail
	Total   int64
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

// SendFriendApply 发送好友申请
func (uc *FriendUsecase) SendFriendApply(ctx context.Context, input *SendFriendApplyInput) (*SendFriendApplyOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	applyID, err := uc.chat.SendFriendApply(ctx, cast.ToString(userID), input.ReceiverID, input.ApplyReason)
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

	err = uc.chat.HandleFriendApply(ctx, input.ApplyID, cast.ToString(userID), input.Accept)
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

	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}

	// 从 chat 获取申请列表（只包含 ID）
	total, applies, err := uc.chat.ListFriendApplies(ctx, userID, input.Status, &PageStats{
		Page:     int(input.Page),
		PageSize: int(input.PageSize),
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取好友申请列表失败: %v", err)
		return nil, errors.New("获取好友申请列表失败")
	}

	if len(applies) == 0 {
		return &ListFriendAppliesOutput{Applies: []*FriendApplyDetail{}, Total: total}, nil
	}

	// 收集所有需要查询的用户ID（申请人+接收人）
	userIDSet := make(map[string]bool)
	for _, apply := range applies {
		userIDSet[apply.ApplicantID] = true
		userIDSet[apply.ReceiverID] = true
	}
	userIDs := make([]string, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	// 从 core 批量获取用户基础信息
	baseInfos, err := uc.core.BatchGetUserBaseInfo(ctx, userIDs)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("批量获取用户信息失败: %v", err)
		// 降级：不填充详细信息
	}

	// 从 chat 批量获取在线状态（可选）
	onlineStatusMap, _ := uc.chat.BatchGetUserOnlineStatus(ctx, userIDs)

	// 构建 userInfo 映射
	userInfoMap := make(map[string]*UserInfo)
	for _, base := range baseInfos {
		userInfoMap[base.ID] = &UserInfo{
			ID:             base.ID,
			Name:           base.Name,
			Nickname:       base.Nickname,
			Avatar:         base.Avatar,
			Signature:      base.Signature,
			Gender:         base.Gender,
			OnlineStatus:   onlineStatusMap[base.ID],
			LastOnlineTime: time.Now(), // 如果有最后在线时间可以填充
		}
	}

	// 组装返回结果
	details := make([]*FriendApplyDetail, 0, len(applies))
	for _, apply := range applies {
		detail := &FriendApplyDetail{
			ID:          apply.ID,
			ApplyReason: apply.ApplyReason,
			Status:      apply.Status,
			HandledAt:   apply.HandledAt,
			CreatedAt:   apply.CreatedAt,
		}
		if u, ok := userInfoMap[apply.ApplicantID]; ok {
			detail.Applicant = u
		} else {
			detail.Applicant = &UserInfo{ID: apply.ApplicantID}
		}
		if u, ok := userInfoMap[apply.ReceiverID]; ok {
			detail.Receiver = u
		} else {
			detail.Receiver = &UserInfo{ID: apply.ReceiverID}
		}
		details = append(details, detail)
	}

	return &ListFriendAppliesOutput{
		Applies: details,
		Total:   total,
	}, nil
}

// ListFriends 获取好友列表
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

	// 1. 从chat服务获取好友关系列表（只包含ID）
	total, relations, err := uc.chat.ListFriends(ctx, cast.ToString(userID), input.GroupName, &PageStats{
		Page:     int(input.Page),
		PageSize: int(input.PageSize),
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取好友列表失败: %v", err)
		return nil, errors.New("获取好友列表失败")
	}
	if len(relations) == 0 {
		return &ListFriendsOutput{Friends: []*FriendInfo{}, Total: total}, nil
	}

	// 2. 提取好友ID列表
	friendIDs := make([]string, 0, len(relations))
	for _, rel := range relations {
		friendIDs = append(friendIDs, rel.FriendID)
	}

	// 3. 从core服务批量获取好友详细信息（返回 []*UserBaseInfo）
	baseInfos, err := uc.core.BatchGetUserBaseInfo(ctx, friendIDs)
	userMap := make(map[string]*UserInfo)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取好友详细信息失败: %v", err)
		// 降级：不填充详细信息，userMap为空
	} else {
		for _, base := range baseInfos {
			// 将 UserBaseInfo 转换为 UserInfo（基础字段）
			user := &UserInfo{
				ID:        base.ID,
				Name:      base.Name,
				Nickname:  base.Nickname,
				Avatar:    base.Avatar,
				Signature: base.Signature,
				Gender:    base.Gender,
				// 其他字段如 OnlineStatus 后续填充
			}
			userMap[base.ID] = user
		}
	}

	// 4. 从chat服务批量获取在线状态
	onlineStatusMap, err := uc.chat.BatchGetUserOnlineStatus(ctx, friendIDs)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("批量获取在线状态失败: %v", err)
		onlineStatusMap = make(map[string]int32)
	}

	// 5. 合并数据
	friends := make([]*FriendInfo, 0, len(relations))
	for _, rel := range relations {
		user, ok := userMap[rel.FriendID]
		if !ok {
			// 如果core返回缺失，创建占位对象（只包含ID）
			user = &UserInfo{ID: rel.FriendID}
		}
		// 填充在线状态
		if status, ok := onlineStatusMap[rel.FriendID]; ok {
			user.OnlineStatus = status
		}
		// 转换时间
		createdAt, _ := time.Parse("2006-01-02 15:04:05", rel.CreatedAt)
		friendInfo := &FriendInfo{
			ID:        rel.ID,
			Friend:    user,
			Remark:    rel.Remark,
			GroupName: rel.GroupName,
			Status:    rel.Status,
			CreatedAt: createdAt,
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

	err = uc.chat.DeleteFriend(ctx, cast.ToString(userID), input.FriendID)
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

	err = uc.chat.UpdateFriendRemark(ctx, cast.ToString(userID), input.FriendID, input.Remark)
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

	err = uc.chat.SetFriendGroup(ctx, cast.ToString(userID), input.FriendID, input.GroupName)
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

	isFriend, status, err := uc.chat.CheckFriendRelation(ctx, cast.ToString(userID), input.TargetID)
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
	socialInfo, err := uc.chat.GetUserOnlineStatus(ctx, input.UserID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取用户在线状态失败: %v", err)
		return nil, errors.New("获取用户在线状态失败")
	}

	return &GetUserOnlineStatusOutput{
		OnlineStatus:   socialInfo.OnlineStatus,
		LastOnlineTime: socialInfo.LastOnlineTime,
	}, nil
}

// BatchGetUserOnlineStatus 批量获取用户在线状态
func (uc *FriendUsecase) BatchGetUserOnlineStatus(ctx context.Context, input *BatchGetUserOnlineStatusInput) (*BatchGetUserOnlineStatusOutput, error) {
	onlineStatus, err := uc.chat.BatchGetUserOnlineStatus(ctx, input.UserIDs)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("批量获取用户在线状态失败: %v", err)
		return nil, errors.New("批量获取用户在线状态失败")
	}

	// onlineStatus 已经是 map[string]int32，直接使用
	return &BatchGetUserOnlineStatusOutput{
		OnlineStatus: onlineStatus,
	}, nil
}
