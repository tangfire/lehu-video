package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// 群聊信息领域对象
type Group struct {
	ID        int64
	Name      string
	Notice    string
	Members   []int64 // 成员ID列表
	MemberCnt int
	OwnerID   int64
	AddMode   int32 // 0:直接加入, 1:需要审核
	Avatar    string
	Status    int32 // 0:正常, 1:禁用, 2:解散
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

type HandleJoinApplyCommand struct {
	ApplyID   int64
	HandlerID int64
	Accept    bool
	ReplyMsg  string
}

type HandleJoinApplyResult struct{}

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

type UpdateGroupInfoCommand struct {
	GroupID    int64
	OperatorID int64
	Name       string
	Notice     string
	AddMode    int32
	Avatar     string
}

type UpdateGroupInfoResult struct{}

type ListGroupMembersQuery struct {
	GroupID   int64
	PageStats PageStats
}

type ListGroupMembersResult struct {
	Members []*GroupMember
	Total   int64
}

type RemoveGroupMemberCommand struct {
	GroupID      int64
	OperatorID   int64
	TargetUserID int64
}

type RemoveGroupMemberResult struct{}

type TransferGroupOwnerCommand struct {
	GroupID    int64
	FromUserID int64
	ToUserID   int64
}

type TransferGroupOwnerResult struct{}

type SetGroupAdminCommand struct {
	GroupID      int64
	OperatorID   int64
	TargetUserID int64
	SetAsAdmin   bool
}

type SetGroupAdminResult struct{}

type ListMyJoinedGroupsQuery struct {
	UserID    int64
	PageStats PageStats
}

type ListMyJoinedGroupsResult struct {
	Groups []*Group
	Total  int64
}

// 仓储接口
type GroupRepo interface {
	// 群聊基础操作
	CreateGroup(ctx context.Context, group *Group) error
	GetGroupByID(ctx context.Context, id int64) (*Group, error)
	GetGroupByOwnerAndID(ctx context.Context, ownerID, id int64) (*Group, error)
	UpdateGroup(ctx context.Context, group *Group) error
	DeleteGroup(ctx context.Context, id int64) error
	ListGroupsByOwner(ctx context.Context, ownerID int64, offset, limit int) ([]*Group, error)
	CountGroupsByOwner(ctx context.Context, ownerID int64) (int64, error)

	// 群成员操作
	CreateGroupMember(ctx context.Context, member *GroupMember) error
	GetGroupMember(ctx context.Context, groupID, userID int64) (*GroupMember, error)
	UpdateGroupMember(ctx context.Context, member *GroupMember) error
	DeleteGroupMember(ctx context.Context, id int64) error
	ListGroupMembers(ctx context.Context, groupID int64, offset, limit int) ([]*GroupMember, error)
	CountGroupMembers(ctx context.Context, groupID int64) (int64, error)
	IsGroupMember(ctx context.Context, groupID, userID int64) (bool, error)
	IsGroupOwner(ctx context.Context, groupID, userID int64) (bool, error)
	IsGroupAdmin(ctx context.Context, groupID, userID int64) (bool, error)

	// 加群申请
	CreateGroupApply(ctx context.Context, apply *GroupApply) error
	GetGroupApply(ctx context.Context, id int64) (*GroupApply, error)
	UpdateGroupApply(ctx context.Context, apply *GroupApply) error
	ListPendingApplies(ctx context.Context, groupID int64, offset, limit int) ([]*GroupApply, error)
	CountPendingApplies(ctx context.Context, groupID int64) (int64, error)

	// 查询用户加入的群聊
	ListJoinedGroups(ctx context.Context, userID int64, offset, limit int) ([]*Group, error)
	CountJoinedGroups(ctx context.Context, userID int64) (int64, error)
}

type GroupUsecase struct {
	repo GroupRepo
	log  *log.Helper
}

func NewGroupUsecase(repo GroupRepo, logger log.Logger) *GroupUsecase {
	return &GroupUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
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

	// 创建群聊
	group := &Group{
		Name:      cmd.Name,
		Notice:    cmd.Notice,
		Members:   []int64{cmd.OwnerID},
		MemberCnt: 1,
		OwnerID:   cmd.OwnerID,
		AddMode:   cmd.AddMode,
		Avatar:    cmd.Avatar,
		Status:    0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	group.ID = int64(uuid.New().ID())

	// 创建群聊
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
		JoinTime: time.Now(),
	}
	ownerMember.ID = int64(uuid.New().ID())

	err = uc.repo.CreateGroupMember(ctx, ownerMember)
	if err != nil {
		uc.log.Errorf("创建群主成员记录失败: %v", err)
		// 回滚：删除已创建的群聊
		_ = uc.repo.DeleteGroup(ctx, group.ID)
		return nil, fmt.Errorf("创建群聊失败")
	}

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

	groups, err := uc.repo.ListGroupsByOwner(ctx, query.OwnerID, int(offset), int(query.PageStats.PageSize))
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
	group, err := uc.repo.GetGroupByID(ctx, query.GroupID)
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
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
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

	// 创建成员记录
	member := &GroupMember{
		UserID:   cmd.UserID,
		GroupID:  cmd.GroupID,
		Role:     0, // 普通成员
		JoinTime: time.Now(),
	}
	member.ID = int64(uuid.New().ID())

	err = uc.repo.CreateGroupMember(ctx, member)
	if err != nil {
		uc.log.Errorf("加入群聊失败: %v", err)
		return nil, fmt.Errorf("加入群聊失败")
	}

	// 更新群成员数量
	group.MemberCnt++
	group.UpdatedAt = time.Now()
	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("更新群成员数量失败: %v", err)
		// 回滚：删除成员记录
		_ = uc.repo.DeleteGroupMember(ctx, member.ID)
		return nil, fmt.Errorf("加入群聊失败")
	}

	return &EnterGroupDirectlyResult{}, nil
}

