package data

import (
	"context"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

// 群聊相关方法
func (r *chatAdapterImpl) CreateGroup(ctx context.Context, ownerID, name, notice string, addMode int32, avatar string) (string, error) {
	req := &chat.CreateGroupReq{
		OwnerId: ownerID,
		Name:    name,
		Notice:  notice,
		AddMode: addMode,
		Avatar:  avatar,
	}

	resp, err := r.group.CreateGroup(ctx, req)
	if err != nil {
		return "0", err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", err
	}

	return resp.GroupId, nil
}

func (r *chatAdapterImpl) LoadMyGroup(ctx context.Context, ownerID string, pageStats *biz.PageStats) (int64, []*biz.Group, error) {
	req := &chat.LoadMyGroupReq{
		OwnerId: ownerID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	}

	resp, err := r.group.LoadMyGroup(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var groups []*biz.Group
	for _, g := range resp.Groups {
		groups = append(groups, &biz.Group{
			ID:        g.Id,
			Name:      g.Name,
			Notice:    g.Notice,
			MemberCnt: int(g.MemberCnt),
			OwnerID:   g.OwnerId,
			AddMode:   g.AddMode,
			Avatar:    g.Avatar,
			Status:    g.Status,
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
		})
	}

	return int64(resp.PageStats.Total), groups, nil
}

func (r *chatAdapterImpl) CheckGroupAddMode(ctx context.Context, groupID string) (int32, error) {
	req := &chat.CheckGroupAddModeReq{
		GroupId: groupID,
	}

	resp, err := r.group.CheckGroupAddMode(ctx, req)
	if err != nil {
		return 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}

	return resp.AddMode, nil
}

func (r *chatAdapterImpl) EnterGroupDirectly(ctx context.Context, userID, groupID string) error {
	req := &chat.EnterGroupDirectlyReq{
		UserId:  userID,
		GroupId: groupID,
	}

	resp, err := r.group.EnterGroupDirectly(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) ApplyJoinGroup(ctx context.Context, userID, groupID, applyReason string) error {
	req := &chat.ApplyJoinGroupReq{
		UserId:      userID,
		GroupId:     groupID,
		ApplyReason: applyReason,
	}

	resp, err := r.group.ApplyJoinGroup(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) GetGroupInfo(ctx context.Context, groupID string) (*biz.Group, error) {
	req := &chat.GetGroupInfoReq{
		GroupId: groupID,
	}

	resp, err := r.group.GetGroupInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	group := resp.Group
	return &biz.Group{
		ID:        group.Id,
		Name:      group.Name,
		Notice:    group.Notice,
		MemberCnt: int(group.MemberCnt),
		OwnerID:   group.OwnerId,
		AddMode:   group.AddMode,
		Avatar:    group.Avatar,
		Status:    group.Status,
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}, nil
}

func (r *chatAdapterImpl) ListMyJoinedGroups(ctx context.Context, userID string, pageStats *biz.PageStats) (int64, []*biz.Group, error) {
	req := &chat.ListMyJoinedGroupsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	}

	resp, err := r.group.ListMyJoinedGroups(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var groups []*biz.Group
	for _, g := range resp.Groups {
		groups = append(groups, &biz.Group{
			ID:        g.Id,
			Name:      g.Name,
			Notice:    g.Notice,
			MemberCnt: int(g.MemberCnt),
			OwnerID:   g.OwnerId,
			AddMode:   g.AddMode,
			Avatar:    g.Avatar,
			Status:    g.Status,
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
		})
	}

	return int64(resp.PageStats.Total), groups, nil
}

func (r *chatAdapterImpl) LeaveGroup(ctx context.Context, userID, groupID string) error {
	req := &chat.LeaveGroupReq{
		UserId:  userID,
		GroupId: groupID,
	}

	resp, err := r.group.LeaveGroup(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) DismissGroup(ctx context.Context, ownerID string, groupID string) error {
	req := &chat.DismissGroupReq{
		OwnerId: ownerID,
		GroupId: groupID,
	}

	resp, err := r.group.DismissGroup(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

// 在 data/chatadapter.go 中添加缺少的方法
// 获取群成员列表
func (r *chatAdapterImpl) GetGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	// 调用群聊服务的获取成员方法
	// 注意：需要先在 chat.proto 中定义这个rpc方法
	req := &chat.GetGroupMembersReq{
		GroupId: groupID,
	}

	resp, err := r.group.GetGroupMembers(ctx, req)
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	return resp.MemberIds, nil
}

func (r *chatAdapterImpl) HandleGroupApply(ctx context.Context, applyID, handlerID string, accept bool, replyMsg string) error {
	req := &chat.HandleGroupApplyReq{
		ApplyId:   applyID,
		HandlerId: handlerID,
		Accept:    accept,
		ReplyMsg:  replyMsg,
	}
	resp, err := r.group.HandleGroupApply(ctx, req)
	if err != nil {
		return err
	}
	return respcheck.ValidateResponseMeta(resp.Meta)
}
