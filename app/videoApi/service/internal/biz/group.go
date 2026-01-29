package biz

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type Group struct {
	ID        int64
	Name      string
	Notice    string
	Members   []int64
	MemberCnt int
	OwnerID   int64
	AddMode   int32
	Avatar    string
	Status    int32
	CreatedAt string
	UpdatedAt string
}

type CreateGroupInput struct {
	Name    string
	Notice  string
	AddMode int32
	Avatar  string
}

type LoadMyGroupInput struct {
	PageStats *PageStats
}

type LoadMyGroupOutput struct {
	Groups []*Group
	Total  int64
}

type CheckGroupAddModeInput struct {
	GroupID int64
}

type CheckGroupAddModeOutput struct {
	AddMode int32
}

type EnterGroupDirectlyInput struct {
	GroupID int64
}

type ApplyJoinGroupInput struct {
	GroupID     int64
	ApplyReason string
}

type LeaveGroupInput struct {
	GroupID int64
}

type DismissGroupInput struct {
	GroupID int64
}

type GetGroupInfoInput struct {
	GroupID int64
}

type GetGroupInfoOutput struct {
	Group *Group
}

type ListMyJoinedGroupsInput struct {
	PageStats *PageStats
}

type ListMyJoinedGroupsOutput struct {
	Groups []*Group
	Total  int64
}

type GroupUsecase struct {
	chat ChatAdapter
	log  *log.Helper
}

func NewGroupUsecase(chat ChatAdapter, logger log.Logger) *GroupUsecase {
	return &GroupUsecase{
		chat: chat,
		log:  log.NewHelper(logger),
	}
}

func (uc *GroupUsecase) CreateGroup(ctx context.Context, input *CreateGroupInput) (int64, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return 0, errors.New("获取用户信息失败")
	}

	// 参数验证
	if input.Name == "" {
		return 0, errors.New("群聊名称不能为空")
	}
	if len(input.Name) > 20 {
		return 0, errors.New("群聊名称不能超过20个字符")
	}
	if input.AddMode != 0 && input.AddMode != 1 {
		return 0, errors.New("加群方式参数错误")
	}

	groupID, err := uc.chat.CreateGroup(ctx, userId, input.Name, input.Notice, input.AddMode, input.Avatar)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("创建群聊失败: %v", err)
		return 0, errors.New("创建群聊失败")
	}

	return groupID, nil
}

func (uc *GroupUsecase) LoadMyGroup(ctx context.Context, input *LoadMyGroupInput) (*LoadMyGroupOutput, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	total, groups, err := uc.chat.LoadMyGroup(ctx, userId, input.PageStats)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取我创建的群聊失败: %v", err)
		return nil, errors.New("获取群聊列表失败")
	}

	return &LoadMyGroupOutput{
		Groups: groups,
		Total:  total,
	}, nil
}

func (uc *GroupUsecase) CheckGroupAddMode(ctx context.Context, input *CheckGroupAddModeInput) (*CheckGroupAddModeOutput, error) {
	addMode, err := uc.chat.CheckGroupAddMode(ctx, input.GroupID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("检查群聊加群方式失败: %v", err)
		return nil, errors.New("检查加群方式失败")
	}

	return &CheckGroupAddModeOutput{
		AddMode: addMode,
	}, nil
}

func (uc *GroupUsecase) EnterGroupDirectly(ctx context.Context, input *EnterGroupDirectlyInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.EnterGroupDirectly(ctx, userId, input.GroupID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("直接进群失败: %v", err)
		return errors.New("加入群聊失败")
	}

	return nil
}

func (uc *GroupUsecase) ApplyJoinGroup(ctx context.Context, input *ApplyJoinGroupInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.ApplyJoinGroup(ctx, userId, input.GroupID, input.ApplyReason)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("申请加群失败: %v", err)
		return errors.New("申请加入失败")
	}

	return nil
}

func (uc *GroupUsecase) LeaveGroup(ctx context.Context, input *LeaveGroupInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.LeaveGroup(ctx, userId, input.GroupID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("退群失败: %v", err)
		return errors.New("退群失败")
	}

	return nil
}

func (uc *GroupUsecase) DismissGroup(ctx context.Context, input *DismissGroupInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.DismissGroup(ctx, userId, input.GroupID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("解散群聊失败: %v", err)
		return errors.New("解散群聊失败")
	}

	return nil
}

func (uc *GroupUsecase) GetGroupInfo(ctx context.Context, input *GetGroupInfoInput) (*GetGroupInfoOutput, error) {
	group, err := uc.chat.GetGroupInfo(ctx, input.GroupID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取群聊信息失败: %v", err)
		return nil, errors.New("获取群聊信息失败")
	}

	if group == nil {
		return nil, errors.New("群聊不存在")
	}

	return &GetGroupInfoOutput{
		Group: group,
	}, nil
}

func (uc *GroupUsecase) ListMyJoinedGroups(ctx context.Context, input *ListMyJoinedGroupsInput) (*ListMyJoinedGroupsOutput, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	total, groups, err := uc.chat.ListMyJoinedGroups(ctx, userId, input.PageStats)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取我加入的群聊失败: %v", err)
		return nil, errors.New("获取群聊列表失败")
	}

	return &ListMyJoinedGroupsOutput{
		Groups: groups,
		Total:  total,
	}, nil
}
