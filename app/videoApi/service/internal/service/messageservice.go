package service

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	v1 "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type MessageServiceService struct {
	v1.UnimplementedMessageServiceServer
	uc  *biz.MessageUsecase
	log *log.Helper
}

func NewMessageServiceService(uc *biz.MessageUsecase, logger log.Logger) *MessageServiceService {
	return &MessageServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *MessageServiceService) SendMessage(ctx context.Context, req *v1.SendMessageReq) (*v1.SendMessageResp, error) {
	input := &biz.SendMessageInput{
		ReceiverID: req.ReceiverId,
		ConvType:   int32(req.ConvType),
		MsgType:    int32(req.MsgType),
		Content: &biz.MessageContent{
			Text:          req.Content.Text,
			ImageURL:      req.Content.ImageUrl,
			ImageWidth:    req.Content.ImageWidth,
			ImageHeight:   req.Content.ImageHeight,
			VoiceURL:      req.Content.VoiceUrl,
			VoiceDuration: req.Content.VoiceDuration,
			VideoURL:      req.Content.VideoUrl,
			VideoCover:    req.Content.VideoCover,
			VideoDuration: req.Content.VideoDuration,
			FileURL:       req.Content.FileUrl,
			FileName:      req.Content.FileName,
			FileSize:      req.Content.FileSize,
			Extra:         req.Content.Extra,
		},
		ClientMsgID: req.ClientMsgId,
	}

	output, err := s.uc.SendMessage(ctx, input)
	if err != nil {
		return &v1.SendMessageResp{}, err
	}

	return &v1.SendMessageResp{
		MessageId: output.MessageID,
	}, nil
}

