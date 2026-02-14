package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// Commands and Queries
type CreateGroupCommand struct {
	OwnerID int64
	Name    string
	Notice  string
	AddMode int32
	Avatar  string
}

type CreateGroupResult struct {
	GroupID int64
}

type LoadMyGroupQuery struct {
	OwnerID   int64
	PageStats PageStats
}

type LoadMyGroupResult struct {
	Groups []*Group
	Total  int64
}

type CheckGroupAddModeQuery struct {
	GroupID int64
}

type CheckGroupAddModeResult struct {
	AddMode int32
}

type EnterGroupDirectlyCommand struct {
	UserID  int64
	GroupID int64
}

type EnterGroupDirectlyResult struct{}

type ApplyJoinGroupCommand struct {
	UserID      int64
	GroupID     int64
	ApplyReason string
}

type ApplyJoinGroupResult struct{}

type LeaveGroupCommand struct {
	UserID  int64
	GroupID int64
}

type LeaveGroupResult struct{}

type DismissGroupCommand struct {
	OwnerID int64
	GroupID int64
}

type DismissGroupResult struct{}

type GetGroupInfoQuery struct {
	GroupID int64
}

type GetGroupInfoResult struct {
	Group *Group
}

type ListMyJoinedGroupsQuery struct {
	UserID    int64
	PageStats PageStats
}

type ListMyJoinedGroupsResult struct {
	Groups []*Group
	Total  int64
}

// 新增：获取群成员列表查询
type GetGroupMembersQuery struct {
	GroupID int64
}

type GetGroupMembersResult struct {
	MemberIDs []int64
}

// 新增：检查是否为群成员查询
type IsGroupMemberQuery struct {
	GroupID int64
	UserID  int64
}

type IsGroupMemberResult struct {
	IsMember bool
}

// 群聊信息领域对象

