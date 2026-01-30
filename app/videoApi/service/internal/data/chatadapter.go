package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

type chatAdapterImpl struct {
	group   chat.GroupServiceClient
	message chat.MessageServiceClient
	friend  chat.FriendServiceClient
}

func NewChatAdapter(
	group chat.GroupServiceClient,
	message chat.MessageServiceClient,
	friend chat.FriendServiceClient,
) biz.ChatAdapter {
	return &chatAdapterImpl{
		group:   group,
		message: message,
		friend:  friend,
	}
}

func NewGroupServiceClient(r registry.Discovery) chat.GroupServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return chat.NewGroupServiceClient(conn)
}

func NewMessageServiceClient(r registry.Discovery) chat.MessageServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return chat.NewMessageServiceClient(conn)
}

func NewFriendServiceClient(r registry.Discovery) chat.FriendServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return chat.NewFriendServiceClient(conn)
}

// 消息相关方法
func (r *chatAdapterImpl) SendMessage(ctx context.Context, senderID, receiverID int64, convType, msgType int32, content *biz.MessageContent, clientMsgID string) (int64, error) {
	req := &chat.SendMessageReq{
		SenderId:   senderID,
		ReceiverId: receiverID,
		ConvType:   chat.ConversationType(convType),
		MsgType:    chat.MessageType(msgType),
		Content: &chat.MessageContent{
			Text:          content.Text,
			ImageUrl:      content.ImageURL,
			ImageWidth:    content.ImageWidth,
			ImageHeight:   content.ImageHeight,
			VoiceUrl:      content.VoiceURL,
			VoiceDuration: content.VoiceDuration,
			VideoUrl:      content.VideoURL,
			VideoCover:    content.VideoCover,
			VideoDuration: content.VideoDuration,
			FileUrl:       content.FileURL,
			FileName:      content.FileName,
			FileSize:      content.FileSize,
			Extra:         content.Extra,
		},
		ClientMsgId: clientMsgID,
	}

	resp, err := r.message.SendMessage(ctx, req)
	if err != nil {
		return 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}

	return resp.MessageId, nil
}

func (r *chatAdapterImpl) ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int32) ([]*biz.Message, bool, int64, error) {
	req := &chat.ListMessagesReq{
		UserId:    userID,
		TargetId:  targetID,
		ConvType:  chat.ConversationType(convType),
		LastMsgId: lastMsgID,
		Limit:     limit,
	}

	resp, err := r.message.ListMessages(ctx, req)
	if err != nil {
		return nil, false, 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, false, 0, err
	}

	var messages []*biz.Message
	for _, m := range resp.Messages {
		messages = append(messages, &biz.Message{
			ID:         m.Id,
			SenderID:   m.SenderId,
			ReceiverID: m.ReceiverId,
			ConvType:   int8(m.ConvType),
			MsgType:    int8(m.MsgType),
			Content: &biz.MessageContent{
				Text:          m.Content.Text,
				ImageURL:      m.Content.ImageUrl,
				ImageWidth:    m.Content.ImageWidth,
				ImageHeight:   m.Content.ImageHeight,
				VoiceURL:      m.Content.VoiceUrl,
				VoiceDuration: m.Content.VoiceDuration,
				VideoURL:      m.Content.VideoUrl,
				VideoCover:    m.Content.VideoCover,
				VideoDuration: m.Content.VideoDuration,
				FileURL:       m.Content.FileUrl,
				FileName:      m.Content.FileName,
				FileSize:      m.Content.FileSize,
				Extra:         m.Content.Extra,
			},
			Status:     int8(m.Status),
			IsRecalled: m.IsRecalled,
			CreatedAt:  parseTime(m.CreatedAt),
			UpdatedAt:  parseTime(m.UpdatedAt),
		})
	}

	return messages, resp.HasMore, resp.LastMsgId, nil
}

