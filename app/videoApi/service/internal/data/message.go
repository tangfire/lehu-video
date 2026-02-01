package data

import (
	"context"
	"fmt"
	"github.com/spf13/cast"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
	"time"
)

// 发送消息
func (r *chatAdapterImpl) SendMessage(ctx context.Context, senderID, receiverID string, convType, msgType int32, content *biz.MessageContent, clientMsgID string) (string, string, error) {
	req := &chat.SendMessageReq{
		SenderId:    senderID,
		ReceiverId:  receiverID,
		ConvType:    chat.ConversationType(convType),
		MsgType:     chat.MessageType(msgType),
		ClientMsgId: clientMsgID,
	}

	if content != nil {
		req.Content = &chat.MessageContent{
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
		}
	}

	resp, err := r.message.SendMessage(ctx, req)
	if err != nil {
		return "", "", err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return "", "", err
	}

	return resp.MessageId, resp.ConversationId, nil
}

// 获取消息列表
func (r *chatAdapterImpl) ListMessages(ctx context.Context, userID, conversationID, lastMsgID string, limit int32) ([]*biz.Message, bool, string, error) {
	req := &chat.ListMessagesReq{
		UserId:         userID,
		ConversationId: conversationID,
		LastMsgId:      lastMsgID,
		Limit:          limit,
	}

	resp, err := r.message.ListMessages(ctx, req)
	if err != nil {
		return nil, false, "0", err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, false, "0", err
	}

	messages := make([]*biz.Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		var content *biz.MessageContent
		if m.Content != nil {
			content = &biz.MessageContent{
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
			}
		}

		messages = append(messages, &biz.Message{
			ID:         m.Id,
			SenderID:   m.SenderId,
			ReceiverID: m.ReceiverId,
			ConvType:   int8(m.ConvType),
			MsgType:    int8(m.MsgType),
			Content:    content,
			Status:     int8(m.Status),
			IsRecalled: m.IsRecalled,
			CreatedAt:  parseTime(m.CreatedAt),
			UpdatedAt:  parseTime(m.UpdatedAt),
		})
	}

	return messages, resp.HasMore, resp.LastMsgId, nil
}

func (r *chatAdapterImpl) RecallMessage(ctx context.Context, messageID, userID string) error {
	req := &chat.RecallMessageReq{
		MessageId: messageID,
		UserId:    userID,
	}
	resp, err := r.message.RecallMessage(ctx, req)
	if err != nil {
		return err
	}
	return respcheck.ValidateResponseMeta(resp.Meta)
}

// 标记消息已读
func (r *chatAdapterImpl) MarkMessagesRead(ctx context.Context, userID, conversationID, lastMsgID string) error {
	req := &chat.MarkMessagesReadReq{
		UserId:         userID,
		ConversationId: conversationID,
		LastMsgId:      lastMsgID,
	}
	resp, err := r.message.MarkMessagesRead(ctx, req)
	if err != nil {
		return err
	}
	return respcheck.ValidateResponseMeta(resp.Meta)
}

// 新增：更新消息状态
func (r *chatAdapterImpl) UpdateMessageStatus(ctx context.Context, messageID string, status int32) error {
	resp, err := r.message.UpdateMessageStatus(ctx, &chat.UpdateMessageStatusReq{
		MessageId:  messageID,
		Status:     chat.MessageStatus(status),
		OperatorId: cast.ToString(0),
	})
	if err != nil {
		return err
	}
	return respcheck.ValidateResponseMeta(resp.Meta)
}

// 新增：获取消息详情
func (r *chatAdapterImpl) GetMessageByID(ctx context.Context, messageID string) (*biz.Message, error) {
	resp, err := r.message.GetMessage(ctx, &chat.GetMessageReq{
		MessageId: messageID,
	})
	if err != nil {
		return nil, err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}
	if resp.Message == nil {
		return nil, nil
	}

	m := resp.Message
	var content *biz.MessageContent
	if m.Content != nil {
		content = &biz.MessageContent{
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
		}
	}

	return &biz.Message{
		ID:         m.Id,
		SenderID:   m.SenderId,
		ReceiverID: m.ReceiverId,
		ConvType:   int8(m.ConvType),
		MsgType:    int8(m.MsgType),
		Content:    content,
		Status:     int8(m.Status),
		IsRecalled: m.IsRecalled,
		CreatedAt:  parseTime(m.CreatedAt),
		UpdatedAt:  parseTime(m.UpdatedAt),
	}, nil
}

func (r *chatAdapterImpl) GetUnreadCount(ctx context.Context, userID string) (int64, map[string]int64, error) {
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

func (r *chatAdapterImpl) ListConversations(ctx context.Context, userID string, pageStats *biz.PageStats) (int64, []*biz.Conversation, error) {
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
			ID:          c.Conversation.Id,
			UserID:      userID,
			Type:        int32(c.Conversation.Type),
			TargetID:    c.Conversation.TargetId,
			LastMessage: c.Conversation.LastMessage,
			LastMsgType: (*int32)(c.Conversation.LastMsgType),
			LastMsgTime: parseUnixPointer(c.Conversation.LastMsgTime),
			UnreadCount: int32(c.UnreadCount),
			UpdatedAt:   parseTime(c.Conversation.UpdatedAt),
		})
	}

	return int64(resp.PageStats.Total), conversations, nil
}

func parseUnixPointer(ts *int64) *time.Time {
	if ts == nil || *ts == 0 {
		return nil
	}
	t := time.Unix(*ts, 0)
	return &t
}

func (r *chatAdapterImpl) DeleteConversation(ctx context.Context, userID, conversationID string) error {
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

func (r *chatAdapterImpl) ClearMessages(ctx context.Context, userID, conversationID string) error {
	req := &chat.ClearMessagesReq{
		UserId:         userID,
		ConversationId: conversationID,
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

// 检查用户关系（单聊检查好友，群聊检查群成员）
func (r *chatAdapterImpl) CheckUserRelation(ctx context.Context, userID, targetID string, convType int32) (bool, error) {
	if convType == 0 { // 单聊
		// 检查好友关系
		isFriend, _, err := r.CheckFriendRelation(ctx, userID, targetID)
		if err != nil {
			return false, fmt.Errorf("检查好友关系失败: %v", err)
		}
		return isFriend, nil
	} else if convType == 1 { // 群聊
		// 检查是否是群成员
		// 注意：这里需要chat服务有IsGroupMember方法
		isMemberReq := &chat.IsGroupMemberReq{
			UserId:  userID,
			GroupId: targetID,
		}

		// 需要确保chat服务有这个RPC
		resp, err := r.group.IsGroupMember(ctx, isMemberReq)
		if err != nil {
			return false, fmt.Errorf("检查群成员关系失败: %v", err)
		}

		err = respcheck.ValidateResponseMeta(resp.Meta)
		if err != nil {
			return false, err
		}

		return resp.IsMember, nil
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
