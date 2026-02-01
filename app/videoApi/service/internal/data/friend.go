package data

import (
	"context"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

// 好友相关方法
func (r *chatAdapterImpl) SearchUsers(ctx context.Context, keyword string, page, size int32) (int64, []*biz.UserInfo, error) {
	req := &chat.SearchUsersReq{
		Keyword: keyword,
		PageStats: &chat.PageStatsReq{
			Page: page,
			Size: size,
		},
	}

	resp, err := r.friend.SearchUsers(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var users []*biz.UserInfo
	for _, u := range resp.Users {
		users = append(users, &biz.UserInfo{
			Id:             u.Id,
			Name:           u.Name,
			Nickname:       u.Nickname,
			Avatar:         u.Avatar,
			Signature:      u.Signature,
			Gender:         u.Gender,
			OnlineStatus:   u.OnlineStatus,
			LastOnlineTime: parseTime(u.LastOnlineTime),
		})
	}

	return int64(resp.PageStats.Total), users, nil
}

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
			ApplicantID: a.Applicant.Id,
			ReceiverID:  a.Receiver.Id,
			ApplyReason: a.ApplyReason,
			Status:      a.Status,
			HandledAt:   parseTimePointer(a.HandledAt),
			CreatedAt:   parseTime(a.CreatedAt),
		}
		applies = append(applies, apply)
	}

	return int64(resp.PageStats.Total), applies, nil
}

func (r *chatAdapterImpl) ListFriends(ctx context.Context, userID string, groupName *string, pageStats *biz.PageStats) (int64, []*biz.FriendInfo, error) {
	req := &chat.ListFriendsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	}

	if groupName != nil {
		req.GroupName = groupName
	}

	resp, err := r.friend.ListFriends(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var friends []*biz.FriendInfo
	for _, f := range resp.Friends {
		friendInfo := &biz.FriendInfo{
			ID:        f.Id,
			Remark:    f.Remark,
			GroupName: f.GroupName,
			Status:    f.Status,
			CreatedAt: parseTime(f.CreatedAt),
		}

		if f.Friend != nil {
			friendInfo.Friend = &biz.UserInfo{
				Id: f.Friend.Id,
			}
		}

		friends = append(friends, friendInfo)
	}

	return int64(resp.PageStats.Total), friends, nil
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

func (r *chatAdapterImpl) GetUserOnlineStatus(ctx context.Context, userID string) (int32, string, error) {
	req := &chat.GetUserOnlineStatusReq{
		UserId: userID,
	}

	resp, err := r.friend.GetUserOnlineStatus(ctx, req)
	if err != nil {
		return 0, "", err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, "", err
	}

	return resp.OnlineStatus, resp.LastOnlineTime, nil
}

func (r *chatAdapterImpl) BatchGetUserOnlineStatus(ctx context.Context, userIDs []string) (map[string]int32, error) {
	req := &chat.BatchGetUserOnlineStatusReq{
		UserIds: userIDs,
	}

	resp, err := r.friend.BatchGetUserOnlineStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	return resp.OnlineStatus, nil
}