func (r *chatAdapterImpl) RecallMessage(ctx context.Context, messageID, userID int64) error {
	req := &chat.RecallMessageReq{
		MessageId: messageID,
		UserId:    userID,
	}

	resp, err := r.message.RecallMessage(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) MarkMessagesRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error {
	req := &chat.MarkMessagesReadReq{
		UserId:    userID,
		TargetId:  targetID,
		ConvType:  chat.ConversationType(convType),
		LastMsgId: lastMsgID,
	}

	resp, err := r.message.MarkMessagesRead(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

// 新增：更新消息状态
func (r *chatAdapterImpl) UpdateMessageStatus(ctx context.Context, messageID int64, status int32) error {
	req := &chat.UpdateMessageStatusReq{
		MessageId:  messageID,
		Status:     chat.MessageStatus(status),
		OperatorId: 0, // 系统操作
	}

	resp, err := r.message.UpdateMessageStatus(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

// 新增：获取消息详情
func (r *chatAdapterImpl) GetMessageByID(ctx context.Context, messageID int64) (*biz.Message, error) {
	req := &chat.GetMessageReq{
		MessageId: messageID,
	}

	resp, err := r.message.GetMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	m := resp.Message
	return &biz.Message{
		ID:         m.Id,
		SenderID:   m.SenderId,
		ReceiverID: m.ReceiverId,
		ConvType:   int8(m.ConvType),
		MsgType:    int8(m.MsgType),
		Content: &biz.MessageContent{
			Text:          m.Content.Text,
			ImageURL:      m.Content.ImageUrl,
			ImageWidth:    m.Content.ImageWidth,
			ImageHeight:   m.Content.ImageHeight,
			VoiceURL:      m.Content.VoiceUrl,
			VoiceDuration: m.Content.VoiceDuration,
			VideoURL:      m.Content.VideoUrl,
			VideoCover:    m.Content.VideoCover,
			VideoDuration: m.Content.VideoDuration,
			FileURL:       m.Content.FileUrl,
			FileName:      m.Content.FileName,
			FileSize:      m.Content.FileSize,
			Extra:         m.Content.Extra,
		},
		Status:     int8(m.Status),
		IsRecalled: m.IsRecalled,
		CreatedAt:  parseTime(m.CreatedAt),
		UpdatedAt:  parseTime(m.UpdatedAt),
	}, nil
}

// 新增：获取会话详情
func (r *chatAdapterImpl) GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*biz.Conversation, error) {
	req := &chat.GetConversationReq{
		UserId:   userID,
		TargetId: targetID,
		ConvType: chat.ConversationType(convType),
	}

	resp, err := r.message.GetConversation(ctx, req)
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	c := resp.Conversation
	return &biz.Conversation{
		ID:          c.Id,
		UserID:      userID,
		Type:        int8(c.Type),
		TargetID:    c.TargetId,
		LastMessage: c.LastMessage,
		LastMsgType: int8(c.LastMsgType),
		LastMsgTime: c.LastMsgTime,
		UnreadCount: int32(c.UnreadCount),
		UpdatedAt:   parseTime(c.UpdatedAt),
	}, nil
}

func (r *chatAdapterImpl) GetUnreadCount(ctx context.Context, userID int64) (int64, map[int64]int64, error) {
	req := &chat.GetUnreadCountReq{
		UserId: userID,
	}

	resp, err := r.message.GetUnreadCount(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	return resp.TotalUnread, resp.ConvUnread, nil
}

func (r *chatAdapterImpl) ListConversations(ctx context.Context, userID int64, pageStats *biz.PageStats) (int64, []*biz.Conversation, error) {
	req := &chat.ListConversationsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
			Sort: pageStats.Sort,
		},
	}

	resp, err := r.message.ListConversations(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var conversations []*biz.Conversation
	for _, c := range resp.Conversations {
		conversations = append(conversations, &biz.Conversation{
			ID:          c.Id,
			UserID:      userID,
			Type:        int8(c.Type),
			TargetID:    c.TargetId,
			LastMessage: c.LastMessage,
			LastMsgType: int8(c.LastMsgType),
			LastMsgTime: c.LastMsgTime,
			UnreadCount: int32(c.UnreadCount),
			UpdatedAt:   parseTime(c.UpdatedAt),
		})
	}

	return int64(resp.PageStats.Total), conversations, nil
}

func (r *chatAdapterImpl) DeleteConversation(ctx context.Context, userID, conversationID int64) error {
	req := &chat.DeleteConversationReq{
		UserId:         userID,
		ConversationId: conversationID,
	}

	resp, err := r.message.DeleteConversation(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) ClearMessages(ctx context.Context, userID, targetID int64, convType int32) error {
	req := &chat.ClearMessagesReq{
		UserId:   userID,
		TargetId: targetID,
		ConvType: chat.ConversationType(convType),
	}

	resp, err := r.message.ClearMessages(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

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

func (r *chatAdapterImpl) SendFriendApply(ctx context.Context, applicantID, receiverID int64, applyReason string) (int64, error) {
	req := &chat.SendFriendApplyReq{
		ApplicantId: applicantID,
		ReceiverId:  receiverID,
		ApplyReason: applyReason,
	}

	resp, err := r.friend.SendFriendApply(ctx, req)
	if err != nil {
		return 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}

	return resp.ApplyId, nil
}

func (r *chatAdapterImpl) HandleFriendApply(ctx context.Context, applyID, handlerID int64, accept bool) error {
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

func (r *chatAdapterImpl) ListFriendApplies(ctx context.Context, userID int64, page, size int32, status *int32) (int64, []*biz.FriendApply, error) {
	req := &chat.ListFriendAppliesReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: page,
			Size: size,
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

func (r *chatAdapterImpl) ListFriends(ctx context.Context, userID int64, page, size int32, groupName *string) (int64, []*biz.FriendInfo, error) {
	req := &chat.ListFriendsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: page,
			Size: size,
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

func (r *chatAdapterImpl) DeleteFriend(ctx context.Context, userID, friendID int64) error {
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

func (r *chatAdapterImpl) UpdateFriendRemark(ctx context.Context, userID, friendID int64, remark string) error {
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

func (r *chatAdapterImpl) SetFriendGroup(ctx context.Context, userID, friendID int64, groupName string) error {
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

func (r *chatAdapterImpl) CheckFriendRelation(ctx context.Context, userID, targetID int64) (bool, int32, error) {
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

func (r *chatAdapterImpl) GetUserOnlineStatus(ctx context.Context, userID int64) (int32, string, error) {
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

func (r *chatAdapterImpl) BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]int32, error) {
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

// 群聊相关方法
func (r *chatAdapterImpl) CreateGroup(ctx context.Context, ownerID int64, name, notice string, addMode int32, avatar string) (int64, error) {
	req := &chat.CreateGroupReq{
		OwnerId: ownerID,
		Name:    name,
		Notice:  notice,
		AddMode: addMode,
		Avatar:  avatar,
	}

	resp, err := r.group.CreateGroup(ctx, req)
	if err != nil {
		return 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}

	return resp.GroupId, nil
}

func (r *chatAdapterImpl) LoadMyGroup(ctx context.Context, ownerID int64, pageStats *biz.PageStats) (int64, []*biz.Group, error) {
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

func (r *chatAdapterImpl) CheckGroupAddMode(ctx context.Context, groupID int64) (int32, error) {
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

func (r *chatAdapterImpl) EnterGroupDirectly(ctx context.Context, userID, groupID int64) error {
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

func (r *chatAdapterImpl) ApplyJoinGroup(ctx context.Context, userID, groupID int64, applyReason string) error {
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

func (r *chatAdapterImpl) GetGroupInfo(ctx context.Context, groupID int64) (*biz.Group, error) {
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

func (r *chatAdapterImpl) ListMyJoinedGroups(ctx context.Context, userID int64, pageStats *biz.PageStats) (int64, []*biz.Group, error) {
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

func (r *chatAdapterImpl) LeaveGroup(ctx context.Context, userID, groupID int64) error {
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

func (r *chatAdapterImpl) DismissGroup(ctx context.Context, ownerID, groupID int64) error {
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
func (r *chatAdapterImpl) GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error) {
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

// 检查用户关系（单聊时检查好友，群聊时检查群成员）
func (r *chatAdapterImpl) CheckUserRelation(ctx context.Context, userID, targetID int64, convType int32) (bool, error) {
	if convType == 0 { // 单聊
		// 检查好友关系
		isFriend, _, err := r.CheckFriendRelation(ctx, userID, targetID)
		return isFriend, err
	} else if convType == 1 { // 群聊
		// 检查是否是群成员

		resp, err := r.group.IsGroupMember(ctx, &chat.IsGroupMemberReq{
			GroupId: targetID,
			UserId:  userID,
		})
		return resp.IsMember, err
	}
	return false, nil
}

// 辅助函数
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// 尝试多种时间格式
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	return time.Now()
}

func parseTimePointer(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}
	t := parseTime(timeStr)
	return &t
}
