package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/biz"

	pb "lehu-video/api/videoApi/service/v1"
)

type MessageServiceService struct {
	pb.UnimplementedMessageServiceServer

	log *log.Helper
	uc  *biz.MessageUsecase
}

func NewMessageServiceService(uc *biz.MessageUsecase, logger log.Logger) *MessageServiceService {
	return &MessageServiceService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *MessageServiceService) SendMessage(ctx context.Context, req *pb.SendMessageReq) (*pb.SendMessageResp, error) {
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
		return nil, err
	}

	return &pb.SendMessageResp{
		MessageId: output.MessageID,
	}, nil
}

func (s *MessageServiceService) ListMessages(ctx context.Context, req *pb.ListMessagesReq) (*pb.ListMessagesResp, error) {
	input := &biz.ListMessagesInput{
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
		Limit:     req.Limit,
	}

	output, err := s.uc.ListMessages(ctx, input)
	if err != nil {
		return nil, err
	}

	var messages []*pb.Message
	for _, m := range output.Messages {
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
		HasMore:   output.HasMore,
		LastMsgId: output.LastMsgID,
	}, nil
}

func (s *MessageServiceService) RecallMessage(ctx context.Context, req *pb.RecallMessageReq) (*pb.RecallMessageResp, error) {
	input := &biz.RecallMessageInput{
		MessageID: req.MessageId,
	}

	err := s.uc.RecallMessage(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.RecallMessageResp{}, nil
}

func (s *MessageServiceService) MarkMessagesRead(ctx context.Context, req *pb.MarkMessagesReadReq) (*pb.MarkMessagesReadResp, error) {
	input := &biz.MarkMessagesReadInput{
		TargetID:  req.TargetId,
		ConvType:  int32(req.ConvType),
		LastMsgID: req.LastMsgId,
	}

	err := s.uc.MarkMessagesRead(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.MarkMessagesReadResp{}, nil
}

func (s *MessageServiceService) ListConversations(ctx context.Context, req *pb.ListConversationsReq) (*pb.ListConversationsResp, error) {
	input := &biz.ListConversationsInput{
		PageStats: &biz.PageStats{
			Page:     req.PageStats.Page,
			PageSize: req.PageStats.Size,
		},
	}

	output, err := s.uc.ListConversations(ctx, input)
	if err != nil {
		return nil, err
	}

	var conversations []*pb.Conversation
	for _, c := range output.Conversations {
		conversations = append(conversations, &pb.Conversation{
			Id:          c.ID,
			Type:        pb.ConversationType(c.Type),
			TargetId:    c.TargetID,
			LastMessage: c.LastMessage,
			LastMsgType: pb.MessageType(c.LastMsgType),
			LastMsgTime: c.LastMsgTime,
			UnreadCount: c.UnreadCount,
			UpdatedAt:   c.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListConversationsResp{
		Conversations: conversations,
		PageStats: &pb.PageStatsResp{
			Total: int32(output.Total),
		},
	}, nil
}

func (s *MessageServiceService) DeleteConversation(ctx context.Context, req *pb.DeleteConversationReq) (*pb.DeleteConversationResp, error) {
	input := &biz.DeleteConversationInput{
		ConversationID: req.ConversationId,
	}

	err := s.uc.DeleteConversation(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteConversationResp{}, nil
}
