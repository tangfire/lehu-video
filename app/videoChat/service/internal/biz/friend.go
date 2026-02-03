package biz

import (
	"context"
	"errors"
	"github.com/spf13/cast"
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
	Applies []*FriendApplyInfo
	Total   int64
}

type ListFriendsQuery struct {
	UserID    int64
	Page      int
	Limit     int
	GroupName *string
}

type ListFriendsResult struct {
	Friends []*FriendInfo
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
	GetUserOnlineStatus(ctx context.Context, userID int64) (*UserInfo, error)
	BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]*UserInfo, error)
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
		uc.log.Errorf("检查好友关系失败: %v", err)
		return nil, errors.New("系统错误")
	}

	if relation != nil && relation.Status == 1 {
		return nil, errors.New("已经是好友")
	}

	// 检查是否有待处理的申请
	hasPending, err := uc.repo.CheckPendingApply(ctx, cmd.ApplicantID, cmd.ReceiverID)
	if err != nil {
		uc.log.Errorf("检查待处理申请失败: %v", err)
		return nil, errors.New("系统错误")
	}
	if hasPending {
		return nil, errors.New("已经发送过好友申请，请等待对方处理")
	}

	// 创建申请
	apply := &FriendApply{
		ApplicantID: cmd.ApplicantID,
		ReceiverID:  cmd.ReceiverID,
		ApplyReason: cmd.ApplyReason,
		Status:      0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	apply.GenerateId()

	err = uc.repo.CreateFriendApply(ctx, apply)
	if err != nil {
		uc.log.Errorf("创建好友申请失败: %v", err)
		return nil, errors.New("发送申请失败")
	}

	return &SendFriendApplyResult{
		ApplyID: apply.ID,
	}, nil
}

