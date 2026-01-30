package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// 好友关系领域对象
type FriendRelation struct {
	ID        int64
	UserID    int64
	FriendID  int64
	Status    int32 // 0=待处理，1=已同意，2=已拒绝，3=已拉黑
	Remark    string
	GroupName string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 好友申请领域对象
type FriendApply struct {
	ID          int64
	ApplicantID int64
	ReceiverID  int64
	ApplyReason string
	Status      int32 // 0=待处理，1=已同意，2=已拒绝
	HandledAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 用户在线状态领域对象
type UserOnlineStatus struct {
	ID             int64
	UserID         int64
	Status         int32 // 0=离线，1=在线，2=忙碌，3=离开
	DeviceType     string
	LastOnlineTime time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// 用户信息领域对象
type UserInfo struct {
	ID             int64
	Username       string
	Nickname       string
	Avatar         string
	Signature      string
	Gender         int32
	OnlineStatus   int32
	LastOnlineTime time.Time
}

// ServiceError 服务错误
type ServiceError struct {
	Code    int
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("服务错误: %d - %s", e.Code, e.Message)
}

// Commands and Queries
type SearchUsersCommand struct {
	Keyword   string
	PageStats *PageStats
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
	UserID    int64
	PageStats *PageStats
	Status    *int32 // 可选
}

type ListFriendAppliesResult struct {
	Applies []*FriendApply
	Total   int64
}

type ListFriendsQuery struct {
	UserID    int64
	PageStats *PageStats
	GroupName *string
}

type ListFriendsResult struct {
	Friends []*FriendInfo
	Total   int64
}

type FriendInfo struct {
	ID        int64
	Friend    *UserInfo
	Remark    string
	GroupName string
	Status    int32
	CreatedAt time.Time
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
	OnlineStatus map[int64]int32 // user_id -> status
}

// 仓储接口
type FriendRepo interface {
	// 好友关系操作
	CreateFriendRelation(ctx context.Context, relation *FriendRelation) error
	GetFriendRelation(ctx context.Context, userID, friendID int64) (*FriendRelation, error)
	UpdateFriendRelation(ctx context.Context, relation *FriendRelation) error
	DeleteFriendRelation(ctx context.Context, id int64) error
	ListFriends(ctx context.Context, userID int64, offset, limit int, groupName *string) ([]*FriendRelation, error)
	CountFriends(ctx context.Context, userID int64, groupName *string) (int64, error)

	// 好友申请操作
	CreateFriendApply(ctx context.Context, apply *FriendApply) error
	GetFriendApply(ctx context.Context, id int64) (*FriendApply, error)
	UpdateFriendApply(ctx context.Context, apply *FriendApply) error
	ListFriendApplies(ctx context.Context, userID int64, status *int32, offset, limit int) ([]*FriendApply, error)
	CountFriendApplies(ctx context.Context, userID int64, status *int32) (int64, error)

	// 用户在线状态操作
	UpdateUserOnlineStatus(ctx context.Context, status *UserOnlineStatus) error
	GetUserOnlineStatus(ctx context.Context, userID int64) (*UserOnlineStatus, error)
	BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]*UserOnlineStatus, error)

	// 检查好友关系
	CheckFriendRelation(ctx context.Context, userID, targetID int64) (bool, int32, error)

	SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*UserInfo, int64, error)
	BatchGetUserInfo(ctx context.Context, userIDs []int64) (map[int64]*UserInfo, error)
	GetUserInfo(ctx context.Context, userId int64) (*UserInfo, error)
}

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