func (s *MessageServiceService) ListMessages(ctx context.Context, req *v1.ListMessagesReq) (*v1.ListMessagesResp, error) {
	input := &biz.ListMessagesInput{
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
		Limit:     req.Limit,
	}

	output, err := s.uc.ListMessages(ctx, input)
	if err != nil {
		return &v1.ListMessagesResp{}, err
	}

	// 转换消息
	var messages []*v1.Message
	for _, msg := range output.Messages {
		messages = append(messages, &v1.Message{
			Id:         msg.ID,
			SenderId:   msg.SenderID,
			ReceiverId: msg.ReceiverID,
			ConvType:   v1.ConversationType(msg.ConvType),
			MsgType:    v1.MessageType(msg.MsgType),
			Content: &v1.MessageContent{
				Text:          msg.Content.Text,
				ImageUrl:      msg.Content.ImageURL,
				ImageWidth:    msg.Content.ImageWidth,
				ImageHeight:   msg.Content.ImageHeight,
				VoiceUrl:      msg.Content.VoiceURL,
				VoiceDuration: msg.Content.VoiceDuration,
				VideoUrl:      msg.Content.VideoURL,
				VideoCover:    msg.Content.VideoCover,
				VideoDuration: msg.Content.VideoDuration,
				FileUrl:       msg.Content.FileURL,
				FileName:      msg.Content.FileName,
				FileSize:      msg.Content.FileSize,
				Extra:         msg.Content.Extra,
			},
			Status:     v1.MessageStatus(msg.Status),
			IsRecalled: msg.IsRecalled,
			CreatedAt:  msg.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  msg.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &v1.ListMessagesResp{
		Messages:  messages,
		HasMore:   output.HasMore,
		LastMsgId: output.LastMsgID,
	}, nil
}

func (s *MessageServiceService) RecallMessage(ctx context.Context, req *v1.RecallMessageReq) (*v1.RecallMessageResp, error) {
	input := &biz.RecallMessageInput{
		MessageID: req.MessageId,
	}

	err := s.uc.RecallMessage(ctx, input)
	if err != nil {
		return &v1.RecallMessageResp{}, err
	}

	return &v1.RecallMessageResp{}, nil
}

func (s *MessageServiceService) MarkMessagesRead(ctx context.Context, req *v1.MarkMessagesReadReq) (*v1.MarkMessagesReadResp, error) {
	input := &biz.MarkMessagesReadInput{
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
	}

	err := s.uc.MarkMessagesRead(ctx, input)
	if err != nil {
		return &v1.MarkMessagesReadResp{}, err
	}

	return &v1.MarkMessagesReadResp{}, nil
}

func (s *MessageServiceService) ListConversations(ctx context.Context, req *v1.ListConversationsReq) (*v1.ListConversationsResp, error) {
	input := &biz.ListConversationsInput{
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
			Sort:     req.PageStats.Sort,
		},
	}

	output, err := s.uc.ListConversations(ctx, input)
	if err != nil {
		return &v1.ListConversationsResp{}, err
	}

	// 转换会话
	var conversations []*v1.Conversation
	for _, conv := range output.Conversations {
		conversations = append(conversations, &v1.Conversation{
			Id:          conv.ID,
			Type:        v1.ConversationType(conv.Type),
			TargetId:    conv.TargetID,
			LastMessage: conv.LastMessage,
			LastMsgType: v1.MessageType(conv.LastMsgType),
			LastMsgTime: conv.LastMsgTime,
			UnreadCount: conv.UnreadCount,
			UpdatedAt:   conv.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &v1.ListConversationsResp{
		Conversations: conversations,
		PageStats: &v1.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *MessageServiceService) DeleteConversation(ctx context.Context, req *v1.DeleteConversationReq) (*v1.DeleteConversationResp, error) {
	input := &biz.DeleteConversationInput{
		ConversationID: req.ConversationId,
	}

	err := s.uc.DeleteConversation(ctx, input)
	if err != nil {
		return &v1.DeleteConversationResp{}, err
	}

	return &v1.DeleteConversationResp{}, nil
}

// ClearMessages - 清空聊天记录
// 在 service/messageservice.go 中完善 ClearMessages 方法
func (s *MessageServiceService) ClearMessages(ctx context.Context, req *v1.ClearMessagesReq) (*v1.ClearMessagesResp, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return &v1.ClearMessagesResp{}, errors.New("获取用户信息失败")
	}

	// 调用chat适配器清空消息
	err = s.uc.ClearMessages(ctx, userID, req.TargetId, int32(req.ConvType))
	if err != nil {
		s.log.WithContext(ctx).Errorf("清空聊天记录失败: %v", err)
		return &v1.ClearMessagesResp{}, errors.New("清空聊天记录失败")
	}

	return &v1.ClearMessagesResp{}, nil
}

// GetMessage - 获取消息详情
func (s *MessageServiceService) GetMessage(ctx context.Context, req *v1.GetMessageReq) (*v1.GetMessageResp, error) {
	message, err := s.uc.GetMessage(ctx, req.MessageId)
	if err != nil {
		return &v1.GetMessageResp{}, err
	}

	// 转换消息类型
	return &v1.GetMessageResp{
		Message: &v1.Message{
			Id:         message.ID,
			SenderId:   message.SenderID,
			ReceiverId: message.ReceiverID,
			ConvType:   v1.ConversationType(message.ConvType),
			MsgType:    v1.MessageType(message.MsgType),
			Content: &v1.MessageContent{
				Text:          message.Content.Text,
				ImageUrl:      message.Content.ImageURL,
				ImageWidth:    message.Content.ImageWidth,
				ImageHeight:   message.Content.ImageHeight,
				VoiceUrl:      message.Content.VoiceURL,
				VoiceDuration: message.Content.VoiceDuration,
				VideoUrl:      message.Content.VideoURL,
				VideoCover:    message.Content.VideoCover,
				VideoDuration: message.Content.VideoDuration,
				FileUrl:       message.Content.FileURL,
				FileName:      message.Content.FileName,
				FileSize:      message.Content.FileSize,
				Extra:         message.Content.Extra,
			},
			Status:     v1.MessageStatus(message.Status),
			IsRecalled: message.IsRecalled,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  message.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// GetConversation - 获取会话详情
func (s *MessageServiceService) GetConversation(ctx context.Context, req *v1.GetConversationReq) (*v1.GetConversationResp, error) {
	conversation, err := s.uc.GetConversation(ctx, req.TargetId, int32(req.ConvType))
	if err != nil {
		return &v1.GetConversationResp{}, err
	}

	// 转换会话类型
	return &v1.GetConversationResp{
		Conversation: &v1.Conversation{
			Id:          conversation.ID,
			Type:        v1.ConversationType(conversation.Type),
			TargetId:    conversation.TargetID,
			LastMessage: conversation.LastMessage,
			LastMsgType: v1.MessageType(conversation.LastMsgType),
			LastMsgTime: conversation.LastMsgTime,
			UnreadCount: conversation.UnreadCount,
			UpdatedAt:   conversation.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// GetUnreadCount - 获取未读消息数
func (s *MessageServiceService) GetUnreadCount(ctx context.Context, req *v1.GetUnreadCountReq) (*v1.GetUnreadCountResp, error) {
	// 调用chat适配器获取未读数
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return &v1.GetUnreadCountResp{}, errors.New("获取用户信息失败")
	}

	var convUnread int64
	totalUnread, convUnreadMap, err := s.uc.GetUnreadCount(ctx, userID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("获取未读消息数失败: %v", err)
		return &v1.GetUnreadCountResp{}, errors.New("获取未读消息数失败")
	}

	// 如果请求了特定会话的未读数
	if req.TargetId > 0 {
		// 生成会话key，与chat服务保持一致
		convKey := req.TargetId*10 + int64(req.ConvType)
		if count, ok := convUnreadMap[convKey]; ok {
			convUnread = count
		}
	}

	return &v1.GetUnreadCountResp{
		TotalUnread: totalUnread,
		ConvUnread:  convUnread,
	}, nil
}

// UpdateMessageStatus - 更新消息状态
func (s *MessageServiceService) UpdateMessageStatus(ctx context.Context, req *v1.UpdateMessageStatusReq) (*v1.UpdateMessageStatusResp, error) {
	err := s.uc.UpdateMessageStatus(ctx, req.MessageId, int32(req.Status))
	if err != nil {
		return &v1.UpdateMessageStatusResp{}, err
	}

	return &v1.UpdateMessageStatusResp{}, nil
}

// 添加 CreateConversation 方法
func (s *MessageServiceService) CreateConversation(ctx context.Context, req *v1.CreateConversationReq) (*v1.CreateConversationResp, error) {
	input := &biz.CreateConversationInput{
		TargetID:       req.TargetId,
		ConvType:       int32(req.ConvType),
		InitialMessage: req.InitialMessage,
	}

	output, err := s.uc.CreateConversation(ctx, input)
	if err != nil {
		return &v1.CreateConversationResp{}, err
	}

	return &v1.CreateConversationResp{
		ConversationId: output.ConversationID,
	}, nil
}