func (uc *FriendUsecase) HandleFriendApply(ctx context.Context, cmd *HandleFriendApplyCommand) (*HandleFriendApplyResult, error) {
	// 获取申请
	apply, err := uc.repo.GetFriendApply(ctx, cmd.ApplyID)
	if err != nil {
		uc.log.Errorf("查询好友申请失败: %v", err)
		return nil, errors.New("申请不存在")
	}
	if apply == nil {
		return nil, errors.New("申请不存在")
	}

	// 验证权限
	if apply.ReceiverID != cmd.HandlerID {
		return nil, errors.New("无权处理此申请")
	}

	if apply.Status != 0 {
		return nil, errors.New("申请已处理")
	}

	now := time.Now()
	apply.HandledAt = &now

	if cmd.Accept {
		// 同意申请
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

		relation1.GenerateId()

		relation2 := &FriendRelation{
			UserID:      apply.ReceiverID,
			FriendID:    apply.ApplicantID,
			Status:      1,
			IsFollowing: true,
			IsFollower:  true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		relation2.GenerateId()

		// 创建第一条关系
		if err := uc.repo.CreateFriendRelation(ctx, relation1); err != nil {
			uc.log.Errorf("创建好友关系失败: %v", err)
			return nil, errors.New("处理申请失败")
		}

		// 创建第二条关系
		if err := uc.repo.CreateFriendRelation(ctx, relation2); err != nil {
			uc.log.Errorf("创建好友关系失败: %v", err)
			// 回滚第一条
			_ = uc.repo.DeleteFriendRelation(ctx, relation1.UserID, relation1.FriendID)
			return nil, errors.New("处理申请失败")
		}
	} else {
		// 拒绝申请
		apply.Status = 2
	}

	apply.UpdatedAt = now
	if err := uc.repo.UpdateFriendApply(ctx, apply); err != nil {
		uc.log.Errorf("更新申请状态失败: %v", err)
		return nil, errors.New("处理申请失败")
	}

	return &HandleFriendApplyResult{}, nil
}

func (uc *FriendUsecase) ListFriendApplies(ctx context.Context, query *ListFriendAppliesQuery) (*ListFriendAppliesResult, error) {
	// 参数验证
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

	// 获取申请列表
	applies, total, err := uc.repo.ListFriendApplies(ctx, query.UserID, query.Status, offset, query.Limit)
	if err != nil {
		uc.log.Errorf("获取好友申请列表失败: %v", err)
		return nil, errors.New("获取申请列表失败")
	}

	// 转换为FriendApplyInfo
	result := make([]*FriendApplyInfo, 0, len(applies))
	for _, apply := range applies {
		// 获取申请人和接收人信息
		applicant, _ := uc.repo.GetUserInfo(ctx, apply.ApplicantID)
		receiver, _ := uc.repo.GetUserInfo(ctx, apply.ReceiverID)

		applyInfo := &FriendApplyInfo{
			ID:          cast.ToInt64(apply.ID),
			Applicant:   applicant,
			Receiver:    receiver,
			ApplyReason: apply.ApplyReason,
			Status:      apply.Status,
			HandledAt:   apply.HandledAt,
			CreatedAt:   apply.CreatedAt,
		}
		result = append(result, applyInfo)
	}

	return &ListFriendAppliesResult{
		Applies: result,
		Total:   total,
	}, nil
}

func (uc *FriendUsecase) ListFriends(ctx context.Context, query *ListFriendsQuery) (*ListFriendsResult, error) {
	// 参数验证
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

	// 获取好友关系列表
	relations, total, err := uc.repo.ListFriends(ctx, query.UserID, offset, query.Limit, query.GroupName)
	if err != nil {
		uc.log.Errorf("获取好友列表失败: %v", err)
		return nil, errors.New("获取好友列表失败")
	}

	// 获取好友ID
	friendIDs := make([]int64, 0, len(relations))
	for _, relation := range relations {
		friendIDs = append(friendIDs, relation.FriendID)
	}

	// 批量获取好友信息
	friendInfos := make(map[int64]*UserInfo)
	if len(friendIDs) > 0 {
		friendInfos, err = uc.repo.BatchGetUserInfo(ctx, friendIDs)
		if err != nil {
			uc.log.Errorf("批量获取好友信息失败: %v", err)
		}

		// 获取在线状态
		onlineStatus, _ := uc.repo.BatchGetUserOnlineStatus(ctx, friendIDs)
		for userID, user := range friendInfos {
			if onlineUser, ok := onlineStatus[userID]; ok {
				user.OnlineStatus = onlineUser.OnlineStatus
				user.LastOnlineTime = onlineUser.LastOnlineTime
			}
		}
	}

	// 组装结果
	friends := make([]*FriendInfo, 0, len(relations))
	for _, relation := range relations {
		friend := &FriendInfo{
			ID:        relation.ID,
			Remark:    relation.Remark,
			GroupName: relation.GroupName,
			Status:    relation.Status,
			CreatedAt: relation.CreatedAt,
		}

		if userInfo, ok := friendInfos[relation.FriendID]; ok {
			friend.Friend = userInfo
		}

		friends = append(friends, friend)
	}

	return &ListFriendsResult{
		Friends: friends,
		Total:   total,
	}, nil
}

func (uc *FriendUsecase) DeleteFriend(ctx context.Context, cmd *DeleteFriendCommand) (*DeleteFriendResult, error) {
	// 检查好友关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		uc.log.Errorf("检查好友关系失败: %v", err)
		return nil, errors.New("系统错误")
	}

	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}

	// 删除好友关系（双向）
	if err := uc.repo.DeleteFriendRelation(ctx, cmd.UserID, cmd.FriendID); err != nil {
		uc.log.Errorf("删除好友关系失败: %v", err)
		return nil, errors.New("删除好友失败")
	}

	// 删除反向关系
	_ = uc.repo.DeleteFriendRelation(ctx, cmd.FriendID, cmd.UserID)

	return &DeleteFriendResult{}, nil
}

