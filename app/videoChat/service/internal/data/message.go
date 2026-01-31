package data

import (
	"context"
	"encoding/json"
	"errors"
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

	if errors.Is(err, gorm.ErrRecordNotFound) {
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
			"status":     int8(status),
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

// ListMessages 改进的分页查询方法，按时间戳分页
func (r *messageRepo) ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int) ([]*biz.Message, bool, error) {
	var dbMessages []*model.Message

	// 构建基础查询
	query := r.data.db.WithContext(ctx).
		Where("((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND conv_type = ? AND is_deleted = ?",
			userID, targetID, targetID, userID, int8(convType), false)

	// 如果提供了lastMsgID，先查询该消息的时间戳
	if lastMsgID > 0 {
		var lastMessage model.Message
		err := r.data.db.WithContext(ctx).
			Where("id = ?", lastMsgID).
			First(&lastMessage).Error
		if err == nil {
			// 使用时间戳进行分页
			query = query.Where("created_at < ?", lastMessage.CreatedAt)
		}
	}

	// 获取limit+1条记录来判断是否还有更多
	err := query.Order("created_at DESC").
		Limit(limit + 1). // 多取一条用于判断
		Find(&dbMessages).Error

	if err != nil {
		return nil, false, err
	}

	// 判断是否还有更多数据
	hasMore := false
	if len(dbMessages) > limit {
		hasMore = true
		dbMessages = dbMessages[:limit] // 只保留limit条
	}

	// 反转顺序，使消息按时间正序排列（从旧到新）
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
				return nil, false, err
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

	return messages, hasMore, nil
}

func (r *messageRepo) CountUnreadMessages(ctx context.Context, userID, targetID int64, convType int32) (int64, error) {
	var count int64

	// ✅ 单聊：查询对方发送的未读消息（status < 3）
	// ✅ 群聊：不查询未读消息数，返回0（群聊不显示已读状态）
	if convType == 0 {
		query := r.data.db.WithContext(ctx).
			Model(&model.Message{}).
			Where("sender_id = ? AND receiver_id = ? AND conv_type = ? AND status < ? AND is_recalled = ? AND is_deleted = ?",
				targetID, userID, 0, 3, false, false)

		err := query.Count(&count).Error
		return count, err
	}

	// 群聊：返回0，不统计未读
	return 0, nil
}

func (r *messageRepo) MarkMessagesAsRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error {
	// 只有单聊才需要标记已读
	if convType != 0 {
		// 群聊：什么都不做，直接返回成功
		return nil
	}

	// 单聊：标记消息为已读
	tx := r.data.db.WithContext(ctx).Begin()

	// 如果有lastMsgID，标记该消息及之前的消息为已读
	if lastMsgID > 0 {
		// 获取最后一条已读消息的时间
		var lastMessage model.Message
		err := tx.Where("id = ?", lastMsgID).First(&lastMessage).Error
		if err == nil {
			// 更新该时间之前的所有消息为已读
			err = tx.Model(&model.Message{}).
				Where("sender_id = ? AND receiver_id = ? AND conv_type = ? AND status < ? AND created_at <= ? AND is_deleted = ?",
					targetID, userID, 0, 3, lastMessage.CreatedAt, false).
				Updates(map[string]interface{}{
					"status":     3, // 已读
					"updated_at": time.Now(),
				}).Error
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	} else {
		// 标记所有消息为已读
		err := tx.Model(&model.Message{}).
			Where("sender_id = ? AND receiver_id = ? AND conv_type = ? AND status < ? AND is_deleted = ?",
				targetID, userID, 0, 3, false).
			Updates(map[string]interface{}{
				"status":     3,
				"updated_at": time.Now(),
			}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *messageRepo) CreateOrUpdateConversation(ctx context.Context, conv *biz.Conversation) error {
	// 先尝试查询现有会话
	var existingConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND target_id = ? AND is_deleted = ?",
			conv.UserID, int8(conv.Type), conv.TargetID, false).
		First(&existingConv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新会话
		dbConv := model.Conversation{
			ID:          conv.ID,
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

		// 检查ID是否已存在（防止冲突）
		var count int64
		r.data.db.WithContext(ctx).Model(&model.Conversation{}).
			Where("id = ?", conv.ID).Count(&count)
		if count > 0 {
			// ID冲突，使用数据库自增ID
			return r.data.db.WithContext(ctx).Create(&dbConv).Error
		}

		return r.data.db.WithContext(ctx).Create(&dbConv).Error
	}

	if err != nil {
		return err
	}

	// 更新现有会话
	updateData := map[string]interface{}{
		"last_message":  conv.LastMessage,
		"last_msg_type": conv.LastMsgType,
		"last_msg_time": conv.LastMsgTime,
		"updated_at":    time.Now(),
	}

	// 处理未读计数：如果是更新会话，通常不需要累加，直接设置
	if conv.UnreadCount >= 0 {
		updateData["unread_count"] = gorm.Expr("unread_count + ?", conv.UnreadCount)
	} else {
		// 如果是清除未读
		updateData["unread_count"] = 0
	}

	// 更新其他字段
	if conv.IsPinned != existingConv.IsPinned {
		updateData["is_pinned"] = conv.IsPinned
	}
	if conv.IsMuted != existingConv.IsMuted {
		updateData["is_muted"] = conv.IsMuted
	}

	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", existingConv.ID).
		Updates(updateData).Error
}

// GetConversationByUniqueKey 新增：通过唯一键查询会话
func (r *messageRepo) GetConversationByUniqueKey(ctx context.Context, userID, targetID int64, convType int32) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND target_id = ? AND type = ? AND is_deleted = ?",
			userID, targetID, int8(convType), false).
		First(&dbConv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
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

// GetConversationByID 新增：通过ID查询会话
func (r *messageRepo) GetConversationByID(ctx context.Context, conversationID int64) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", conversationID, false).
		First(&dbConv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
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

// ResetConversationUnreadCount 重置会话未读计数
func (r *messageRepo) ResetConversationUnreadCount(ctx context.Context, conversationID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("unread_count", 0).Error
}

// 批量更新消息状态
func (r *messageRepo) BatchUpdateMessageStatus(ctx context.Context, messageIDs []int64, status int32) error {
	if len(messageIDs) == 0 {
		return nil
	}

	return r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id IN ?", messageIDs).
		Updates(map[string]interface{}{
			"status":     int8(status),
			"updated_at": time.Now(),
		}).Error
}

func (r *messageRepo) GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND target_id = ? AND is_deleted = ?",
			userID, int8(convType), targetID, false).
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
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
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

func (r *messageRepo) CountTotalUnread(ctx context.Context, userID int64) (int64, error) {
	var count int64

	// ✅ 只统计单聊未读消息
	err := r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("receiver_id = ? AND conv_type = ? AND status < ? AND is_recalled = ? AND is_deleted = ?",
			userID, 0, 3, false, false).
		Count(&count).Error

	return count, err
}

func (r *messageRepo) CountUnreadByConversations(ctx context.Context, userID int64) (map[int64]int64, error) {
	result := make(map[int64]int64)

	// ✅ 只统计单聊未读，群聊返回0
	type SingleChatResult struct {
		TargetID int64
		Count    int64
	}

	var singleResults []SingleChatResult
	err := r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Select("sender_id as target_id, COUNT(*) as count").
		Where("receiver_id = ? AND conv_type = ? AND status < ? AND is_recalled = ? AND is_deleted = ?",
			userID, 0, 3, false, false).
		Group("sender_id").
		Find(&singleResults).Error

	if err != nil {
		return nil, err
	}

	// 单聊：key = targetID * 10 + 0 (convType=0)
	for _, r := range singleResults {
		key := r.TargetID*10 + 0
		result[key] = r.Count
	}

	// 群聊：直接返回0，不需要统计
	return result, nil
}

// 清空聊天记录
func (r *messageRepo) DeleteMessagesByConversation(ctx context.Context, userID, targetID int64, convType int32) error {
	// 使用软删除，只标记为删除状态，不物理删除
	// 注意：这里只删除用户接收的消息，发送的消息不删除（保留发送记录）

	return r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("receiver_id = ? AND ((sender_id = ? AND conv_type = ?) OR (receiver_id = ? AND conv_type = ?)) AND is_deleted = ?",
			userID, targetID, convType, targetID, convType, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}