func (uc *FriendUsecase) SearchUsers(ctx context.Context, cmd *SearchUsersCommand) (*SearchUsersResult, error) {
	// 分页参数验证
	if cmd.PageStats.Page < 1 {
		cmd.PageStats.Page = 1
	}
	if cmd.PageStats.PageSize <= 0 {
		cmd.PageStats.PageSize = 20
	}
	if cmd.PageStats.PageSize > 100 {
		cmd.PageStats.PageSize = 100
	}

	offset := (cmd.PageStats.Page - 1) * cmd.PageStats.PageSize

	// 调用用户服务搜索用户（通过RPC）
	users, total, err := uc.repo.SearchUsers(ctx, cmd.Keyword, offset, cmd.PageStats.PageSize)
	if err != nil {
		uc.log.Errorf("搜索用户失败: %v", err)
		return nil, fmt.Errorf("搜索用户失败: %v", err)
	}

	// 获取在线状态（从本地数据库）
	userIDs := make([]int64, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	onlineStatus, err := uc.repo.BatchGetUserOnlineStatus(ctx, userIDs)
	if err == nil && onlineStatus != nil {
		for _, user := range users {
			if status, ok := onlineStatus[user.ID]; ok {
				user.OnlineStatus = status.Status
				user.LastOnlineTime = status.LastOnlineTime
			}
		}
	}

	return &SearchUsersResult{
		Users: users,
		Total: total,
	}, nil
}

func (uc *FriendUsecase) SendFriendApply(ctx context.Context, cmd *SendFriendApplyCommand) (*SendFriendApplyResult, error) {
	// 验证参数
	if cmd.ApplicantID == cmd.ReceiverID {
		return nil, fmt.Errorf("不能添加自己为好友")
	}

	// 检查是否已经是好友
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.ApplicantID, cmd.ReceiverID)
	if err != nil {
		uc.log.Errorf("检查好友关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if relation != nil && relation.Status == 1 { // 已经是好友
		return nil, fmt.Errorf("已经是好友")
	}

	// 检查是否有待处理的申请
	applies, err := uc.repo.ListFriendApplies(ctx, cmd.ReceiverID, nil, 0, 1)
	if err == nil && len(applies) > 0 {
		for _, apply := range applies {
			if apply.ApplicantID == cmd.ApplicantID && apply.Status == 0 {
				return nil, fmt.Errorf("已经发送过好友申请，请等待对方处理")
			}
		}
	}

	// 创建好友申请
	apply := &FriendApply{
		ApplicantID: cmd.ApplicantID,
		ReceiverID:  cmd.ReceiverID,
		ApplyReason: cmd.ApplyReason,
		Status:      0, // 待处理
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	apply.ID = int64(uuid.New().ID())

	err = uc.repo.CreateFriendApply(ctx, apply)
	if err != nil {
		uc.log.Errorf("创建好友申请失败: %v", err)
		return nil, fmt.Errorf("发送好友申请失败")
	}

	return &SendFriendApplyResult{
		ApplyID: apply.ID,
	}, nil
}

func (uc *FriendUsecase) HandleFriendApply(ctx context.Context, cmd *HandleFriendApplyCommand) (*HandleFriendApplyResult, error) {
	// 获取申请信息
	apply, err := uc.repo.GetFriendApply(ctx, cmd.ApplyID)
	if err != nil {
		uc.log.Errorf("查询好友申请失败: %v", err)
		return nil, fmt.Errorf("申请不存在")
	}

	if apply == nil {
		return nil, fmt.Errorf("申请不存在")
	}

	// 检查申请状态
	if apply.Status != 0 {
		return nil, fmt.Errorf("申请已处理")
	}

	// 检查处理人权限
	if apply.ReceiverID != cmd.HandlerID {
		return nil, fmt.Errorf("无权处理此申请")
	}

	now := time.Now()
	if cmd.Accept {
		// 同意申请，创建双向好友关系

		// 创建正向关系（用户 -> 好友）
		relation1 := &FriendRelation{
			UserID:    apply.ApplicantID,
			FriendID:  apply.ReceiverID,
			Status:    1, // 已同意
			CreatedAt: now,
			UpdatedAt: now,
		}
		relation1.ID = int64(uuid.New().ID())

		err = uc.repo.CreateFriendRelation(ctx, relation1)
		if err != nil {
			uc.log.Errorf("创建好友关系失败: %v", err)
			return nil, fmt.Errorf("处理申请失败")
		}

		// 创建反向关系（好友 -> 用户）
		relation2 := &FriendRelation{
			UserID:    apply.ReceiverID,
			FriendID:  apply.ApplicantID,
			Status:    1, // 已同意
			CreatedAt: now,
			UpdatedAt: now,
		}
		relation2.ID = int64(uuid.New().ID())

		err = uc.repo.CreateFriendRelation(ctx, relation2)
		if err != nil {
			uc.log.Errorf("创建好友关系失败: %v", err)
			// 回滚：删除正向关系
			_ = uc.repo.DeleteFriendRelation(ctx, relation1.ID)
			return nil, fmt.Errorf("处理申请失败")
		}
	}

	// 更新申请状态
	apply.Status = 1
	if !cmd.Accept {
		apply.Status = 2
	}
	apply.HandledAt = &now
	apply.UpdatedAt = now

	err = uc.repo.UpdateFriendApply(ctx, apply)
	if err != nil {
		uc.log.Errorf("更新申请状态失败: %v", err)
		return nil, fmt.Errorf("处理申请失败")
	}

	return &HandleFriendApplyResult{}, nil
}

func (uc *FriendUsecase) ListFriendApplies(ctx context.Context, query *ListFriendAppliesQuery) (*ListFriendAppliesResult, error) {
	// 分页参数验证
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}

	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize

	applies, err := uc.repo.ListFriendApplies(ctx, query.UserID, query.Status, offset, query.PageStats.PageSize)
	if err != nil {
		uc.log.Errorf("查询好友申请列表失败: %v", err)
		return nil, fmt.Errorf("查询申请列表失败")
	}

	total, err := uc.repo.CountFriendApplies(ctx, query.UserID, query.Status)
	if err != nil {
		uc.log.Errorf("统计好友申请数量失败: %v", err)
		return nil, fmt.Errorf("统计失败")
	}

	return &ListFriendAppliesResult{
		Applies: applies,
		Total:   total,
	}, nil
}

func (uc *FriendUsecase) ListFriends(ctx context.Context, query *ListFriendsQuery) (*ListFriendsResult, error) {
	// 分页参数验证
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}

	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize

	// 查询好友关系
	relations, err := uc.repo.ListFriends(ctx, query.UserID, offset, query.PageStats.PageSize, query.GroupName)
	if err != nil {
		uc.log.Errorf("查询好友列表失败: %v", err)
		return nil, fmt.Errorf("查询好友列表失败")
	}

	// 获取好友ID列表
	friendIDs := make([]int64, 0, len(relations))
	for _, relation := range relations {
		friendIDs = append(friendIDs, relation.FriendID)
	}

	// 批量获取好友信息
	friendInfos, err := uc.repo.BatchGetUserInfo(ctx, friendIDs)
	if err != nil {
		uc.log.Errorf("获取好友信息失败: %v", err)
		return nil, fmt.Errorf("获取好友信息失败")
	}

	// 获取好友在线状态
	onlineStatus, err := uc.repo.BatchGetUserOnlineStatus(ctx, friendIDs)
	if err != nil {
		uc.log.Errorf("获取好友在线状态失败: %v", err)
		// 不返回错误，继续处理
	}

	// 组装结果
	friends := make([]*FriendInfo, 0, len(relations))
	for _, relation := range relations {
		friendInfo := &FriendInfo{
			ID:        relation.ID,
			Remark:    relation.Remark,
			GroupName: relation.GroupName,
			Status:    relation.Status,
			CreatedAt: relation.CreatedAt,
		}

		// 填充好友信息
		if userInfo, ok := friendInfos[relation.FriendID]; ok {
			friendInfo.Friend = userInfo

			// 填充在线状态
			if status, ok := onlineStatus[relation.FriendID]; ok {
				friendInfo.Friend.OnlineStatus = status.Status
				friendInfo.Friend.LastOnlineTime = status.LastOnlineTime
			}
		}

		friends = append(friends, friendInfo)
	}

	total, err := uc.repo.CountFriends(ctx, query.UserID, query.GroupName)
	if err != nil {
		uc.log.Errorf("统计好友数量失败: %v", err)
		return nil, fmt.Errorf("统计失败")
	}

	return &ListFriendsResult{
		Friends: friends,
		Total:   total,
	}, nil
}

