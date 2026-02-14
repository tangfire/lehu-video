package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type UserInfo struct {
	ID             int64
	Name           string
	Nickname       string
	Avatar         string
	Signature      string
	Gender         int32
	OnlineStatus   int32
	LastOnlineTime time.Time
}

type FriendRelation struct {
	ID          int64
	UserID      int64
	FriendID    int64
	Status      int32
	Remark      string
	GroupName   string
	IsFollowing bool
	IsFollower  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (f *FriendRelation) GenerateId() {
	f.ID = int64(uuid.New().ID())
}

type FriendApply struct {
	ID          int64
	ApplicantID int64
	ReceiverID  int64
	ApplyReason string
	Status      int32
	HandledAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (f *FriendApply) GenerateId() {
	f.ID = int64(uuid.New().ID())
}

// 用户在线状态领域对象
type UserOnlineStatus struct {
	ID             int64
	UserID         int64
	OnlineStatus   int32
	DeviceType     string
	LastOnlineTime time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type FriendInfo struct {
	ID        int64
	Friend    *UserInfo
	Remark    string
	GroupName string
	Status    int32
	CreatedAt time.Time
}

type FriendApplyInfo struct {
	ID          int64
	Applicant   *UserInfo
	Receiver    *UserInfo
	ApplyReason string
	Status      int32
	HandledAt   *time.Time
	CreatedAt   time.Time
}

// Commands and Queries
type SearchUsersQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

type SearchUsersResult struct {
	Users []*UserInfo
	Total int64
}

type SendFriendApplyCommand struct {
	ApplicantID int64
	ReceiverID  int64
	ApplyReason string
}

type SendFriendApplyResult struct {
	ApplyID int64
}

type HandleFriendApplyCommand struct {
	ApplyID   int64
	HandlerID int64
	Accept    bool
}

type HandleFriendApplyResult struct{}

type ListFriendAppliesQuery struct {
	UserID int64
	Page   int
	Limit  int
	Status *int32
}

type ListFriendAppliesResult struct {
	Applies []*FriendApply
	Total   int64
}

type ListFriendsQuery struct {
	UserID    int64
	Page      int
	Limit     int
	GroupName *string
}

type ListFriendsResult struct {
	Friends []*FriendRelation
	Total   int64
}

type DeleteFriendCommand struct {
	UserID   int64
	FriendID int64
}

type DeleteFriendResult struct{}

type UpdateFriendRemarkCommand struct {
	UserID   int64
	FriendID int64
	Remark   string
}

type UpdateFriendRemarkResult struct{}

type SetFriendGroupCommand struct {
	UserID    int64
	FriendID  int64
	GroupName string
}

type SetFriendGroupResult struct{}

type CheckFriendRelationQuery struct {
	UserID   int64
	TargetID int64
}

type CheckFriendRelationResult struct {
	IsFriend bool
	Status   int32
}

type GetUserOnlineStatusQuery struct {
	UserID int64
}

type GetUserOnlineStatusResult struct {
	Status         int32
	LastOnlineTime time.Time
}

type BatchGetUserOnlineStatusQuery struct {
	UserIDs []int64
}

type BatchGetUserOnlineStatusResult struct {
	Statuses map[int64]int32 // user_id -> status
}

type UpdateUserOnlineStatusCommand struct {
	UserID     int64
	Status     int32
	DeviceType string
}

type UpdateUserOnlineStatusResult struct{}

// 仓储接口
type FriendRepo interface {
	// 用户搜索和获取
	GetUserInfo(ctx context.Context, userID int64) (*UserInfo, error)
	BatchGetUserInfo(ctx context.Context, userIDs []int64) (map[int64]*UserInfo, error)

	// 好友关系
	CreateFriendRelation(ctx context.Context, relation *FriendRelation) error
	GetFriendRelation(ctx context.Context, userID, friendID int64) (*FriendRelation, error)
	UpdateFriendRelation(ctx context.Context, relation *FriendRelation) error
	DeleteFriendRelation(ctx context.Context, userID, friendID int64) error
	ListFriends(ctx context.Context, userID int64, offset, limit int, groupName *string) ([]*FriendRelation, int64, error)
	CheckFriendRelation(ctx context.Context, userID, friendID int64) (bool, error)

	// 好友申请
	CreateFriendApply(ctx context.Context, apply *FriendApply) error
	GetFriendApply(ctx context.Context, applyID int64) (*FriendApply, error)
	UpdateFriendApply(ctx context.Context, apply *FriendApply) error
	ListFriendApplies(ctx context.Context, userID int64, status *int32, offset, limit int) ([]*FriendApply, int64, error)
	CheckPendingApply(ctx context.Context, applicantID int64, receiverID int64) (bool, error)

	// 在线状态
	UpdateUserOnlineStatus(ctx context.Context, userID int64, status int32, deviceType string) error
	BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]*UserOnlineStatus, error)
	GetUserOnlineStatus(ctx context.Context, userID int64) (*UserOnlineStatus, error)
}

// Usecase
type FriendUsecase struct {
	repo FriendRepo
	log  *log.Helper
}

func NewFriendUsecase(repo FriendRepo, logger log.Logger) *FriendUsecase {
	return &FriendUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// SendFriendApply 发送好友申请
func (uc *FriendUsecase) SendFriendApply(ctx context.Context, cmd *SendFriendApplyCommand) (*SendFriendApplyResult, error) {
	// 参数验证
	if cmd.ApplicantID == 0 || cmd.ReceiverID == 0 {
		return nil, errors.New("用户ID不能为空")
	}
	if cmd.ApplicantID == cmd.ReceiverID {
		return nil, errors.New("不能添加自己为好友")
	}

	// 检查是否已经是好友
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.ApplicantID, cmd.ReceiverID)
	if err != nil {
		return nil, err
	}
	if relation != nil && relation.Status == 1 {
		return nil, errors.New("已经是好友")
	}

	// 检查是否有待处理的申请
	hasPending, err := uc.repo.CheckPendingApply(ctx, cmd.ApplicantID, cmd.ReceiverID)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, errors.New("已经发送过好友申请，请等待对方处理")
	}

	// 创建申请
	now := time.Now()
	apply := &FriendApply{
		ApplicantID: cmd.ApplicantID,
		ReceiverID:  cmd.ReceiverID,
		ApplyReason: cmd.ApplyReason,
		Status:      0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// 生成ID（可使用雪花算法，这里简化）
	apply.GenerateId()

	if err := uc.repo.CreateFriendApply(ctx, apply); err != nil {
		return nil, err
	}
	return &SendFriendApplyResult{ApplyID: apply.ID}, nil
}

// HandleFriendApply 处理好友申请
func (uc *FriendUsecase) HandleFriendApply(ctx context.Context, cmd *HandleFriendApplyCommand) (*HandleFriendApplyResult, error) {
	// 获取申请
	apply, err := uc.repo.GetFriendApply(ctx, cmd.ApplyID)
	if err != nil {
		return nil, err
	}
	if apply == nil {
		return nil, errors.New("申请不存在")
	}
	if apply.ReceiverID != cmd.HandlerID {
		return nil, errors.New("无权处理此申请")
	}
	if apply.Status != 0 {
		return nil, errors.New("申请已处理")
	}

	now := time.Now()
	apply.HandledAt = &now
	if cmd.Accept {
		apply.Status = 1
		// 创建双向好友关系
		relation1 := &FriendRelation{
			UserID:      apply.ApplicantID,
			FriendID:    apply.ReceiverID,
			Status:      1,
			IsFollowing: true,
			IsFollower:  true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		relation2 := &FriendRelation{
			UserID:      apply.ReceiverID,
			FriendID:    apply.ApplicantID,
			Status:      1,
			IsFollowing: true,
			IsFollower:  true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		relation1.GenerateId()
		relation2.GenerateId()

		if err := uc.repo.CreateFriendRelation(ctx, relation1); err != nil {
			return nil, err
		}
		if err := uc.repo.CreateFriendRelation(ctx, relation2); err != nil {
			// 回滚第一个
			_ = uc.repo.DeleteFriendRelation(ctx, relation1.UserID, relation1.FriendID)
			return nil, err
		}
	} else {
		apply.Status = 2
	}
	apply.UpdatedAt = now
	if err := uc.repo.UpdateFriendApply(ctx, apply); err != nil {
		return nil, err
	}
	return &HandleFriendApplyResult{}, nil
}

// ListFriendApplies 获取好友申请列表
func (uc *FriendUsecase) ListFriendApplies(ctx context.Context, query *ListFriendAppliesQuery) (*ListFriendAppliesResult, error) {
	if query.UserID == 0 {
		return nil, errors.New("用户ID不能为空")
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	offset := (query.Page - 1) * query.Limit

	applies, total, err := uc.repo.ListFriendApplies(ctx, query.UserID, query.Status, offset, query.Limit)
	if err != nil {
		return nil, err
	}
	return &ListFriendAppliesResult{Applies: applies, Total: total}, nil
}

// ListFriends 只返回好友关系，不填充用户信息
func (uc *FriendUsecase) ListFriends(ctx context.Context, query *ListFriendsQuery) (*ListFriendsResult, error) {
	if query.UserID == 0 {
		return nil, errors.New("用户ID不能为空")
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	offset := (query.Page - 1) * query.Limit

	relations, total, err := uc.repo.ListFriends(ctx, query.UserID, offset, query.Limit, query.GroupName)
	if err != nil {
		return nil, err
	}
	return &ListFriendsResult{Friends: relations, Total: total}, nil
}

// DeleteFriend 删除好友
func (uc *FriendUsecase) DeleteFriend(ctx context.Context, cmd *DeleteFriendCommand) (*DeleteFriendResult, error) {
	// 检查关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		return nil, err
	}
	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}
	// 双向删除
	if err := uc.repo.DeleteFriendRelation(ctx, cmd.UserID, cmd.FriendID); err != nil {
		return nil, err
	}
	_ = uc.repo.DeleteFriendRelation(ctx, cmd.FriendID, cmd.UserID)
	return &DeleteFriendResult{}, nil
}

// UpdateFriendRemark 更新好友备注
func (uc *FriendUsecase) UpdateFriendRemark(ctx context.Context, cmd *UpdateFriendRemarkCommand) (*UpdateFriendRemarkResult, error) {
	if cmd.Remark == "" {
		return nil, errors.New("备注不能为空")
	}
	if len(cmd.Remark) > 100 {
		return nil, errors.New("备注不能超过100个字符")
	}
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		return nil, err
	}
	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}
	relation.Remark = cmd.Remark
	relation.UpdatedAt = time.Now()
	if err := uc.repo.UpdateFriendRelation(ctx, relation); err != nil {
		return nil, err
	}
	return &UpdateFriendRemarkResult{}, nil
}

// SetFriendGroup 设置好友分组
func (uc *FriendUsecase) SetFriendGroup(ctx context.Context, cmd *SetFriendGroupCommand) (*SetFriendGroupResult, error) {
	if cmd.GroupName == "" {
		return nil, errors.New("分组名称不能为空")
	}
	if len(cmd.GroupName) > 50 {
		return nil, errors.New("分组名称不能超过50个字符")
	}
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		return nil, err
	}
	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}
	relation.GroupName = cmd.GroupName
	relation.UpdatedAt = time.Now()
	if err := uc.repo.UpdateFriendRelation(ctx, relation); err != nil {
		return nil, err
	}
	return &SetFriendGroupResult{}, nil
}

