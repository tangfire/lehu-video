package service

import (
	"context"

	pb "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/pkg/utils"
)

type MessageServiceService struct {
	pb.UnimplementedMessageServiceServer

	uc *biz.MessageUsecase
}

func NewMessageServiceService(uc *biz.MessageUsecase) *MessageServiceService {
	return &MessageServiceService{uc: uc}
}

func (s *MessageServiceService) SendMessage(ctx context.Context, req *pb.SendMessageReq) (*pb.SendMessageResp, error) {
	cmd := &biz.SendMessageCommand{
		SenderID:   req.SenderId,
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

	result, err := s.uc.SendMessage(ctx, cmd)
	if err != nil {
		return &pb.SendMessageResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.SendMessageResp{
		MessageId: result.MessageID,
		Meta:      utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) ListMessages(ctx context.Context, req *pb.ListMessagesReq) (*pb.ListMessagesResp, error) {
	query := &biz.ListMessagesQuery{
		UserID:    req.UserId,
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
		Limit:     int(req.Limit),
	}

	result, err := s.uc.ListMessages(ctx, query)
	if err != nil {
		return &pb.ListMessagesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var messages []*pb.Message
	for _, m := range result.Messages {
		messages = append(messages, &pb.Message{
			Id:         m.ID,
			SenderId:   m.SenderID,
			ReceiverId: m.ReceiverID,
			ConvType:   pb.ConversationType(m.ConvType),
			MsgType:    pb.MessageType(m.MsgType),
			Content: &pb.MessageContent{
				Text:          m.Content.Text,
				ImageUrl:      m.Content.ImageURL,
				ImageWidth:    m.Content.ImageWidth,
				ImageHeight:   m.Content.ImageHeight,
				VoiceUrl:      m.Content.VoiceURL,
				VoiceDuration: m.Content.VoiceDuration,
				VideoUrl:      m.Content.VideoURL,
				VideoCover:    m.Content.VideoCover,
				VideoDuration: m.Content.VideoDuration,
				FileUrl:       m.Content.FileURL,
				FileName:      m.Content.FileName,
				FileSize:      m.Content.FileSize,
				Extra:         m.Content.Extra,
			},
			Status:     pb.MessageStatus(m.Status),
			IsRecalled: m.IsRecalled,
			CreatedAt:  m.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  m.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListMessagesResp{
		Messages:  messages,
		HasMore:   result.HasMore,
		LastMsgId: result.LastMsgID,
		Meta:      utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) RecallMessage(ctx context.Context, req *pb.RecallMessageReq) (*pb.RecallMessageResp, error) {
	cmd := &biz.RecallMessageCommand{
		MessageID: req.MessageId,
		UserID:    req.UserId,
	}

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
	cmd := &biz.MarkMessagesReadCommand{
		UserID:    req.UserId,
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
	}

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

func (s *MessageServiceService) ListConversations(ctx context.Context, req *pb.ListConversationsReq) (*pb.ListConversationsResp, error) {
	query := &biz.ListConversationsQuery{
		UserID: req.UserId,
		PageStats: biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	result, err := s.uc.ListConversations(ctx, query)
	if err != nil {
		return &pb.ListConversationsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	var conversations []*pb.Conversation
	for _, c := range result.Conversations {
		conversations = append(conversations, &pb.Conversation{
			Id:          c.ID,
			Type:        pb.ConversationType(c.Type),
			TargetId:    c.TargetID,
			LastMessage: c.LastMessage,
			LastMsgType: pb.MessageType(c.LastMsgType),
			LastMsgTime: c.LastMsgTime.Unix(),
			UnreadCount: int64(c.UnreadCount),
			UpdatedAt:   c.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListConversationsResp{
		Conversations: conversations,
		Meta:          utils.GetSuccessMeta(),
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

func (s *MessageServiceService) DeleteConversation(ctx context.Context, req *pb.DeleteConversationReq) (*pb.DeleteConversationResp, error) {
	cmd := &biz.DeleteConversationCommand{
		UserID:         req.UserId,
		ConversationID: req.ConversationId,
	}

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

func (s *MessageServiceService) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountReq) (*pb.GetUnreadCountResp, error) {
	// TODO: 实现获取未读消息数量
	return &pb.GetUnreadCountResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

func (s *MessageServiceService) ClearMessages(ctx context.Context, req *pb.ClearMessagesReq) (*pb.ClearMessagesResp, error) {
	// TODO: 实现清空聊天记录
	return &pb.ClearMessagesResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}