func (uc *FriendUsecase) DeleteFriend(ctx context.Context, cmd *DeleteFriendCommand) (*DeleteFriendResult, error) {
	// 获取好友关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		uc.log.Errorf("查询好友关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if relation == nil || relation.Status != 1 {
		return nil, fmt.Errorf("不是好友关系")
	}

	// 删除正向关系
	err = uc.repo.DeleteFriendRelation(ctx, relation.ID)
	if err != nil {
		uc.log.Errorf("删除好友关系失败: %v", err)
		return nil, fmt.Errorf("删除好友失败")
	}

	// 删除反向关系
	reverseRelation, err := uc.repo.GetFriendRelation(ctx, cmd.FriendID, cmd.UserID)
	if err == nil && reverseRelation != nil {
		_ = uc.repo.DeleteFriendRelation(ctx, reverseRelation.ID)
	}

	return &DeleteFriendResult{}, nil
}

func (uc *FriendUsecase) CheckFriendRelation(ctx context.Context, query *CheckFriendRelationQuery) (*CheckFriendRelationResult, error) {
	relation, err := uc.repo.GetFriendRelation(ctx, query.UserID, query.TargetID)
	if err != nil {
		uc.log.Errorf("检查好友关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	result := &CheckFriendRelationResult{
		IsFriend: false,
		Status:   0,
	}

	if relation != nil {
		result.IsFriend = relation.Status == 1
		result.Status = relation.Status
	}

	return result, nil
}

func (uc *FriendUsecase) UpdateUserOnlineStatus(ctx context.Context, userID int64, status int32, deviceType string) error {
	onlineStatus := &UserOnlineStatus{
		UserID:         userID,
		Status:         status,
		DeviceType:     deviceType,
		LastOnlineTime: time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := uc.repo.UpdateUserOnlineStatus(ctx, onlineStatus)
	if err != nil {
		uc.log.Errorf("更新用户在线状态失败: %v", err)
		return fmt.Errorf("更新状态失败")
	}

	return nil
}

func (uc *FriendUsecase) GetUserOnlineStatus(ctx context.Context, query *GetUserOnlineStatusQuery) (*GetUserOnlineStatusResult, error) {
	status, err := uc.repo.GetUserOnlineStatus(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("获取用户在线状态失败: %v", err)
		return nil, fmt.Errorf("获取状态失败")
	}

	if status == nil {
		return &GetUserOnlineStatusResult{
			Status:         0, // 默认离线
			LastOnlineTime: time.Now(),
		}, nil
	}

	return &GetUserOnlineStatusResult{
		Status:         status.Status,
		LastOnlineTime: status.LastOnlineTime,
	}, nil
}

func (uc *FriendUsecase) BatchGetUserOnlineStatus(ctx context.Context, query *BatchGetUserOnlineStatusQuery) (*BatchGetUserOnlineStatusResult, error) {
	statuses, err := uc.repo.BatchGetUserOnlineStatus(ctx, query.UserIDs)
	if err != nil {
		uc.log.Errorf("批量获取用户在线状态失败: %v", err)
		return nil, fmt.Errorf("获取状态失败")
	}

	result := make(map[int64]int32)
	for userID, status := range statuses {
		result[userID] = status.Status
	}

	return &BatchGetUserOnlineStatusResult{
		OnlineStatus: result,
	}, nil
}

// 在biz包中需要添加这个方法
func (uc *FriendUsecase) GetUserInfo(ctx context.Context, userID int64) (*UserInfo, error) {
	return uc.repo.GetUserInfo(ctx, userID)
}

func (uc *FriendUsecase) UpdateFriendRemark(ctx context.Context, cmd *UpdateFriendRemarkCommand) (*UpdateFriendRemarkResult, error) {
	// 这里需要实现更新好友备注的逻辑
	// 先获取好友关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		return nil, errors.New("系统错误")
	}

	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}

	// 更新备注
	relation.Remark = cmd.Remark
	relation.UpdatedAt = time.Now()

	err = uc.repo.UpdateFriendRelation(ctx, relation)
	if err != nil {
		return nil, errors.New("更新备注失败")
	}

	return &UpdateFriendRemarkResult{}, nil
}

func (uc *FriendUsecase) SetFriendGroup(ctx context.Context, cmd *SetFriendGroupCommand) (*SetFriendGroupResult, error) {
	// 这里需要实现设置好友分组的逻辑
	// 先获取好友关系
	relation, err := uc.repo.GetFriendRelation(ctx, cmd.UserID, cmd.FriendID)
	if err != nil {
		return nil, errors.New("系统错误")
	}

	if relation == nil || relation.Status != 1 {
		return nil, errors.New("不是好友关系")
	}

	// 更新分组
	relation.GroupName = cmd.GroupName
	relation.UpdatedAt = time.Now()

	err = uc.repo.UpdateFriendRelation(ctx, relation)
	if err != nil {
		return nil, &ServiceError{Code: 500, Message: "设置分组失败"}
	}

	return &SetFriendGroupResult{}, nil
}
