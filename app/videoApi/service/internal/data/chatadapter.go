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
}

func NewChatAdapter(group chat.GroupServiceClient, message chat.MessageServiceClient) biz.ChatAdapter {
	return &chatAdapterImpl{
		group:   group,
		message: message,
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

func (r *chatAdapterImpl) CreateGroup(ctx context.Context, ownerID int64, name, notice string, addMode int32, avatar string) (int64, error) {
	resp, err := r.group.CreateGroup(ctx, &chat.CreateGroupReq{
		OwnerId: ownerID,
		Name:    name,
		Notice:  notice,
		AddMode: addMode,
		Avatar:  avatar,
	})
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
			Page: pageStats.Page,
			Size: pageStats.PageSize,
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
			Members:   g.Members,
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
	resp, err := r.group.CheckGroupAddMode(ctx, &chat.CheckGroupAddModeReq{
		GroupId: groupID,
	})
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
	resp, err := r.group.EnterGroupDirectly(ctx, &chat.EnterGroupDirectlyReq{
		UserId:  userID,
		GroupId: groupID,
	})
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
	resp, err := r.group.ApplyJoinGroup(ctx, &chat.ApplyJoinGroupReq{
		UserId:      userID,
		GroupId:     groupID,
		ApplyReason: applyReason,
	})
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatAdapterImpl) LeaveGroup(ctx context.Context, userID, groupID int64) error {
	resp, err := r.group.LeaveGroup(ctx, &chat.LeaveGroupReq{
		UserId:  userID,
		GroupId: groupID,
	})
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
	resp, err := r.group.DismissGroup(ctx, &chat.DismissGroupReq{
		OwnerId: ownerID,
		GroupId: groupID,
	})
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
	resp, err := r.group.GetGroupInfo(ctx, &chat.GetGroupInfoReq{
		GroupId: groupID,
	})
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	if resp.Group == nil {
		return nil, nil
	}

	g := resp.Group
	return &biz.Group{
		ID:        g.Id,
		Name:      g.Name,
		Notice:    g.Notice,
		Members:   g.Members,
		MemberCnt: int(g.MemberCnt),
		OwnerID:   g.OwnerId,
		AddMode:   g.AddMode,
		Avatar:    g.Avatar,
		Status:    g.Status,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}, nil
}

func (r *chatAdapterImpl) ListMyJoinedGroups(ctx context.Context, userID int64, pageStats *biz.PageStats) (int64, []*biz.Group, error) {
	req := &chat.ListMyJoinedGroupsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: pageStats.Page,
			Size: pageStats.PageSize,
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
			Members:   g.Members,
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
		// 直接创建 Message 结构体，Content 为 *MessageContent 类型
		message := &biz.Message{
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
		}

		messages = append(messages, message)
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

func (r *chatAdapterImpl) ListConversations(ctx context.Context, userID int64, pageStats *biz.PageStats) (int64, []*biz.Conversation, error) {
	req := &chat.ListConversationsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: pageStats.Page,
			Size: pageStats.PageSize,
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
			UnreadCount: int32(int(c.UnreadCount)),
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

// 辅助函数
func parseTime(timeStr string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return time.Now()
	}
	return t
}

func parseUnixTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}
