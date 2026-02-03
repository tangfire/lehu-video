package service

import (
	"context"
	"errors"
	"github.com/spf13/cast"
	"time"

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

	// 使用 cast.ToInt64 转换 string ID
	senderID := cast.ToInt64(req.SenderId)
	receiverID := cast.ToInt64(req.ReceiverId)
	conversationID := cast.ToInt64(req.ConversationId)

	if senderID == 0 || receiverID == 0 {
		return &pb.SendMessageResp{
			Meta: utils.GetMetaWithError(errors.New("invalid sender or receiver")),
		}, nil
	}

	cmd := &biz.SendMessageCommand{
		SenderID:       senderID,
		ReceiverID:     receiverID,
		ConversationID: conversationID,
		ConvType:       int32(req.ConvType),
		MsgType:        int32(req.MsgType),
		ClientMsgID:    req.ClientMsgId,
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
		MessageId:      cast.ToString(result.MessageID),
		ConversationId: cast.ToString(result.ConversationID),
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
		UserID:         cast.ToInt64(req.UserId),
		ConversationID: cast.ToInt64(req.ConversationId),
		LastMsgID:      cast.ToInt64(req.LastMsgId),
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
			Id:             cast.ToString(msg.ID),             // 转 string
			ConversationId: cast.ToString(msg.ConversationID), // 转 string
			SenderId:       cast.ToString(msg.SenderID),       // 转 string
			ReceiverId:     cast.ToString(msg.ReceiverID),     // 转 string
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
		LastMsgId: cast.ToString(result.LastMsgID),
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
		UserID: cast.ToInt64(req.UserId),
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
			Id:          cast.ToString(conv.ID),
			Type:        pb.ConversationType(conv.Type),
			GroupId:     cast.ToString(conv.GroupID),
			Name:        conv.Name,
			Avatar:      conv.Avatar,
			LastMessage: conv.LastMessage,
			LastMsgType: (*pb.MessageType)(conv.LastMsgType),
			LastMsgTime: &[]int64{conv.LastMsgTime.Unix()}[0], // 简洁写法,
			MemberCount: conv.MemberCount,
			CreatedAt:   conv.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   conv.UpdatedAt.Format("2006-01-02 15:04:05"),
			MemberIds:   cast.ToStringSlice(conv.MemberIDs),
		}

		if conv.GroupID != 0 {
			groupID := cast.ToString(conv.GroupID)
			pbConv.GroupId = groupID
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

	messageID := cast.ToInt64(req.MessageId)
	userID := cast.ToInt64(req.UserId)

	if messageID == 0 || userID == 0 {
		return &pb.RecallMessageResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	cmd := &biz.RecallMessageCommand{
		MessageID: messageID, // 转换
		UserID:    userID,    // 转换
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

	userID := cast.ToInt64(req.UserId)
	conversationID := cast.ToInt64(req.ConversationId)

	if userID == 0 || conversationID == 0 {
		return &pb.MarkMessagesReadResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	cmd := &biz.MarkMessagesReadCommand{
		UserID:         userID,                      // 转换
		ConversationID: conversationID,              // 转换
		LastMsgID:      cast.ToInt64(req.LastMsgId), // 转换
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

	userID := cast.ToInt64(req.UserId)
	if userID == 0 {
		return &pb.GetUnreadCountResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	query := &biz.GetUnreadCountQuery{
		UserID: userID, // 转换
	}

	result, err := s.uc.GetUnreadCount(ctx, query)
	if err != nil {
		return &pb.GetUnreadCountResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换conv_unread为proto格式
	convUnread := make(map[string]int64)
	for k, v := range result.ConvUnread {
		convUnread[cast.ToString(k)] = v
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

	userID := cast.ToInt64(req.UserId)
	conversationID := cast.ToInt64(req.ConversationId)

	if userID == 0 || conversationID == 0 {
		return &pb.DeleteConversationResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	cmd := &biz.DeleteConversationCommand{
		UserID:         userID,         // 转换
		ConversationID: conversationID, // 转换
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

func (s *MessageServiceService) GetConversation(
	ctx context.Context,
	req *pb.GetConversationReq,
) (*pb.GetConversationResp, error) {

	conversationID := cast.ToInt64(req.ConversationId)
	userID := cast.ToInt64(req.UserId) // 使用新增的 user_id 参数

	if conversationID == 0 {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(errors.New("conversation_id is required")),
		}, nil
	}

	if userID == 0 {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(errors.New("user_id is required")),
		}, nil
	}

	// 调用 usecase 获取带用户状态的会话视图
	convView, err := s.uc.GetConversationView(ctx, conversationID, userID)
	if err != nil {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if convView == nil {
		return &pb.GetConversationResp{
			Meta: utils.GetMetaWithError(errors.New("会话不存在")),
		}, nil
	}

	// 转换为 proto 结构
	pbConv := &pb.Conversation{
		Id:          cast.ToString(convView.ID),
		Type:        pb.ConversationType(convView.Type),
		Name:        convView.Name,
		Avatar:      convView.Avatar,
		LastMessage: convView.LastMessage,
		MemberCount: convView.MemberCount,
		CreatedAt:   convView.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   convView.UpdatedAt.Format(time.RFC3339),
	}

	// 处理可选字段
	if convView.GroupID != 0 {
		groupIDStr := cast.ToString(convView.GroupID)
		pbConv.GroupId = groupIDStr
	}
	if convView.LastMsgType != nil {
		msgType := pb.MessageType(*convView.LastMsgType)
		pbConv.LastMsgType = &msgType
	}
	if convView.LastMsgTime != nil {
		timestamp := convView.LastMsgTime.Unix()
		pbConv.LastMsgTime = &timestamp
	}

	// 添加成员ID列表
	if len(convView.MemberIDs) > 0 {
		pbConv.MemberIds = cast.ToStringSlice(convView.MemberIDs)
	}

	// 构建 ConversationView
	conversationView := &pb.ConversationView{
		Conversation: pbConv,
		UnreadCount:  convView.UnreadCount,
		IsPinned:     convView.IsPinned,
		IsMuted:      convView.IsMuted,
	}

	return &pb.GetConversationResp{
		Meta:             utils.GetSuccessMeta(),
		ConversationView: conversationView, // 返回 ConversationView
	}, nil
}

// ClearMessages 清空聊天记录
func (s *MessageServiceService) ClearMessages(
	ctx context.Context,
	req *pb.ClearMessagesReq,
) (*pb.ClearMessagesResp, error) {

	userID := cast.ToInt64(req.UserId)
	conversationID := cast.ToInt64(req.ConversationId)

	if userID == 0 || conversationID == 0 {
		return &pb.ClearMessagesResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	// 调用修复后的业务逻辑
	cmd := &biz.ClearMessagesCommand{
		UserID:         userID,
		ConversationID: conversationID,
	}

	_, err := s.uc.ClearMessages(ctx, cmd)
	if err != nil {
		return &pb.ClearMessagesResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.ClearMessagesResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

// UpdateMessageStatus 更新消息状态
func (s *MessageServiceService) UpdateMessageStatus(
	ctx context.Context,
	req *pb.UpdateMessageStatusReq,
) (*pb.UpdateMessageStatusResp, error) {

	messageID := cast.ToInt64(req.MessageId)
	operatorID := cast.ToInt64(req.OperatorId)

	if messageID == 0 {
		return &pb.UpdateMessageStatusResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	cmd := &biz.UpdateMessageStatusCommand{
		MessageID:  messageID,
		Status:     int32(req.Status),
		OperatorID: operatorID,
	}

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

// GetMessage 获取单条消息
func (s *MessageServiceService) GetMessage(
	ctx context.Context,
	req *pb.GetMessageReq,
) (*pb.GetMessageResp, error) {

	messageID := cast.ToInt64(req.MessageId)
	if messageID == 0 {
		return &pb.GetMessageResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	message, err := s.uc.GetMessage(ctx, messageID)
	if err != nil {
		return &pb.GetMessageResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	if message == nil {
		return &pb.GetMessageResp{
			Meta: utils.GetMetaWithError(errors.New("消息不存在")),
		}, nil
	}

	pbMsg := &pb.Message{
		Id:             cast.ToString(message.ID),
		ConversationId: cast.ToString(message.ConversationID),
		SenderId:       cast.ToString(message.SenderID),
		ReceiverId:     cast.ToString(message.ReceiverID),
		ConvType:       pb.ConversationType(message.ConvType),
		MsgType:        pb.MessageType(message.MsgType),
		Status:         pb.MessageStatus(message.Status),
		IsRecalled:     message.IsRecalled,
		CreatedAt:      message.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      message.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if message.Content != nil {
		pbMsg.Content = &pb.MessageContent{
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
		}
	}

	return &pb.GetMessageResp{
		Meta:    utils.GetSuccessMeta(),
		Message: pbMsg,
	}, nil
}

// CreateConversation 创建会话
func (s *MessageServiceService) CreateConversation(
	ctx context.Context,
	req *pb.CreateConversationReq,
) (*pb.CreateConversationResp, error) {

	userIDs := cast.ToInt64Slice(req.UserIds)
	GroupID := cast.ToInt64(req.GroupId)

	if len(userIDs) <= 1 {
		return &pb.CreateConversationResp{
			Meta: utils.GetMetaWithError(errors.New("参数错误")),
		}, nil
	}

	cmd := &biz.CreateConversationCommand{
		UserIDs:        userIDs,
		GroupID:        GroupID,
		ConvType:       int32(req.ConvType),
		InitialMessage: req.InitialMessage,
	}

	result, err := s.uc.CreateConversation(ctx, cmd)
	if err != nil {
		return &pb.CreateConversationResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.CreateConversationResp{
		Meta:           utils.GetSuccessMeta(),
		ConversationId: cast.ToString(result.ConversationID),
	}, nil
}
