package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/data/model"
)

type messageRepo struct {
	data *Data
	log  *log.Helper
}

func NewMessageRepo(data *Data, logger log.Logger) biz.MessageRepo {
	return &messageRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *messageRepo) CreateMessage(ctx context.Context, message *biz.Message) error {
	// 序列化消息内容
	contentJSON, err := json.Marshal(message.Content)
	if err != nil {
		return err
	}

	dbMessage := model.Message{
		ID:         message.ID,
		SenderID:   message.SenderID,
		ReceiverID: message.ReceiverID,
		ConvType:   int8(message.ConvType),
		MsgType:    int8(message.MsgType),
		Content:    contentJSON,
		Status:     int8(message.Status),
		IsRecalled: message.IsRecalled,
		CreatedAt:  message.CreatedAt,
		UpdatedAt:  message.UpdatedAt,
		IsDeleted:  false,
	}

	return r.data.db.WithContext(ctx).Create(&dbMessage).Error
}

func (r *messageRepo) GetMessageByID(ctx context.Context, id int64) (*biz.Message, error) {
	var dbMessage model.Message
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbMessage).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 反序列化消息内容
	var content biz.MessageContent
	if len(dbMessage.Content) > 0 {
		err = json.Unmarshal(dbMessage.Content, &content)
		if err != nil {
			return nil, err
		}
	}

	return &biz.Message{
		ID:         dbMessage.ID,
		SenderID:   dbMessage.SenderID,
		ReceiverID: dbMessage.ReceiverID,
		ConvType:   int32(dbMessage.ConvType),
		MsgType:    int32(dbMessage.MsgType),
		Content:    &content,
		Status:     int32(dbMessage.Status),
		IsRecalled: dbMessage.IsRecalled,
		CreatedAt:  dbMessage.CreatedAt,
		UpdatedAt:  dbMessage.UpdatedAt,
	}, nil
}

func (r *messageRepo) UpdateMessageStatus(ctx context.Context, id int64, status int32) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *messageRepo) RecallMessage(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_recalled": true,
			"status":      4, // 已撤回状态
			"updated_at":  time.Now(),
		}).Error
}

func (r *messageRepo) ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int) ([]*biz.Message, error) {
	var dbMessages []*model.Message

	query := r.data.db.WithContext(ctx).
		Where("((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND conv_type = ? AND is_deleted = ?",
			userID, targetID, targetID, userID, convType, false)

	if lastMsgID > 0 {
		query = query.Where("id < ?", lastMsgID)
	}

	err := query.Order("id DESC").
		Limit(limit).
		Find(&dbMessages).Error

	if err != nil {
		return nil, err
	}

	// 反转顺序，使消息按时间正序排列
	for i, j := 0, len(dbMessages)-1; i < j; i, j = i+1, j-1 {
		dbMessages[i], dbMessages[j] = dbMessages[j], dbMessages[i]
	}

	messages := make([]*biz.Message, 0, len(dbMessages))
	for _, dbMessage := range dbMessages {
		// 反序列化消息内容
		var content biz.MessageContent
		if len(dbMessage.Content) > 0 {
			err = json.Unmarshal(dbMessage.Content, &content)
			if err != nil {
				return nil, err
			}
		}

		messages = append(messages, &biz.Message{
			ID:         dbMessage.ID,
			SenderID:   dbMessage.SenderID,
			ReceiverID: dbMessage.ReceiverID,
			ConvType:   int32(dbMessage.ConvType),
			MsgType:    int32(dbMessage.MsgType),
			Content:    &content,
			Status:     int32(dbMessage.Status),
			IsRecalled: dbMessage.IsRecalled,
			CreatedAt:  dbMessage.CreatedAt,
			UpdatedAt:  dbMessage.UpdatedAt,
		})
	}

	return messages, nil
}

func (r *messageRepo) CountUnreadMessages(ctx context.Context, userID, targetID int64, convType int32) (int64, error) {
	var count int64

	// 查询对方发送的未读消息
	query := r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND conv_type = ? AND status < ? AND is_recalled = ? AND is_deleted = ?",
			targetID, userID, convType, 3, false, false)

	// 对于群聊，还需要考虑已读记录
	if convType == 1 { // 群聊
		// TODO: 需要结合message_read表来统计
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *messageRepo) MarkMessagesAsRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error {
	// 更新消息状态
	err := r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND conv_type = ? AND status < ? AND id <= ? AND is_deleted = ?",
			targetID, userID, convType, 3, lastMsgID, false).
		Updates(map[string]interface{}{
			"status":     3, // 已读
			"updated_at": time.Now(),
		}).Error

	if err != nil {
		return err
	}

	// 创建已读记录
	if lastMsgID > 0 {
		// TODO: 为每条消息创建已读记录
	}

	return nil
}

