package service

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/pkg/utils"
)

type MessageServiceService struct {
	pb.UnimplementedMessageServiceServer
	uc  *biz.MessageUsecase
	log *log.Helper
}

func NewMessageServiceService(uc *biz.MessageUsecase, logger log.Logger) *MessageServiceService {
	return &MessageServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *MessageServiceService) SendMessage(ctx context.Context, req *pb.SendMessageReq) (*pb.SendMessageResp, error) {
	// ✅ 构建Command
	cmd := &biz.SendMessageCommand{
		SenderID:    req.SenderId,
		ReceiverID:  req.ReceiverId,
		ConvType:    int32(req.ConvType),
		MsgType:     int32(req.MsgType),
		ClientMsgID: req.ClientMsgId,
	}

	// 设置消息内容
	if req.Content != nil {
		cmd.Content = &biz.MessageContent{
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
		}
	}

	// ✅ 调用业务层
	result, err := s.uc.SendMessage(ctx, cmd)
	if err != nil {
		return &pb.SendMessageResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.SendMessageResp{
		Meta:      utils.GetSuccessMeta(),
		MessageId: result.MessageID,
	}, nil
}

func (s *MessageServiceService) ListMessages(ctx context.Context, req *pb.ListMessagesReq) (*pb.ListMessagesResp, error) {
	// ✅ 构建Query
	query := &biz.ListMessagesQuery{
		UserID:    req.UserId,
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
		Limit:     int(req.Limit),
	}

	// ✅ 调用业务层
	result, err := s.uc.ListMessages(ctx, query)
	if err != nil {
		return &pb.ListMessagesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	messages := make([]*pb.Message, 0, len(result.Messages))
	for _, msg := range result.Messages {
		messages = append(messages, &pb.Message{
			Id:         msg.ID,
			SenderId:   msg.SenderID,
			ReceiverId: msg.ReceiverID,
			ConvType:   pb.ConversationType(msg.ConvType),
			MsgType:    pb.MessageType(msg.MsgType),
			Content: &pb.MessageContent{
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
			Status:     pb.MessageStatus(msg.Status),
			IsRecalled: msg.IsRecalled,
			CreatedAt:  msg.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  msg.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListMessagesResp{
		Meta:      utils.GetSuccessMeta(),
		Messages:  messages,
		HasMore:   result.HasMore,
		LastMsgId: result.LastMsgID,
	}, nil
}

func (s *MessageServiceService) RecallMessage(ctx context.Context, req *pb.RecallMessageReq) (*pb.RecallMessageResp, error) {
	// ✅ 构建Command
	cmd := &biz.RecallMessageCommand{
		MessageID: req.MessageId,
		UserID:    req.UserId,
	}

	// ✅ 调用业务层
	_, err := s.uc.RecallMessage(ctx, cmd)
	if err != nil {
		return &pb.RecallMessageResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.RecallMessageResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) MarkMessagesRead(ctx context.Context, req *pb.MarkMessagesReadReq) (*pb.MarkMessagesReadResp, error) {
	// ✅ 构建Command
	cmd := &biz.MarkMessagesReadCommand{
		UserID:    req.UserId,
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
	}

	// ✅ 调用业务层
	_, err := s.uc.MarkMessagesRead(ctx, cmd)
	if err != nil {
		return &pb.MarkMessagesReadResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.MarkMessagesReadResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountReq) (*pb.GetUnreadCountResp, error) {
	// ✅ 构建Query
	query := &biz.GetUnreadCountQuery{
		UserID: req.UserId,
	}

	// ✅ 调用业务层
	result, err := s.uc.GetUnreadCount(ctx, query)
	if err != nil {
		return &pb.GetUnreadCountResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.GetUnreadCountResp{
		Meta:        utils.GetSuccessMeta(),
		TotalUnread: result.TotalUnread,
		ConvUnread:  result.ConvUnread,
	}, nil
}

func (s *MessageServiceService) ListConversations(ctx context.Context, req *pb.ListConversationsReq) (*pb.ListConversationsResp, error) {
	// ✅ 构建Query
	query := &biz.ListConversationsQuery{
		UserID: req.UserId,
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	// ✅ 调用业务层
	result, err := s.uc.ListConversations(ctx, query)
	if err != nil {
		return &pb.ListConversationsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	conversations := make([]*pb.Conversation, 0, len(result.Conversations))
	for _, conv := range result.Conversations {
		conversations = append(conversations, &pb.Conversation{
			Id:          conv.ID,
			Type:        pb.ConversationType(conv.Type),
			TargetId:    conv.TargetID,
			LastMessage: conv.LastMessage,
			LastMsgType: pb.MessageType(conv.LastMsgType),
			LastMsgTime: conv.LastMsgTime.Unix(), // 转换为时间戳
			UnreadCount: int64(conv.UnreadCount),
			UpdatedAt:   conv.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListConversationsResp{
		Meta:          utils.GetSuccessMeta(),
		Conversations: conversations,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *MessageServiceService) DeleteConversation(ctx context.Context, req *pb.DeleteConversationReq) (*pb.DeleteConversationResp, error) {
	// ✅ 构建Command
	cmd := &biz.DeleteConversationCommand{
		UserID:         req.UserId,
		ConversationID: req.ConversationId,
	}

	// ✅ 调用业务层
	_, err := s.uc.DeleteConversation(ctx, cmd)
	if err != nil {
		return &pb.DeleteConversationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.DeleteConversationResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) UpdateMessageStatus(ctx context.Context, req *pb.UpdateMessageStatusReq) (*pb.UpdateMessageStatusResp, error) {
	// ✅ 构建Command
	cmd := &biz.UpdateMessageStatusCommand{
		MessageID:  req.MessageId,
		Status:     int32(req.Status),
		OperatorID: req.OperatorId,
	}

	// ✅ 调用业务层
	_, err := s.uc.UpdateMessageStatus(ctx, cmd)
	if err != nil {
		return &pb.UpdateMessageStatusResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.UpdateMessageStatusResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) GetMessage(ctx context.Context, req *pb.GetMessageReq) (*pb.GetMessageResp, error) {
	// ✅ 调用业务层获取消息
	message, err := s.uc.GetMessage(ctx, req.MessageId)
	if err != nil {
		return &pb.GetMessageResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// ✅ 转换为proto结构
	pbMessage := &pb.Message{
		Id:         message.ID,
		SenderId:   message.SenderID,
		ReceiverId: message.ReceiverID,
		ConvType:   pb.ConversationType(message.ConvType),
		MsgType:    pb.MessageType(message.MsgType),
		Content: &pb.MessageContent{
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
		Status:     pb.MessageStatus(message.Status),
		IsRecalled: message.IsRecalled,
		CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  message.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return &pb.GetMessageResp{
		Meta:    utils.GetSuccessMeta(),
		Message: pbMessage,
	}, nil
}

func (s *MessageServiceService) GetConversation(ctx context.Context, req *pb.GetConversationReq) (*pb.GetConversationResp, error) {
	// ✅ 调用业务层获取会话
	conversation, err := s.uc.GetConversation(ctx, req.UserId, req.TargetId, int32(req.ConvType))
	if err != nil {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if conversation == nil {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(errors.New("会话不存在")),
		}, nil
	}

	// ✅ 转换为proto结构
	pbConversation := &pb.Conversation{
		Id:          conversation.ID,
		Type:        pb.ConversationType(conversation.Type),
		TargetId:    conversation.TargetID,
		LastMessage: conversation.LastMessage,
		LastMsgType: pb.MessageType(conversation.LastMsgType),
		LastMsgTime: conversation.LastMsgTime.Unix(),
		UnreadCount: int64(conversation.UnreadCount),
		UpdatedAt:   conversation.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return &pb.GetConversationResp{
		Meta:         utils.GetSuccessMeta(),
		Conversation: pbConversation,
	}, nil
}

func (s *MessageServiceService) ClearMessages(ctx context.Context, req *pb.ClearMessagesReq) (*pb.ClearMessagesResp, error) {
	// ✅ 调用业务层清空消息
	_, err := s.uc.ClearMessages(ctx, req.UserId, req.TargetId, int32(req.ConvType))
	if err != nil {
		return &pb.ClearMessagesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.ClearMessagesResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

// 添加 CreateConversation 方法
func (s *MessageServiceService) CreateConversation(ctx context.Context, req *pb.CreateConversationReq) (*pb.CreateConversationResp, error) {
	// 构建Command
	cmd := &biz.CreateConversationCommand{
		UserID:         req.UserId,
		TargetID:       req.TargetId,
		ConvType:       int32(req.ConvType),
		InitialMessage: req.InitialMessage,
	}

	// 调用业务层
	result, err := s.uc.CreateConversation(ctx, cmd)
	if err != nil {
		return &pb.CreateConversationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateConversationResp{
		Meta:           utils.GetSuccessMeta(),
		ConversationId: result.ConversationID,
	}, nil
}
