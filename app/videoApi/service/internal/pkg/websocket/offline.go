package websocket

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// OfflineMessage 离线消息结构
type OfflineMessage struct {
	MessageID      string          `json:"message_id"`
	SenderID       string          `json:"sender_id"`
	ReceiverID     string          `json:"receiver_id"`
	ConversationID string          `json:"conversation_id"`
	ConvType       int32           `json:"conv_type"`
	MsgType        int32           `json:"msg_type"`
	Content        json.RawMessage `json:"content"`
	CreatedAt      time.Time       `json:"created_at"`
}

// OfflineManager 离线消息管理器（Redis版）
type OfflineManager struct {
	redisClient *redis.Client
	logger      log.Logger
}

func NewOfflineManager(redisClient *redis.Client, logger log.Logger) *OfflineManager {
	return &OfflineManager{
		redisClient: redisClient,
		logger:      logger,
	}
}

// StoreOfflineMessage 存储离线消息到Redis
func (om *OfflineManager) StoreOfflineMessage(userID string, message *OfflineMessage) {
	key := "offline:" + userID
	data, err := json.Marshal(message)
	if err != nil {
		om.logger.Log(log.LevelError, "msg", "序列化离线消息失败", "error", err)
		return
	}
	err = om.redisClient.RPush(context.Background(), key, data).Err()
	if err != nil {
		om.logger.Log(log.LevelError, "msg", "存储离线消息到Redis失败", "error", err)
		return
	}
	// 设置过期时间7天
	om.redisClient.Expire(context.Background(), key, 7*24*time.Hour)
	om.logger.Log(log.LevelInfo, "msg", "存储离线消息", "user_id", userID, "message_id", message.MessageID)
}

// DeliverOfflineMessages 投递离线消息（上线时调用）
func (om *OfflineManager) DeliverOfflineMessages(ctx context.Context, userID string, sendFunc func([]byte)) error {
	key := "offline:" + userID
	for {
		// 每次取一条，避免一次性取出太多内存压力
		result, err := om.redisClient.LPop(ctx, key).Bytes()
		if err == redis.Nil {
			break // 队列为空
		}
		if err != nil {
			om.logger.Log(log.LevelError, "msg", "从Redis取离线消息失败", "error", err)
			return err
		}

		var msg OfflineMessage
		if err := json.Unmarshal(result, &msg); err != nil {
			om.logger.Log(log.LevelError, "msg", "解析离线消息失败", "error", err)
			continue
		}

		// 构建推送消息
		pushMsg := om.buildPushMessage(&msg)
		sendFunc(pushMsg)
	}
	return nil
}

// buildPushMessage 构建推送消息
func (om *OfflineManager) buildPushMessage(msg *OfflineMessage) []byte {
	response := WSMessageResponse{
		Action:    "receive_message",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message_id":      msg.MessageID,
			"sender_id":       msg.SenderID,
			"receiver_id":     msg.ReceiverID,
			"conversation_id": msg.ConversationID,
			"conv_type":       msg.ConvType,
			"msg_type":        msg.MsgType,
			"content":         msg.Content,
			"is_offline":      true,
			"timestamp":       msg.CreatedAt.Unix(),
		},
	}
	respBytes, _ := json.Marshal(response)
	return respBytes
}
