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

	// 查询用户加入的群聊
	ListJoinedGroups(ctx context.Context, userID int64, offset, limit int) ([]*Group, error)
	GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error)
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
		MemberCnt: 1, // 只有群主一人
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

	// todo 这里简化处理，直接加入（实际应该创建申请记录）
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
		return nil, fmt.Errorf("申请加入失败")
	}

	// 更新群成员数量
	group.MemberCnt++
	group.UpdatedAt = time.Now()
	err = uc.repo.UpdateGroup(ctx, group)
	if err != nil {
		uc.log.Errorf("更新群成员数量失败: %v", err)
		// 回滚：删除成员记录
		_ = uc.repo.DeleteGroupMember(ctx, member.ID)
		return nil, fmt.Errorf("申请加入失败")
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