func (uc *GroupUsecase) ApplyJoinGroup(ctx context.Context, cmd *ApplyJoinGroupCommand) (*ApplyJoinGroupResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
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
	if group.AddMode != 1 {
		return nil, fmt.Errorf("该群聊可以直接加入")
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
	// TODO: 可以添加检查是否有待处理申请的逻辑

	// 创建加群申请
	apply := &GroupApply{
		UserID:      cmd.UserID,
		GroupID:     cmd.GroupID,
		ApplyReason: cmd.ApplyReason,
		Status:      0, // 待处理
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	apply.ID = int64(uuid.New().ID())

	err = uc.repo.CreateGroupApply(ctx, apply)
	if err != nil {
		uc.log.Errorf("创建加群申请失败: %v", err)
		return nil, fmt.Errorf("申请加入失败")
	}

	return &ApplyJoinGroupResult{}, nil
}

func (uc *GroupUsecase) HandleJoinApply(ctx context.Context, cmd *HandleJoinApplyCommand) (*HandleJoinApplyResult, error) {
	// 获取申请信息
	apply, err := uc.repo.GetGroupApply(ctx, cmd.ApplyID)
	if err != nil {
		uc.log.Errorf("查询加群申请失败: %v", err)
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
	isOwner, err := uc.repo.IsGroupOwner(ctx, apply.GroupID, cmd.HandlerID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	isAdmin, err := uc.repo.IsGroupAdmin(ctx, apply.GroupID, cmd.HandlerID)
	if err != nil {
		uc.log.Errorf("检查管理员权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isOwner && !isAdmin {
		return nil, fmt.Errorf("无权处理加群申请")
	}

	if cmd.Accept {
		// 同意加群
		// 检查是否已经是成员
		isMember, err := uc.repo.IsGroupMember(ctx, apply.GroupID, apply.UserID)
		if err != nil {
			uc.log.Errorf("检查群成员关系失败: %v", err)
			return nil, fmt.Errorf("系统错误")
		}

		if isMember {
			// 已经是成员，更新申请状态即可
			apply.Status = 1
			apply.HandlerID = cmd.HandlerID
			apply.ReplyMsg = cmd.ReplyMsg
			apply.UpdatedAt = time.Now()

			err = uc.repo.UpdateGroupApply(ctx, apply)
			if err != nil {
				uc.log.Errorf("更新申请状态失败: %v", err)
				return nil, fmt.Errorf("处理申请失败")
			}

			return &HandleJoinApplyResult{}, nil
		}

		// 获取群聊信息
		group, err := uc.repo.GetGroupByID(ctx, apply.GroupID)
		if err != nil {
			uc.log.Errorf("查询群聊信息失败: %v", err)
			return nil, fmt.Errorf("群聊不存在")
		}

		// 检查群成员数量
		if group.MemberCnt >= 500 {
			return nil, fmt.Errorf("群聊成员已满")
		}

		// 创建成员记录
		member := &GroupMember{
			UserID:   apply.UserID,
			GroupID:  apply.GroupID,
			Role:     0, // 普通成员
			JoinTime: time.Now(),
		}
		member.ID = int64(uuid.New().ID())

		err = uc.repo.CreateGroupMember(ctx, member)
		if err != nil {
			uc.log.Errorf("添加群成员失败: %v", err)
			return nil, fmt.Errorf("处理申请失败")
		}

		// 更新群成员数量
		group.MemberCnt++
		group.UpdatedAt = time.Now()
		err = uc.repo.UpdateGroup(ctx, group)
		if err != nil {
			uc.log.Errorf("更新群成员数量失败: %v", err)
			// 回滚：删除成员记录
			_ = uc.repo.DeleteGroupMember(ctx, member.ID)
			return nil, fmt.Errorf("处理申请失败")
		}
	}

	// 更新申请状态
	apply.Status = 1
	if !cmd.Accept {
		apply.Status = 2
	}
	apply.HandlerID = cmd.HandlerID
	apply.ReplyMsg = cmd.ReplyMsg
	apply.UpdatedAt = time.Now()

	err = uc.repo.UpdateGroupApply(ctx, apply)
	if err != nil {
		uc.log.Errorf("更新申请状态失败: %v", err)
		return nil, fmt.Errorf("处理申请失败")
	}

	return &HandleJoinApplyResult{}, nil
}

func (uc *GroupUsecase) LeaveGroup(ctx context.Context, cmd *LeaveGroupCommand) (*LeaveGroupResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
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

	return &LeaveGroupResult{}, nil
}

func (uc *GroupUsecase) DismissGroup(ctx context.Context, cmd *DismissGroupCommand) (*DismissGroupResult, error) {
	// 检查群聊是否存在且属于该用户
	group, err := uc.repo.GetGroupByOwnerAndID(ctx, cmd.OwnerID, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在或无权操作")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在或无权操作")
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

func (uc *GroupUsecase) GetGroupInfo(ctx context.Context, query *GetGroupInfoQuery) (*GetGroupInfoResult, error) {
	group, err := uc.repo.GetGroupByID(ctx, query.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return &GetGroupInfoResult{Group: nil}, nil
	}

	return &GetGroupInfoResult{Group: group}, nil
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

	groups, err := uc.repo.ListJoinedGroups(ctx, query.UserID, int(offset), int(query.PageStats.PageSize))
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

func (uc *GroupUsecase) UpdateGroupInfo(ctx context.Context, cmd *UpdateGroupInfoCommand) (*UpdateGroupInfoResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查操作者权限（只有群主和管理员可以修改）
	isOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.OperatorID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	isAdmin, err := uc.repo.IsGroupAdmin(ctx, cmd.GroupID, cmd.OperatorID)
	if err != nil {
		uc.log.Errorf("检查管理员权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isOwner && !isAdmin {
		return nil, fmt.Errorf("无权修改群聊信息")
	}

	// 如果是管理员修改，限制权限（只能修改部分信息）
	if !isOwner {
		// 管理员不能修改群名称、群主ID、加群方式等敏感信息
		if cmd.Name != "" && cmd.Name != group.Name {
			return nil, fmt.Errorf("管理员不能修改群名称")
		}
		if cmd.AddMode != 0 && cmd.AddMode != group.AddMode {
			return nil, fmt.Errorf("管理员不能修改加群方式")
		}
	}

	// 更新群聊信息
	if cmd.Name != "" {
		group.Name = cmd.Name
	}
	if cmd.Notice != "" {
		group.Notice = cmd.Notice
	}
	if cmd.AddMode != 0 {
		group.AddMode = cmd.AddMode
	}
	if cmd.Avatar != "" {
		group.Avatar = cmd.Avatar
	}
	group.UpdatedAt = time.Now()

	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("更新群聊信息失败: %v", err)
		return nil, fmt.Errorf("更新群聊信息失败")
	}

	return &UpdateGroupInfoResult{}, nil
}

func (uc *GroupUsecase) ListGroupMembers(ctx context.Context, query *ListGroupMembersQuery) (*ListGroupMembersResult, error) {
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

	members, err := uc.repo.ListGroupMembers(ctx, query.GroupID, int(offset), int(query.PageStats.PageSize))
	if err != nil {
		uc.log.Errorf("查询群成员列表失败: %v", err)
		return nil, fmt.Errorf("查询群成员列表失败")
	}

	total, err := uc.repo.CountGroupMembers(ctx, query.GroupID)
	if err != nil {
		uc.log.Errorf("统计群成员数量失败: %v", err)
		return nil, fmt.Errorf("统计群成员数量失败")
	}

	return &ListGroupMembersResult{
		Members: members,
		Total:   total,
	}, nil
}

func (uc *GroupUsecase) RemoveGroupMember(ctx context.Context, cmd *RemoveGroupMemberCommand) (*RemoveGroupMemberResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查操作者权限（只有群主和管理员可以移除成员）
	isOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.OperatorID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	isAdmin, err := uc.repo.IsGroupAdmin(ctx, cmd.GroupID, cmd.OperatorID)
	if err != nil {
		uc.log.Errorf("检查管理员权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isOwner && !isAdmin {
		return nil, fmt.Errorf("无权移除群成员")
	}

	// 不能移除自己（群主除外）
	if cmd.OperatorID == cmd.TargetUserID && !isOwner {
		return nil, fmt.Errorf("不能移除自己")
	}

	// 检查目标成员是否是群主
	isTargetOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.TargetUserID)
	if err != nil {
		uc.log.Errorf("检查目标用户权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if isTargetOwner {
		return nil, fmt.Errorf("不能移除群主")
	}

	// 如果是管理员移除成员，不能移除其他管理员（只有群主可以）
	if isAdmin && !isOwner {
		isTargetAdmin, err := uc.repo.IsGroupAdmin(ctx, cmd.GroupID, cmd.TargetUserID)
		if err != nil {
			uc.log.Errorf("检查目标用户权限失败: %v", err)
			return nil, fmt.Errorf("系统错误")
		}
		if isTargetAdmin {
			return nil, fmt.Errorf("管理员不能移除其他管理员")
		}
	}

	// 获取成员记录
	member, err := uc.repo.GetGroupMember(ctx, cmd.GroupID, cmd.TargetUserID)
	if err != nil {
		uc.log.Errorf("获取群成员记录失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if member == nil {
		// 不是成员，直接返回成功（幂等性）
		return &RemoveGroupMemberResult{}, nil
	}

	// 删除成员记录
	err = uc.repo.DeleteGroupMember(ctx, member.ID)
	if err != nil {
		uc.log.Errorf("删除群成员记录失败: %v", err)
		return nil, fmt.Errorf("移除成员失败")
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

	return &RemoveGroupMemberResult{}, nil
}

func (uc *GroupUsecase) TransferGroupOwner(ctx context.Context, cmd *TransferGroupOwnerCommand) (*TransferGroupOwnerResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查转让人是否是群主
	isOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.FromUserID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isOwner {
		return nil, fmt.Errorf("只有群主可以转让群主")
	}

	// 检查接收人是否是群成员
	isMember, err := uc.repo.IsGroupMember(ctx, cmd.GroupID, cmd.ToUserID)
	if err != nil {
		uc.log.Errorf("检查群成员关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isMember {
		return nil, fmt.Errorf("接收人不是群成员")
	}

	// 获取转让人成员记录
	fromMember, err := uc.repo.GetGroupMember(ctx, cmd.GroupID, cmd.FromUserID)
	if err != nil {
		uc.log.Errorf("获取转让人成员记录失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	// 获取接收人成员记录
	toMember, err := uc.repo.GetGroupMember(ctx, cmd.GroupID, cmd.ToUserID)
	if err != nil {
		uc.log.Errorf("获取接收人成员记录失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	// 更新转让人角色为普通成员
	fromMember.Role = 0
	err = uc.repo.UpdateGroupMember(ctx, fromMember)
	if err != nil {
		uc.log.Errorf("更新转让人角色失败: %v", err)
		return nil, fmt.Errorf("转让群主失败")
	}

	// 更新接收人角色为群主
	toMember.Role = 2
	err = uc.repo.UpdateGroupMember(ctx, toMember)
	if err != nil {
		uc.log.Errorf("更新接收人角色失败: %v", err)
		// 回滚：恢复转让人角色
		fromMember.Role = 2
		_ = uc.repo.UpdateGroupMember(ctx, fromMember)
		return nil, fmt.Errorf("转让群主失败")
	}

	// 更新群聊的群主ID
	group.OwnerID = cmd.ToUserID
	group.UpdatedAt = time.Now()
	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("更新群聊群主失败: %v", err)
		// 回滚：恢复成员角色
		fromMember.Role = 2
		_ = uc.repo.UpdateGroupMember(ctx, fromMember)
		toMember.Role = 0
		_ = uc.repo.UpdateGroupMember(ctx, toMember)
		return nil, fmt.Errorf("转让群主失败")
	}

	return &TransferGroupOwnerResult{}, nil
}

func (uc *GroupUsecase) SetGroupAdmin(ctx context.Context, cmd *SetGroupAdminCommand) (*SetGroupAdminResult, error) {
	// 检查群聊是否存在
	group, err := uc.repo.GetGroupByID(ctx, cmd.GroupID)
	if err != nil {
		uc.log.Errorf("查询群聊信息失败: %v", err)
		return nil, fmt.Errorf("群聊不存在")
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 检查操作者是否是群主
	isOwner, err := uc.repo.IsGroupOwner(ctx, cmd.GroupID, cmd.OperatorID)
	if err != nil {
		uc.log.Errorf("检查群主权限失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isOwner {
		return nil, fmt.Errorf("只有群主可以设置管理员")
	}

	// 检查目标用户是否是群成员
	isMember, err := uc.repo.IsGroupMember(ctx, cmd.GroupID, cmd.TargetUserID)
	if err != nil {
		uc.log.Errorf("检查群成员关系失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	if !isMember {
		return nil, fmt.Errorf("目标用户不是群成员")
	}

	// 不能设置自己（群主已经是最高权限）
	if cmd.OperatorID == cmd.TargetUserID {
		return nil, fmt.Errorf("不能设置自己")
	}

	// 获取目标成员记录
	member, err := uc.repo.GetGroupMember(ctx, cmd.GroupID, cmd.TargetUserID)
	if err != nil {
		uc.log.Errorf("获取群成员记录失败: %v", err)
		return nil, fmt.Errorf("系统错误")
	}

	// 更新成员角色
	if cmd.SetAsAdmin {
		member.Role = 1 // 管理员
	} else {
		member.Role = 0 // 普通成员
	}

	err = uc.repo.UpdateGroupMember(ctx, member)
	if err != nil {
		uc.log.Errorf("更新成员角色失败: %v", err)
		return nil, fmt.Errorf("设置管理员失败")
	}

	return &SetGroupAdminResult{}, nil
}