// CheckFriendRelation 检查好友关系
func (uc *FriendUsecase) CheckFriendRelation(ctx context.Context, query *CheckFriendRelationQuery) (*CheckFriendRelationResult, error) {
	isFriend, err := uc.repo.CheckFriendRelation(ctx, query.UserID, query.TargetID)
	if err != nil {
		return nil, err
	}
	result := &CheckFriendRelationResult{IsFriend: isFriend}
	if isFriend {
		rel, _ := uc.repo.GetFriendRelation(ctx, query.UserID, query.TargetID)
		if rel != nil {
			result.Status = rel.Status
		}
	}
	return result, nil
}

// GetUserOnlineStatus 获取单个用户在线状态
func (uc *FriendUsecase) GetUserOnlineStatus(ctx context.Context, query *GetUserOnlineStatusQuery) (*GetUserOnlineStatusResult, error) {
	status, err := uc.repo.GetUserOnlineStatus(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return &GetUserOnlineStatusResult{Status: 0, LastOnlineTime: time.Now()}, nil
	}
	return &GetUserOnlineStatusResult{Status: status.OnlineStatus, LastOnlineTime: status.LastOnlineTime}, nil
}

// BatchGetUserOnlineStatus 批量获取用户在线状态
func (uc *FriendUsecase) BatchGetUserOnlineStatus(ctx context.Context, query *BatchGetUserOnlineStatusQuery) (*BatchGetUserOnlineStatusResult, error) {
	if len(query.UserIDs) == 0 {
		return &BatchGetUserOnlineStatusResult{Statuses: make(map[int64]int32)}, nil
	}
	if len(query.UserIDs) > 100 {
		return nil, errors.New("单次最多查询100个用户")
	}
	statuses, err := uc.repo.BatchGetUserOnlineStatus(ctx, query.UserIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int32)
	for uid, s := range statuses {
		result[uid] = s.OnlineStatus
	}
	return &BatchGetUserOnlineStatusResult{Statuses: result}, nil
}

// UpdateUserOnlineStatus 更新用户在线状态
func (uc *FriendUsecase) UpdateUserOnlineStatus(ctx context.Context, cmd *UpdateUserOnlineStatusCommand) (*UpdateUserOnlineStatusResult, error) {
	err := uc.repo.UpdateUserOnlineStatus(ctx, cmd.UserID, cmd.Status, cmd.DeviceType)
	if err != nil {
		return nil, err
	}
	return &UpdateUserOnlineStatusResult{}, nil
}
