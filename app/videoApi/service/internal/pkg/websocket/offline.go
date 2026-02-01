package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/biz"
)

// OfflineMessage 离线消息结构
type OfflineMessage struct {
	MessageID  string          `json:"message_id"`
	SenderID   string          `json:"sender_id"`
	ReceiverID string          `json:"receiver_id"`
	ConvType   int32           `json:"conv_type"`
	MsgType    int32           `json:"msg_type"`
	Content    json.RawMessage `json:"content"`
	CreatedAt  time.Time       `json:"created_at"`
}

// OfflineManager 离线消息管理器
type OfflineManager struct {
	sync.RWMutex
	messages  map[string][]*OfflineMessage // userID -> 离线消息列表
	logger    log.Logger
	messageUC *biz.MessageUsecase
	chat      biz.ChatAdapter
}

// NewOfflineManager 创建离线消息管理器
func NewOfflineManager(logger log.Logger, messageUC *biz.MessageUsecase, chat biz.ChatAdapter) *OfflineManager {
	return &OfflineManager{
		messages:  make(map[string][]*OfflineMessage),
		logger:    logger,
		messageUC: messageUC,
		chat:      chat,
	}
}

// StoreOfflineMessage 存储离线消息
func (om *OfflineManager) StoreOfflineMessage(userID string, message *OfflineMessage) {
	om.Lock()
	defer om.Unlock()

	if om.messages[userID] == nil {
		om.messages[userID] = make([]*OfflineMessage, 0)
	}

	om.messages[userID] = append(om.messages[userID], message)
	om.logger.Log(log.LevelInfo, "msg", "存储离线消息", "user_id", userID, "message_id", message.MessageID)
}

// GetOfflineMessages 获取用户的所有离线消息
func (om *OfflineManager) GetOfflineMessages(userID string) []*OfflineMessage {
	om.RLock()
	defer om.RUnlock()

	return om.messages[userID]
}

// ClearOfflineMessages 清除用户的离线消息
func (om *OfflineManager) ClearOfflineMessages(userID string) {
	om.Lock()
	defer om.Unlock()

	delete(om.messages, userID)
	om.logger.Log(log.LevelInfo, "msg", "清除离线消息", "user_id", userID)
}

// DeliverOfflineMessages 投递离线消息
func (om *OfflineManager) DeliverOfflineMessages(userID string, client *Client) {
	om.RLock()
	messages := om.messages[userID]
	om.RUnlock()

	if len(messages) == 0 {
		return
	}

	om.logger.Log(log.LevelInfo, "msg", "开始投递离线消息", "user_id", userID, "count", len(messages))

	for _, msg := range messages {
		// 构建推送消息
		pushMsg := om.buildPushMessage(msg)

		// 发送给客户端
		select {
		case client.SendChan <- pushMsg:
			om.logger.Log(log.LevelDebug, "msg", "投递离线消息成功", "message_id", msg.MessageID)

			// 更新消息状态为已送达
			go om.updateMessageStatus(msg.MessageID, 2) // 2 = DELIVERED
		default:
			om.logger.Log(log.LevelWarn, "msg", "客户端发送通道已满，离线消息投递失败", "message_id", msg.MessageID)
		}

		// 短暂延迟，避免消息风暴
		time.Sleep(50 * time.Millisecond)
	}

	// 清除已投递的离线消息
	om.ClearOfflineMessages(userID)
}

// buildPushMessage 构建推送消息
func (om *OfflineManager) buildPushMessage(msg *OfflineMessage) []byte {
	response := WSMessageResponse{
		Action:    "receive_message",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message_id":  msg.MessageID,
			"sender_id":   msg.SenderID,
			"receiver_id": msg.ReceiverID,
			"conv_type":   msg.ConvType,
			"msg_type":    msg.MsgType,
			"content":     json.RawMessage(msg.Content),
			"is_offline":  true,
			"timestamp":   msg.CreatedAt.Unix(),
		},
	}

	respBytes, _ := json.Marshal(response)
	return respBytes
}

// updateMessageStatus 更新消息状态
func (om *OfflineManager) updateMessageStatus(messageID string, status int32) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := om.chat.UpdateMessageStatus(ctx, messageID, status)
	if err != nil {
		om.logger.Log(log.LevelError, "msg", "更新消息状态失败", "message_id", messageID, "error", err)
	}
}