type Group struct {
	ID        int64
	Name      string
	Notice    string
	MemberCnt int
	OwnerID   int64
	AddMode   int32
	Avatar    string
	Status    int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 群成员领域对象
type GroupMember struct {
	ID       int64
	UserID   int64
	GroupID  int64
	Role     int32 // 0:普通成员, 1:管理员, 2:群主
	JoinTime time.Time
}

// 加群申请领域对象
type GroupApply struct {
	ID          int64
	UserID      int64
	GroupID     int64
	ApplyReason string
	Status      int32 // 0:待处理, 1:已通过, 2:已拒绝
	HandlerID   int64
	ReplyMsg    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type HandleGroupApplyCommand struct {
	ApplyID   int64
	HandlerID int64
	Accept    bool
	ReplyMsg  string
}

// 仓储接口
type GroupRepo interface {
	// 群聊基础操作
	CreateGroup(ctx context.Context, group *Group) error
	GetGroup(ctx context.Context, id int64) (*Group, error)
	UpdateGroup(ctx context.Context, group *Group) error
	DeleteGroup(ctx context.Context, id int64) error
	ListGroupsByOwner(ctx context.Context, ownerID int64, offset, limit int) ([]*Group, error)
	CountGroupsByOwner(ctx context.Context, ownerID int64) (int64, error)

	// 群成员操作
	CreateGroupMember(ctx context.Context, member *GroupMember) error
	GetGroupMember(ctx context.Context, groupID, userID int64) (*GroupMember, error)
	DeleteGroupMember(ctx context.Context, id int64) error
	ListGroupMembers(ctx context.Context, groupID int64, offset, limit int) ([]*GroupMember, error)
	CountGroupMembers(ctx context.Context, groupID int64) (int64, error)
	IsGroupMember(ctx context.Context, groupID, userID int64) (bool, error)
	IsGroupOwner(ctx context.Context, groupID, userID int64) (bool, error)
	BatchIsGroupMember(ctx context.Context, groupIDs []int64, userID int64) (map[int64]bool, error)
	GetGroupApply(ctx context.Context, id int64) (*GroupApply, error)
	CreateGroupApply(ctx context.Context, apply *GroupApply) error
	IsGroupAdmin(ctx context.Context, groupID, userID int64) (bool, error)
	UpdateGroupApply(ctx context.Context, apply *GroupApply) error

	// 查询用户加入的群聊
	ListJoinedGroups(ctx context.Context, userID int64, offset, limit int) ([]*Group, error)
	GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error)
	CountJoinedGroups(ctx context.Context, userID int64) (int64, error)
}

type GroupUsecase struct {
	repo             GroupRepo
	conversationRepo ConversationRepo // 新增
	log              *log.Helper
}

func NewGroupUsecase(repo GroupRepo, conversationRepo ConversationRepo, logger log.Logger) *GroupUsecase {
	return &GroupUsecase{
		repo:             repo,
		conversationRepo: conversationRepo,
		log:              log.NewHelper(logger),
	}
}

func (uc *GroupUsecase) CreateGroup(ctx context.Context, cmd *CreateGroupCommand) (*CreateGroupResult, error) {
	// 验证参数
	if cmd.Name == "" {
		return nil, fmt.Errorf("群聊名称不能为空")
	}
	if len(cmd.Name) > 20 {
		return nil, fmt.Errorf("群聊名称不能超过20个字符")
	}
	if cmd.AddMode != 0 && cmd.AddMode != 1 {
		return nil, fmt.Errorf("加群方式参数错误")
	}

	now := time.Now()

	// 创建群聊
	group := &Group{
		Name:      cmd.Name,
		Notice:    cmd.Notice,
		MemberCnt: 1, // 只有群主一人
		OwnerID:   cmd.OwnerID,
		AddMode:   cmd.AddMode,
		Avatar:    cmd.Avatar,
		Status:    0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	group.ID = int64(uuid.New().ID())

	// 创建群聊记录
	err := uc.repo.CreateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("创建群聊失败: %v", err)
		return nil, fmt.Errorf("创建群聊失败")
	}

	// 创建群主成员记录
	ownerMember := &GroupMember{
		UserID:   cmd.OwnerID,
		GroupID:  group.ID,
		Role:     2, // 群主
		JoinTime: now,
	}
	ownerMember.ID = int64(uuid.New().ID())

	err = uc.repo.CreateGroupMember(ctx, ownerMember)
	if err != nil {
		uc.log.Errorf("创建群主成员记录失败: %v", err)
		// 回滚：删除已创建的群聊
		_ = uc.repo.DeleteGroup(ctx, group.ID)
		return nil, fmt.Errorf("创建群聊失败")
	}

	// 创建群聊会话
	conv, err := uc.conversationRepo.GetOrCreateGroupConversation(ctx, group.ID)
	if err != nil {
		uc.log.Errorf("创建群聊会话失败: %v", err)
		// 回滚：删除群聊和成员
		_ = uc.repo.DeleteGroup(ctx, group.ID)
		_ = uc.repo.DeleteGroupMember(ctx, ownerMember.ID)
		return nil, fmt.Errorf("创建群聊失败")
	}

	// 将群主添加到会话成员
	err = uc.conversationRepo.AddConversationMember(ctx, &ConversationMember{
		ConversationID: conv.ID,
		UserID:         cmd.OwnerID,
		Type:           2, // 群主
		JoinTime:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		uc.log.Errorf("添加群主到会话失败: %v", err)
		// 同样回滚
		_ = uc.repo.DeleteGroup(ctx, group.ID)
		_ = uc.repo.DeleteGroupMember(ctx, ownerMember.ID)
		return nil, fmt.Errorf("创建群聊失败")
	}

	// 更新会话成员计数（可选）
	_ = uc.conversationRepo.UpdateConversationMemberCount(ctx, conv.ID, 1)

	return &CreateGroupResult{
		GroupID: group.ID,
	}, nil
}

func (uc *GroupUsecase) LoadMyGroup(ctx context.Context, query *LoadMyGroupQuery) (*LoadMyGroupResult, error) {
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

	groups, err := uc.repo.ListGroupsByOwner(ctx, query.OwnerID, offset, query.PageStats.PageSize)
	if err != nil {
		uc.log.Errorf("查询群聊列表失败: %v", err)
		return nil, fmt.Errorf("查询群聊列表失败")
	}

	total, err := uc.repo.CountGroupsByOwner(ctx, query.OwnerID)
	if err != nil {
		uc.log.Errorf("统计群聊数量失败: %v", err)
		return nil, fmt.Errorf("统计群聊数量失败")
	}

	return &LoadMyGroupResult{
		Groups: groups,
		Total:  total,
	}, nil
}

func (uc *GroupUsecase) CheckGroupAddMode(ctx context.Context, query *CheckGroupAddModeQuery) (*CheckGroupAddModeResult, error) {
	group, err := uc.repo.GetGroup(ctx, query.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	return &CheckGroupAddModeResult{
		AddMode: group.AddMode,
	}, nil
}

func (uc *GroupUsecase) EnterGroupDirectly(ctx context.Context, cmd *EnterGroupDirectlyCommand) (*EnterGroupDirectlyResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroup(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查群聊状态
	if group.Status != 0 {
		return nil, fmt.Errorf("群聊状态异常")
	}

	// 检查加群方式
	if group.AddMode != 0 {
		return nil, fmt.Errorf("该群聊需要申请才能加入")
	}

	// 检查是否已经是成员
	isMember, err := uc.repo.IsGroupMember(ctx, cmd.GroupID, cmd.UserID)
	if err != nil {
		uc.log.Errorf("检查群成员关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if isMember {
		// 已经是成员，直接返回成功（幂等性）
		return &EnterGroupDirectlyResult{}, nil
	}

	// 检查群成员数量
	if group.MemberCnt >= 500 {
		return nil, fmt.Errorf("群聊成员已满")
	}

	now := time.Now()

	// 创建成员记录
	member := &GroupMember{
		UserID:   cmd.UserID,
		GroupID:  cmd.GroupID,
		Role:     0, // 普通成员
		JoinTime: now,
	}
	member.ID = int64(uuid.New().ID())

	err = uc.repo.CreateGroupMember(ctx, member)
	if err != nil {
		uc.log.Errorf("加入群聊失败: %v", err)
		return nil, fmt.Errorf("加入群聊失败")
	}

	// 更新群成员数量
	group.MemberCnt++
	group.UpdatedAt = now
	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("更新群成员数量失败: %v", err)
		// 回滚：删除成员记录
		_ = uc.repo.DeleteGroupMember(ctx, member.ID)
		return nil, fmt.Errorf("加入群聊失败")
	}

	// 获取或创建群聊会话
	conv, err := uc.conversationRepo.GetOrCreateGroupConversation(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("获取群聊会话失败: %v", err)
		// 回滚：删除成员记录，并恢复群成员数量
		_ = uc.repo.DeleteGroupMember(ctx, member.ID)
		group.MemberCnt--
		group.UpdatedAt = now
		_ = uc.repo.UpdateGroup(ctx, group)
		return nil, fmt.Errorf("加入群聊失败")
	}

	// 将用户添加到会话成员
	err = uc.conversationRepo.AddConversationMember(ctx, &ConversationMember{
		ConversationID: conv.ID,
		UserID:         cmd.UserID,
		Type:           0, // 普通成员（注意：群主是2，这里是0）
		JoinTime:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		uc.log.Errorf("添加用户到会话失败: %v", err)
		// 回滚：删除成员记录，并恢复群成员数量
		_ = uc.repo.DeleteGroupMember(ctx, member.ID)
		group.MemberCnt--
		group.UpdatedAt = now
		_ = uc.repo.UpdateGroup(ctx, group)
		return nil, fmt.Errorf("加入群聊失败")
	}

	// 更新会话成员计数
	_ = uc.conversationRepo.UpdateConversationMemberCount(ctx, conv.ID, group.MemberCnt)

	return &EnterGroupDirectlyResult{}, nil
}

func (uc *GroupUsecase) ApplyJoinGroup(ctx context.Context, cmd *ApplyJoinGroupCommand) (*ApplyJoinGroupResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroup(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}
	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查群聊状态
	if group.Status != 0 {
		return nil, fmt.Errorf("群聊状态异常")
	}

	// 检查是否已经是成员
	isMember, err := uc.repo.IsGroupMember(ctx, cmd.GroupID, cmd.UserID)
	if err != nil {
		uc.log.Errorf("检查群成员关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}
	if isMember {
		return nil, fmt.Errorf("已经是群成员")
	}

	// 检查是否有待处理的申请
	// 此处需要添加查询 pending 申请的方法，这里简化，默认允许创建新申请

	// 创建申请记录
	apply := &GroupApply{
		UserID:      cmd.UserID,
		GroupID:     cmd.GroupID,
		ApplyReason: cmd.ApplyReason,
		Status:      0, // 待处理
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	apply.ID = int64(uuid.New().ID())

	if err := uc.repo.CreateGroupApply(ctx, apply); err != nil {
		uc.log.Errorf("创建加群申请失败: %v", err)
		return nil, fmt.Errorf("申请失败")
	}

	return &ApplyJoinGroupResult{}, nil
}

func (uc *GroupUsecase) LeaveGroup(ctx context.Context, cmd *LeaveGroupCommand) (*LeaveGroupResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroup(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查是否是群主
	isOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.UserID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if isOwner {
		return nil, fmt.Errorf("群主不能退群，请先转让群主或解散群聊")
	}

	// 检查是否是成员
	isMember, err := uc.repo.IsGroupMember(ctx, cmd.GroupID, cmd.UserID)
	if err != nil {
		uc.log.Errorf("检查群成员关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isMember {
		// 不是成员，直接返回成功（幂等性）
		return &LeaveGroupResult{}, nil
	}

	// 获取成员记录
	member, err := uc.repo.GetGroupMember(ctx, cmd.GroupID, cmd.UserID)
	if err != nil {
		uc.log.Errorf("获取群成员记录失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	// 删除成员记录
	err = uc.repo.DeleteGroupMember(ctx, member.ID)
	if err != nil {
		uc.log.Errorf("删除群成员记录失败: %v", err)
		return nil, fmt.Errorf("退群失败")
	}

	// 更新群成员数量
	group.MemberCnt--
	if group.MemberCnt < 0 {
		group.MemberCnt = 0
	}
	group.UpdatedAt = time.Now()
	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("更新群成员数量失败: %v", err)
		// 这里不进行回滚，因为成员已经删除
	}

	// 从群聊会话中移除成员
	conv, err := uc.conversationRepo.GetGroupConversation(ctx, cmd.GroupID)
	if err == nil && conv != nil {
		// 从会话中移除成员
		_ = uc.conversationRepo.RemoveConversationMember(ctx, conv.ID, cmd.UserID)
		// 更新会话成员计数（减一）
		_ = uc.conversationRepo.UpdateConversationMemberCount(ctx, conv.ID, -1)
	}

	return &LeaveGroupResult{}, nil
}

func (uc *GroupUsecase) DismissGroup(ctx context.Context, cmd *DismissGroupCommand) (*DismissGroupResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroup(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查是否是群主
	isOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.OwnerID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isOwner {
		return nil, fmt.Errorf("只有群主可以解散群聊")
	}

	// 更新群聊状态为解散
	group.Status = 2
	group.UpdatedAt = time.Now()

	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("解散群聊失败: %v", err)
		return nil, fmt.Errorf("解散群聊失败")
	}

	return &DismissGroupResult{}, nil
}

func (uc *GroupUsecase) ListMyJoinedGroups(ctx context.Context, query *ListMyJoinedGroupsQuery) (*ListMyJoinedGroupsResult, error) {
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

	groups, err := uc.repo.ListJoinedGroups(ctx, query.UserID, offset, query.PageStats.PageSize)
	if err != nil {
		uc.log.Errorf("查询加入的群聊列表失败: %v", err)
		return nil, fmt.Errorf("查询失败")
	}

	total, err := uc.repo.CountJoinedGroups(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("统计加入的群聊数量失败: %v", err)
		return nil, fmt.Errorf("统计失败")
	}

	return &ListMyJoinedGroupsResult{
		Groups: groups,
		Total:  total,
	}, nil
}

// 新增方法：获取群成员列表
func (uc *GroupUsecase) GetGroupMembers(ctx context.Context, query *GetGroupMembersQuery) (*GetGroupMembersResult, error) {
	memberIDs, err := uc.repo.GetGroupMembers(ctx, query.GroupID)
	if err != nil {
		uc.log.Errorf("获取群成员列表失败: %v", err)
		return nil, fmt.Errorf("获取群成员列表失败")
	}

	return &GetGroupMembersResult{
		MemberIDs: memberIDs,
	}, nil
}

// 新增方法：检查是否为群成员
func (uc *GroupUsecase) IsGroupMember(ctx context.Context, query *IsGroupMemberQuery) (*IsGroupMemberResult, error) {
	isMember, err := uc.repo.IsGroupMember(ctx, query.GroupID, query.UserID)
	if err != nil {
		uc.log.Errorf("检查群成员关系失败: %v", err)
		return nil, fmt.Errorf("检查群成员关系失败")
	}

	return &IsGroupMemberResult{
		IsMember: isMember,
	}, nil
}

func (uc *GroupUsecase) GetGroupInfo(ctx context.Context, query *GetGroupInfoQuery) (*GetGroupInfoResult, error) {
	group, err := uc.repo.GetGroup(ctx, query.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return &GetGroupInfoResult{Group: nil}, nil
	}

	return &GetGroupInfoResult{Group: group}, nil
}

func (uc *GroupUsecase) HandleGroupApply(ctx context.Context, cmd *HandleGroupApplyCommand) error {
	// 获取申请记录
	apply, err := uc.repo.GetGroupApply(ctx, cmd.ApplyID)
	if err != nil {
		return err
	}
	if apply == nil {
		return fmt.Errorf("申请记录不存在")
	}
	if apply.Status != 0 {
		return fmt.Errorf("申请已处理")
	}

	// 检查处理人权限（必须是群主或管理员）
	isOwner, err := uc.repo.IsGroupOwner(ctx, apply.GroupID, cmd.HandlerID)
	if err != nil {
		return err
	}
	isAdmin, err := uc.repo.IsGroupAdmin(ctx, apply.GroupID, cmd.HandlerID)
	if err != nil {
		return err
	}
	if !isOwner && !isAdmin {
		return fmt.Errorf("无权处理申请")
	}

	now := time.Now()
	apply.HandlerID = cmd.HandlerID
	apply.ReplyMsg = cmd.ReplyMsg
	apply.UpdatedAt = now

	if cmd.Accept {
		apply.Status = 1 // 已通过

		// 检查群成员数量限制
		group, err := uc.repo.GetGroup(ctx, apply.GroupID)
		if err != nil {
			return err
		}
		if group.MemberCnt >= 500 {
			return fmt.Errorf("群聊成员已满")
		}

		// 创建成员记录
		member := &GroupMember{
			UserID:   apply.UserID,
			GroupID:  apply.GroupID,
			Role:     0,
			JoinTime: now,
		}
		member.ID = int64(uuid.New().ID())

		if err := uc.repo.CreateGroupMember(ctx, member); err != nil {
			return err
		}

		// 更新群成员数量
		group.MemberCnt++
		group.UpdatedAt = now
		if err := uc.repo.UpdateGroup(ctx, group); err != nil {
			// 回滚成员记录
			_ = uc.repo.DeleteGroupMember(ctx, member.ID)
			return err
		}

		// 同步到会话成员
		conv, err := uc.conversationRepo.GetOrCreateGroupConversation(ctx, apply.GroupID)
		if err == nil {
			_ = uc.conversationRepo.AddConversationMember(ctx, &ConversationMember{
				ConversationID: conv.ID,
				UserID:         apply.UserID,
				Type:           0,
				JoinTime:       now,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
			// 更新会话成员计数
			_ = uc.conversationRepo.UpdateConversationMemberCount(ctx, conv.ID, 1)
		}
	} else {
		apply.Status = 2 // 已拒绝
	}

	// 更新申请记录
	return uc.repo.UpdateGroupApply(ctx, apply)
}
