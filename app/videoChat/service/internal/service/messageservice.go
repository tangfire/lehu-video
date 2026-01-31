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

// =======================
// SendMessage
// =======================

func (s *MessageServiceService) SendMessage(
	ctx context.Context,
	req *pb.SendMessageReq,
) (*pb.SendMessageResp, error) {

	if req.SenderId == 0 || req.ReceiverId == 0 {
		return &pb.SendMessageResp{
			Meta: utils.GetMetaWithError(errors.New("invalid sender or receiver")),
		}, nil
	}

	cmd := &biz.SendMessageCommand{
		SenderID:    req.SenderId,
		ReceiverID:  req.ReceiverId,
		ConvType:    int32(req.ConvType),
		MsgType:     int32(req.MsgType),
		ClientMsgID: req.ClientMsgId,
	}

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

	result, err := s.uc.SendMessage(ctx, cmd)
	if err != nil {
		return &pb.SendMessageResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.SendMessageResp{
		Meta:           utils.GetSuccessMeta(),
		MessageId:      result.MessageID,
		ConversationId: result.ConversationID,
	}, nil
}

// =======================
// ListMessages
// =======================

func (s *MessageServiceService) ListMessages(
	ctx context.Context,
	req *pb.ListMessagesReq,
) (*pb.ListMessagesResp, error) {

	query := &biz.ListMessagesQuery{
		UserID:         req.UserId,
		ConversationID: req.ConversationId,
		LastMsgID:      req.LastMsgId,
		Limit:          int(req.Limit),
	}

	result, err := s.uc.ListMessages(ctx, query)
	if err != nil {
		return &pb.ListMessagesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	messages := make([]*pb.Message, 0, len(result.Messages))
	for _, msg := range result.Messages {
		pbMsg := &pb.Message{
			Id:             msg.ID,
			ConversationId: msg.ConversationID,
			SenderId:       msg.SenderID,
			ReceiverId:     msg.ReceiverID,
			ConvType:       pb.ConversationType(msg.ConvType),
			MsgType:        pb.MessageType(msg.MsgType),
			Status:         pb.MessageStatus(msg.Status),
			IsRecalled:     msg.IsRecalled,
			CreatedAt:      msg.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      msg.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		if msg.Content != nil {
			pbMsg.Content = &pb.MessageContent{
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
			}
		}

		messages = append(messages, pbMsg)
	}

	return &pb.ListMessagesResp{
		Meta:      utils.GetSuccessMeta(),
		Messages:  messages,
		HasMore:   result.HasMore,
		LastMsgId: result.LastMsgID,
	}, nil
}

// =======================
// ListConversations（重点修复）
// =======================

func (s *MessageServiceService) ListConversations(
	ctx context.Context,
	req *pb.ListConversationsReq,
) (*pb.ListConversationsResp, error) {

	query := &biz.ListConversationsQuery{
		UserID: req.UserId,
		PageStats: &biz.PageStats{
			Page:     int(req.PageStats.Page),
			PageSize: int(req.PageStats.Size),
		},
	}

	result, err := s.uc.ListConversations(ctx, query)
	if err != nil {
		return &pb.ListConversationsResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	views := make([]*pb.ConversationView, 0, len(result.Conversations))
	for _, conv := range result.Conversations {

		pbConv := &pb.Conversation{
			Id:          conv.ID,
			Type:        pb.ConversationType(conv.Type),
			Name:        conv.Name,
			Avatar:      conv.Avatar,
			LastMessage: conv.LastMessage,
			MemberCount: conv.MemberCount,
			UpdatedAt:   conv.UpdatedAt.Format("2006-01-02 15:04:05"),
			CreatedAt:   conv.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if conv.TargetID != nil {
			pbConv.TargetId = conv.TargetID
		}
		if conv.GroupID != nil {
			pbConv.GroupId = conv.GroupID
		}
		if conv.LastMsgType != nil {
			v := pb.MessageType(*conv.LastMsgType)
			pbConv.LastMsgType = &v
		}
		if conv.LastMsgTime != nil {
			t := conv.LastMsgTime.Unix()
			pbConv.LastMsgTime = &t
		}

		view := &pb.ConversationView{
			Conversation: pbConv,
			UnreadCount:  conv.UnreadCount,
			IsPinned:     conv.IsPinned,
			IsMuted:      conv.IsMuted,
		}

		views = append(views, view)
	}

	return &pb.ListConversationsResp{
		Meta:          utils.GetSuccessMeta(),
		Conversations: views,
		PageStats: &pb.PageStatsResp{
			Total: int32(result.Total),
		},
	}, nil
}

// RecallMessage 撤回消息
func (s *MessageServiceService) RecallMessage(
	ctx context.Context,
	req *pb.RecallMessageReq,
) (*pb.RecallMessageResp, error) {

	if req.MessageId == 0 || req.UserId == 0 {
		return &pb.RecallMessageResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

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

// MarkMessagesRead 标记消息已读
func (s *MessageServiceService) MarkMessagesRead(
	ctx context.Context,
	req *pb.MarkMessagesReadReq,
) (*pb.MarkMessagesReadResp, error) {

	if req.UserId == 0 || req.ConversationId == 0 {
		return &pb.MarkMessagesReadResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	cmd := &biz.MarkMessagesReadCommand{
		UserID:         req.UserId,
		ConversationID: req.ConversationId,
		LastMsgID:      req.LastMsgId,
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

// GetUnreadCount 获取未读消息数
func (s *MessageServiceService) GetUnreadCount(
	ctx context.Context,
	req *pb.GetUnreadCountReq,
) (*pb.GetUnreadCountResp, error) {

	if req.UserId == 0 {
		return &pb.GetUnreadCountResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	query := &biz.GetUnreadCountQuery{
		UserID: req.UserId,
	}

	result, err := s.uc.GetUnreadCount(ctx, query)
	if err != nil {
		return &pb.GetUnreadCountResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换conv_unread为proto格式
	convUnread := make(map[int64]int64)
	for k, v := range result.ConvUnread {
		convUnread[k] = v
	}

	return &pb.GetUnreadCountResp{
		Meta:        utils.GetSuccessMeta(),
		TotalUnread: result.TotalUnread,
		ConvUnread:  convUnread,
	}, nil
}

// DeleteConversation 删除会话
func (s *MessageServiceService) DeleteConversation(
	ctx context.Context,
	req *pb.DeleteConversationReq,
) (*pb.DeleteConversationResp, error) {

	if req.UserId == 0 || req.ConversationId == 0 {
		return &pb.DeleteConversationResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

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

// GetConversation 获取会话详情
func (s *MessageServiceService) GetConversation(
	ctx context.Context,
	req *pb.GetConversationReq,
) (*pb.GetConversationResp, error) {

	if req.UserId == 0 || req.TargetId == 0 {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

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

	// 转换为proto结构
	pbConv := &pb.Conversation{
		Id:          conversation.ID,
		Type:        pb.ConversationType(conversation.Type),
		Name:        conversation.Name,
		Avatar:      conversation.Avatar,
		LastMessage: conversation.LastMessage,
		MemberCount: conversation.MemberCount,
		UpdatedAt:   conversation.UpdatedAt.Format("2006-01-02 15:04:05"),
		CreatedAt:   conversation.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if conversation.TargetID != nil {
		pbConv.TargetId = conversation.TargetID
	}
	if conversation.GroupID != nil {
		pbConv.GroupId = conversation.GroupID
	}
	if conversation.LastMsgType != nil {
		v := pb.MessageType(*conversation.LastMsgType)
		pbConv.LastMsgType = &v
	}
	if conversation.LastMsgTime != nil {
		t := conversation.LastMsgTime.Unix()
		pbConv.LastMsgTime = &t
	}

	return &pb.GetConversationResp{
		Meta:         utils.GetSuccessMeta(),
		Conversation: pbConv,
	}, nil
}

// ClearMessages 清空聊天记录
func (s *MessageServiceService) ClearMessages(
	ctx context.Context,
	req *pb.ClearMessagesReq,
) (*pb.ClearMessagesResp, error) {

	if req.UserId == 0 || req.ConversationId == 0 {
		return &pb.ClearMessagesResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	// 获取会话信息
	// 这里需要先获取会话，然后根据会话类型调用不同的方法
	// 由于ClearMessagesReq只有conversation_id，我们需要先获取会话信息
	// 为了简化，我们可以添加一个新的UseCase方法
	// 或者修改proto，添加target_id和conv_type参数

	// 暂时返回未实现
	return &pb.ClearMessagesResp{
		Meta: utils.GetMetaWithError(errors.New("暂未实现")),
	}, nil
}