func (r *messageRepo) CreateOrUpdateConversation(ctx context.Context, conv *biz.Conversation) error {
	// 先尝试查询现有会话
	var existingConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND target_id = ? AND is_deleted = ?",
			conv.UserID, conv.Type, conv.TargetID, false).
		First(&existingConv).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新会话
		dbConv := model.Conversation{
			UserID:      conv.UserID,
			Type:        int8(conv.Type),
			TargetID:    conv.TargetID,
			LastMessage: conv.LastMessage,
			LastMsgType: int8(conv.LastMsgType),
			LastMsgTime: conv.LastMsgTime,
			UnreadCount: conv.UnreadCount,
			IsPinned:    conv.IsPinned,
			IsMuted:     conv.IsMuted,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			IsDeleted:   false,
		}
		return r.data.db.WithContext(ctx).Create(&dbConv).Error
	}

	if err != nil {
		return err
	}

	// 更新现有会话
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", existingConv.ID).
		Updates(map[string]interface{}{
			"last_message":  conv.LastMessage,
			"last_msg_type": conv.LastMsgType,
			"last_msg_time": conv.LastMsgTime,
			"unread_count":  gorm.Expr("unread_count + ?", conv.UnreadCount),
			"updated_at":    time.Now(),
		}).Error
}

func (r *messageRepo) GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND target_id = ? AND is_deleted = ?",
			userID, convType, targetID, false).
		First(&dbConv).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.Conversation{
		ID:          dbConv.ID,
		UserID:      dbConv.UserID,
		Type:        int32(dbConv.Type),
		TargetID:    dbConv.TargetID,
		LastMessage: dbConv.LastMessage,
		LastMsgType: int32(dbConv.LastMsgType),
		LastMsgTime: dbConv.LastMsgTime,
		UnreadCount: dbConv.UnreadCount,
		IsPinned:    dbConv.IsPinned,
		IsMuted:     dbConv.IsMuted,
		CreatedAt:   dbConv.CreatedAt,
		UpdatedAt:   dbConv.UpdatedAt,
	}, nil
}

func (r *messageRepo) GetConversationByID(ctx context.Context, id int64) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbConv).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.Conversation{
		ID:          dbConv.ID,
		UserID:      dbConv.UserID,
		Type:        int32(dbConv.Type),
		TargetID:    dbConv.TargetID,
		LastMessage: dbConv.LastMessage,
		LastMsgType: int32(dbConv.LastMsgType),
		LastMsgTime: dbConv.LastMsgTime,
		UnreadCount: dbConv.UnreadCount,
		IsPinned:    dbConv.IsPinned,
		IsMuted:     dbConv.IsMuted,
		CreatedAt:   dbConv.CreatedAt,
		UpdatedAt:   dbConv.UpdatedAt,
	}, nil
}

func (r *messageRepo) DeleteConversation(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

func (r *messageRepo) ListConversations(ctx context.Context, userID int64, offset, limit int) ([]*biz.Conversation, error) {
	var dbConvs []*model.Conversation

	query := r.data.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("is_pinned DESC, last_msg_time DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbConvs).Error
	if err != nil {
		return nil, err
	}

	convs := make([]*biz.Conversation, 0, len(dbConvs))
	for _, dbConv := range dbConvs {
		convs = append(convs, &biz.Conversation{
			ID:          dbConv.ID,
			UserID:      dbConv.UserID,
			Type:        int32(dbConv.Type),
			TargetID:    dbConv.TargetID,
			LastMessage: dbConv.LastMessage,
			LastMsgType: int32(dbConv.LastMsgType),
			LastMsgTime: dbConv.LastMsgTime,
			UnreadCount: dbConv.UnreadCount,
			IsPinned:    dbConv.IsPinned,
			IsMuted:     dbConv.IsMuted,
			CreatedAt:   dbConv.CreatedAt,
			UpdatedAt:   dbConv.UpdatedAt,
		})
	}

	return convs, nil
}

func (r *messageRepo) CountConversations(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&count).Error

	return count, err
}

func (r *messageRepo) CreateMessageRead(ctx context.Context, messageID, userID int64) error {
	readRecord := model.MessageRead{
		MessageID: messageID,
		UserID:    userID,
		ReadAt:    time.Now(),
		CreatedAt: time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(&readRecord).Error
}

func (r *messageRepo) GetUserMessageSetting(ctx context.Context, userID, targetID int64, convType int32) (*biz.UserMessageSetting, error) {
	var dbSetting model.UserMessageSetting
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND target_id = ? AND conv_type = ? AND is_deleted = ?",
			userID, targetID, convType, false).
		First(&dbSetting).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.UserMessageSetting{
		ID:        dbSetting.ID,
		UserID:    dbSetting.UserID,
		TargetID:  dbSetting.TargetID,
		ConvType:  int32(dbSetting.ConvType),
		IsMuted:   dbSetting.IsMuted,
		CreatedAt: dbSetting.CreatedAt,
		UpdatedAt: dbSetting.UpdatedAt,
	}, nil
}

func (r *messageRepo) UpdateUserMessageSetting(ctx context.Context, setting *biz.UserMessageSetting) error {
	var dbSetting model.UserMessageSetting
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND target_id = ? AND conv_type = ? AND is_deleted = ?",
			setting.UserID, setting.TargetID, setting.ConvType, false).
		First(&dbSetting).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新设置
		dbSetting = model.UserMessageSetting{
			UserID:    setting.UserID,
			TargetID:  setting.TargetID,
			ConvType:  int8(setting.ConvType),
			IsMuted:   setting.IsMuted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsDeleted: false,
		}
		return r.data.db.WithContext(ctx).Create(&dbSetting).Error
	}

	if err != nil {
		return err
	}

	// 更新现有设置
	return r.data.db.WithContext(ctx).
		Model(&model.UserMessageSetting{}).
		Where("id = ?", dbSetting.ID).
		Updates(map[string]interface{}{
			"is_muted":   setting.IsMuted,
			"updated_at": time.Now(),
		}).Error
}
