package data

import (
	"context"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
	"time"
)

func (r *chatAdapterImpl) SendFriendApply(ctx context.Context, applicantID, receiverID, applyReason string) (string, error) {
	req := &chat.SendFriendApplyReq{
		ApplicantId: applicantID,
		ReceiverId:  receiverID,
		ApplyReason: applyReason,
	}

	resp, err := r.friend.SendFriendApply(ctx, req)
	if err != nil {
		return "0", err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", err
	}

	return resp.ApplyId, nil
}

func (r *chatAdapterImpl) HandleFriendApply(ctx context.Context, applyID, handlerID string, accept bool) error {
	req := &chat.HandleFriendApplyReq{
		ApplyId:   applyID,
		HandlerId: handlerID,
		Accept:    accept,
	}

	resp, err := r.friend.HandleFriendApply(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) ListFriendApplies(ctx context.Context, userID string, status *int32, pageStats *biz.PageStats) (int64, []*biz.FriendApply, error) {
	req := &chat.ListFriendAppliesReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	}

	if status != nil {
		req.Status = status
	}

	resp, err := r.friend.ListFriendApplies(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var applies []*biz.FriendApply
	for _, a := range resp.Applies {
		apply := &biz.FriendApply{
			ID:          a.Id,
			ApplicantID: a.ApplicantId,
			ReceiverID:  a.ReceiverId,
			ApplyReason: a.ApplyReason,
			Status:      a.Status,
			HandledAt:   parseTimePointer(a.HandledAt),
			CreatedAt:   parseTime(a.CreatedAt),
		}
		applies = append(applies, apply)
	}

	return int64(resp.PageStats.Total), applies, nil
}

func (r *chatAdapterImpl) DeleteFriend(ctx context.Context, userID, friendID string) error {
	req := &chat.DeleteFriendReq{
		UserId:   userID,
		FriendId: friendID,
	}

	resp, err := r.friend.DeleteFriend(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) UpdateFriendRemark(ctx context.Context, userID, friendID, remark string) error {
	req := &chat.UpdateFriendRemarkReq{
		UserId:   userID,
		FriendId: friendID,
		Remark:   remark,
	}

	resp, err := r.friend.UpdateFriendRemark(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) SetFriendGroup(ctx context.Context, userID, friendID, groupName string) error {
	req := &chat.SetFriendGroupReq{
		UserId:    userID,
		FriendId:  friendID,
		GroupName: groupName,
	}

	resp, err := r.friend.SetFriendGroup(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) CheckFriendRelation(ctx context.Context, userID, targetID string) (bool, int32, error) {
	req := &chat.CheckFriendRelationReq{
		UserId:   userID,
		TargetId: targetID,
	}

	resp, err := r.friend.CheckFriendRelation(ctx, req)
	if err != nil {
		return false, 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return false, 0, err
	}

	return resp.IsFriend, resp.Status, nil
}

// 获取用户在线状态
func (r *chatAdapterImpl) GetUserOnlineStatus(ctx context.Context, userID string) (*biz.UserSocialInfo, error) {
	req := &chat.GetUserOnlineStatusReq{
		UserId: userID,
	}

	resp, err := r.friend.GetUserOnlineStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	return &biz.UserSocialInfo{
		OnlineStatus:   resp.OnlineStatus,
		LastOnlineTime: resp.LastOnlineTime,
	}, nil
}

// 在 chatadapter.go 的 friend.go 部分添加以下方法

func (r *chatAdapterImpl) GetUserRelation(ctx context.Context, userID, targetUserID string) (*biz.UserRelationInfo, error) {
	// 1. 检查好友关系
	isFriend, status, err := r.CheckFriendRelation(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}

	// 2. 获取关注状态（需要调用core服务）
	// 注意：这里需要调用core服务的关注接口
	// 由于缺少core服务的关注状态接口，这里简化处理
	// 实际项目中应该调用core.IsFollowing方法

	// 3. 获取好友备注和分组
	var remark, groupName string
	if isFriend {
		// 查询好友备注和分组
		// 这里可以从数据库查询，或者通过其他方式获取
		// 简化处理，实际项目中需要实现
	}

	return &biz.UserRelationInfo{
		UserID:       userID,
		TargetUserID: targetUserID,
		IsFollowing:  false, // 需要从core服务获取
		IsFollower:   false, // 需要从core服务获取
		IsFriend:     isFriend,
		FriendStatus: status,
		Remark:       remark,
		GroupName:    groupName,
		CreatedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (r *chatAdapterImpl) BatchGetUserRelations(ctx context.Context, userID string, targetUserIDs []string) (map[string]*biz.UserRelationInfo, error) {
	if len(targetUserIDs) == 0 {
		return make(map[string]*biz.UserRelationInfo), nil
	}

	result := make(map[string]*biz.UserRelationInfo)

	// 批量检查好友关系
	for _, targetID := range targetUserIDs {
		isFriend, status, err := r.CheckFriendRelation(ctx, userID, targetID)
		if err != nil {
			// 记录错误但继续处理其他用户
			continue
		}

		result[targetID] = &biz.UserRelationInfo{
			UserID:       userID,
			TargetUserID: targetID,
			IsFollowing:  false, // 需要从core服务获取
			IsFollower:   false, // 需要从core服务获取
			IsFriend:     isFriend,
			FriendStatus: status,
			CreatedAt:    time.Now().Format("2006-01-02 15:04:05"),
		}
	}

	return result, nil
}

// ==================== 好友相关 ====================

func (r *chatAdapterImpl) ListFriends(ctx context.Context, userID string, groupName *string, pageStats *biz.PageStats) (int64, []*biz.FriendRelation, error) {
	req := &chat.ListFriendsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
		GroupName: groupName,
	}
	resp, err := r.friend.ListFriends(ctx, req)
	if err != nil {
		return 0, nil, err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return 0, nil, err
	}
	relations := make([]*biz.FriendRelation, 0, len(resp.Friends))
	for _, f := range resp.Friends {
		relations = append(relations, &biz.FriendRelation{
			ID:        f.Id,
			FriendID:  f.FriendId,
			Remark:    f.Remark,
			GroupName: f.GroupName,
			Status:    f.Status,
			CreatedAt: f.CreatedAt,
		})
	}
	return int64(resp.PageStats.Total), relations, nil
}

// BatchGetUserOnlineStatus 批量获取在线状态
func (r *chatAdapterImpl) BatchGetUserOnlineStatus(ctx context.Context, userIDs []string) (map[string]int32, error) {
	req := &chat.BatchGetUserOnlineStatusReq{
		UserIds: userIDs,
	}
	resp, err := r.friend.BatchGetUserOnlineStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}
	return resp.OnlineStatus, nil
}

// UpdateUserOnlineStatus 更新在线状态
func (r *chatAdapterImpl) UpdateUserOnlineStatus(ctx context.Context, userID string, status int32, deviceType string) error {
	req := &chat.UpdateUserOnlineStatusReq{
		UserId:       userID,
		OnlineStatus: status,
		DeviceType:   deviceType,
	}
	resp, err := r.friend.UpdateUserOnlineStatus(ctx, req)
	if err != nil {
		return err
	}
	return respcheck.ValidateResponseMeta(resp.Meta)
}