func (uc *FriendUsecase) UpdateFriendRemark(ctx context.Context, cmd *UpdateFriendRemarkCommand) (*UpdateFriendRemarkResult, error) {
	// 参数验证
	if cmd.Remark == "" {
		return nil, errors.New("备注不能为空")
	}
	if len(cmd.Remark) > 100 {
		return nil, errors.New("备注不能超过100个字符")
	}

	// 获取好友关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		uc.log.Errorf("获取好友关系失败: %v", err)
		return nil, errors.New("系统错误")
	}

	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}

	// 更新备注
	relation.Remark = cmd.Remark
	relation.UpdatedAt = time.Now()

	if err := uc.repo.UpdateFriendRelation(ctx, relation); err != nil {
		uc.log.Errorf("更新好友备注失败: %v", err)
		return nil, errors.New("更新备注失败")
	}

	return &UpdateFriendRemarkResult{}, nil
}

func (uc *FriendUsecase) SetFriendGroup(ctx context.Context, cmd *SetFriendGroupCommand) (*SetFriendGroupResult, error) {
	// 参数验证
	if cmd.GroupName == "" {
		return nil, errors.New("分组名称不能为空")
	}
	if len(cmd.GroupName) > 50 {
		return nil, errors.New("分组名称不能超过50个字符")
	}

	// 获取好友关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		uc.log.Errorf("获取好友关系失败: %v", err)
		return nil, errors.New("系统错误")
	}

	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}

	// 更新分组
	relation.GroupName = cmd.GroupName
	relation.UpdatedAt = time.Now()

	if err := uc.repo.UpdateFriendRelation(ctx, relation); err != nil {
		uc.log.Errorf("设置好友分组失败: %v", err)
		return nil, errors.New("设置分组失败")
	}

	return &SetFriendGroupResult{}, nil
}

func (uc *FriendUsecase) CheckFriendRelation(ctx context.Context, query *CheckFriendRelationQuery) (*CheckFriendRelationResult, error) {
	relation, err := uc.repo.GetFriendRelation(ctx, query.UserID, query.TargetID)
	if err != nil {
		uc.log.Errorf("检查好友关系失败: %v", err)
		return nil, errors.New("系统错误")
	}

	result := &CheckFriendRelationResult{
		IsFriend: false,
		Status:   0,
	}

	if relation != nil && relation.Status == 1 {
		result.IsFriend = true
		result.Status = relation.Status
	}

	return result, nil
}

func (uc *FriendUsecase) GetUserOnlineStatus(ctx context.Context, query *GetUserOnlineStatusQuery) (*GetUserOnlineStatusResult, error) {
	user, err := uc.repo.GetUserOnlineStatus(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("获取用户在线状态失败: %v", err)
		return nil, errors.New("获取在线状态失败")
	}

	if user == nil {
		return &GetUserOnlineStatusResult{
			Status:         0,
			LastOnlineTime: time.Now(),
		}, nil
	}

	return &GetUserOnlineStatusResult{
		Status:         user.OnlineStatus,
		LastOnlineTime: user.LastOnlineTime,
	}, nil
}

func (uc *FriendUsecase) BatchGetUserOnlineStatus(ctx context.Context, query *BatchGetUserOnlineStatusQuery) (*BatchGetUserOnlineStatusResult, error) {
	if len(query.UserIDs) == 0 {
		return &BatchGetUserOnlineStatusResult{
			Statuses: make(map[int64]int32),
		}, nil
	}

	// 限制数量
	if len(query.UserIDs) > 100 {
		return nil, errors.New("单次最多查询100个用户")
	}

	users, err := uc.repo.BatchGetUserOnlineStatus(ctx, query.UserIDs)
	if err != nil {
		uc.log.Errorf("批量获取用户在线状态失败: %v", err)
		return nil, errors.New("获取在线状态失败")
	}

	statuses := make(map[int64]int32)
	for userID, user := range users {
		statuses[userID] = user.OnlineStatus
	}

	return &BatchGetUserOnlineStatusResult{
		Statuses: statuses,
	}, nil
}
