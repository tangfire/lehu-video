package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// CreateMessage 创建消息
func (r *messageRepo) CreateMessage(ctx context.Context, message *biz.Message) error {
	// 序列化消息内容
	contentJSON, err := json.Marshal(message.Content)
	if err != nil {
		return err
	}

	dbMessage := model.Message{
		ID:             message.ID,
		SenderID:       message.SenderID,
		ReceiverID:     message.ReceiverID,
		ConversationID: message.ConversationID,
		ConvType:       int8(message.ConvType),
		MsgType:        int8(message.MsgType),
		Content:        contentJSON,
		Status:         int8(message.Status),
		IsRecalled:     message.IsRecalled,
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
		IsDeleted:      message.IsDeleted,
	}

	return r.data.db.WithContext(ctx).Create(&dbMessage).Error
}

// GetMessageByID 根据ID获取消息
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

	return r.toBizMessage(&dbMessage), nil
}

// CountTotalUnread 统计总未读数
func (r *messageRepo) CountTotalUnread(ctx context.Context, userID int64) (int64, error) {
	var count int64

	// 只统计单聊未读消息
	err := r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Joins("INNER JOIN conversation c ON message.conversation_id = c.id").
		Where("c.type = ? AND message.receiver_id = ? AND message.status < ? AND message.is_recalled = ? AND message.is_deleted = ?",
			0, userID, 3, false, false).
		Count(&count).Error

	return count, err
}

// toBizMessage 将数据库消息转换为业务消息
func (r *messageRepo) toBizMessage(dbMessage *model.Message) *biz.Message {
	// 反序列化消息内容
	var content biz.MessageContent
	if len(dbMessage.Content) > 0 {
		if err := json.Unmarshal(dbMessage.Content, &content); err != nil {
			r.log.Errorf("反序列化消息内容失败: %v", err)
		}
	}

	return &biz.Message{
		ID:             dbMessage.ID,
		SenderID:       dbMessage.SenderID,
		ReceiverID:     dbMessage.ReceiverID,
		ConversationID: dbMessage.ConversationID,
		ConvType:       int32(dbMessage.ConvType),
		MsgType:        int32(dbMessage.MsgType),
		Content:        &content,
		Status:         int32(dbMessage.Status),
		IsRecalled:     dbMessage.IsRecalled,
		CreatedAt:      dbMessage.CreatedAt,
		UpdatedAt:      dbMessage.UpdatedAt,
		IsDeleted:      dbMessage.IsDeleted,
	}
}

// ListMessages 查询消息列表（按会话）
func (r *messageRepo) ListMessages(ctx context.Context, conversationID, lastMsgID int64, limit int) ([]*biz.Message, bool, error) {
	var dbMessages []*model.Message

	// 构建查询
	query := r.data.db.WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false)

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

	// 转换为业务对象
	messages := make([]*biz.Message, 0, len(dbMessages))
	for _, dbMessage := range dbMessages {
		messages = append(messages, r.toBizMessage(dbMessage))
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

// MarkMessagesAsRead 标记消息为已读
func (r *messageRepo) MarkMessagesAsRead(ctx context.Context, conversationID, userID, lastMsgID int64) error {
	// 获取会话类型
	var convType int8
	err := r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Select("type").
		Where("id = ?", conversationID).
		Scan(&convType).Error

	if err != nil {
		return err
	}

	// 只有单聊才需要标记消息已读，群聊不需要
	if convType != 0 {
		return nil
	}

	// 获取最后一条已读消息的时间
	var lastMessage model.Message
	err = r.data.db.WithContext(ctx).
		Where("id = ?", lastMsgID).
		First(&lastMessage).Error
	if err != nil {
		return err
	}

	// 更新该时间之前的所有对方发送的消息为已读
	return r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND status < ? AND created_at <= ? AND is_deleted = ?",
			conversationID, userID, 3, lastMessage.CreatedAt, false).
		Updates(map[string]interface{}{
			"status":     3, // 已读
			"updated_at": time.Now(),
		}).Error
}

// UpdateMessageStatus 更新消息状态
func (r *messageRepo) UpdateMessageStatus(ctx context.Context, id int64, status int32) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     int8(status),
			"updated_at": time.Now(),
		}).Error
}

// RecallMessage 撤回消息
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

func (r *messageRepo) DeleteConversation(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
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
func (r *messageRepo) DeleteMessagesByConversation(ctx context.Context, userID, conversationID, targetID int64, convType int32) error {
	// 构建查询条件
	query := r.data.db.WithContext(ctx).Model(&model.Message{})

	if convType == 0 { // 单聊
		// 单聊：删除用户发送给对方的和对方发送给用户的所有消息
		query = query.Where("conversation_id = ? AND ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND is_deleted = ?",
			conversationID,
			userID, targetID, // 用户发送给对方的
			targetID, userID, // 对方发送给用户的
			false)
	} else if convType == 1 { // 群聊
		// 群聊：只删除用户在该群中发送的消息
		// 注意：群聊的 receiver_id 是群ID
		query = query.Where("conversation_id = ? AND sender_id = ? AND receiver_id = ? AND is_deleted = ?",
			conversationID,
			userID,
			targetID, // 群ID
			false)
	} else {
		return fmt.Errorf("不支持的会话类型")
	}

	// 软删除消息
	return query.Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": time.Now(),
	}).Error
}
