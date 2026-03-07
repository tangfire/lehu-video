package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	segkafka "github.com/segmentio/kafka-go"

	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/kafka"
	"lehu-video/app/videoApi/service/internal/pkg/websocket"
)

type KafkaConsumerService struct {
	consumer    *kafka.Consumer
	messageUC   *biz.MessageUsecase
	wsManager   *websocket.Manager
	redisClient *redis.Client
	chat        biz.ChatAdapter
	log         *log.Helper
}

func NewKafkaConsumerService(
	consumer *kafka.Consumer,
	messageUC *biz.MessageUsecase,
	wsManager *websocket.Manager,
	redisClient *redis.Client,
	chat biz.ChatAdapter,
	logger log.Logger,
) *KafkaConsumerService {
	return &KafkaConsumerService{
		consumer:    consumer,
		messageUC:   messageUC,
		wsManager:   wsManager,
		redisClient: redisClient,
		chat:        chat,
		log:         log.NewHelper(logger),
	}
}

// Run 启动消费者循环
func (s *KafkaConsumerService) Run(ctx context.Context) error {
	s.log.Info("Kafka消费者启动")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := s.consumer.ReadMessage(ctx)
			if err != nil {
				s.log.Errorf("读取Kafka消息失败: %v", err)
				time.Sleep(time.Second)
				continue
			}
			s.processMessage(ctx, msg)
		}
	}
}

// processMessage 处理单条消息
func (s *KafkaConsumerService) processMessage(ctx context.Context, msg segkafka.Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Value, &data); err != nil {
		s.log.Errorf("解析消息失败: %v", err)
		_ = s.consumer.CommitMessages(ctx, msg)
		return
	}

	clientMsgID, _ := data["client_msg_id"].(string)

	// 幂等校验
	if clientMsgID != "" {
		ok, err := s.redisClient.SetNX(ctx, "idempotent:"+clientMsgID, "1", 24*time.Hour).Result()
		if err != nil {
			s.log.Errorf("幂等校验失败: %v", err)
			return
		}
		if !ok {
			s.log.Infof("重复消息，已忽略: client_msg_id=%s", clientMsgID)
			_ = s.consumer.CommitMessages(ctx, msg)
			return
		}
	}

	senderID, _ := data["sender_id"].(string)
	receiverID, _ := data["receiver_id"].(string)
	conversationID, _ := data["conversation_id"].(string)
	convType, _ := data["conv_type"].(float64)
	msgType, _ := data["msg_type"].(float64)
	contentMap, _ := data["content"].(map[string]interface{})

	content := &biz.MessageContent{
		Text:          getString(contentMap, "text"),
		ImageURL:      getString(contentMap, "image_url"),
		ImageWidth:    getInt64(contentMap, "image_width"),
		ImageHeight:   getInt64(contentMap, "image_height"),
		VoiceURL:      getString(contentMap, "voice_url"),
		VoiceDuration: getInt64(contentMap, "voice_duration"),
		VideoURL:      getString(contentMap, "video_url"),
		VideoCover:    getString(contentMap, "video_cover"),
		VideoDuration: getInt64(contentMap, "video_duration"),
		FileURL:       getString(contentMap, "file_url"),
		FileName:      getString(contentMap, "file_name"),
		FileSize:      getInt64(contentMap, "file_size"),
		Extra:         getString(contentMap, "extra"),
	}

	input := &biz.SendMessageInput{
		SenderID:       senderID,
		ConversationID: conversationID,
		ReceiverID:     receiverID,
		ConvType:       int32(convType),
		MsgType:        int32(msgType),
		Content:        content,
		ClientMsgID:    clientMsgID,
	}

	// 调用业务层发送消息
	output, err := s.messageUC.SendMessage(ctx, input)
	if err != nil {
		s.log.Errorf("发送消息失败: %v", err)
		// 永久错误提交offset
		if isPermanentError(err) {
			_ = s.consumer.CommitMessages(ctx, msg)
		}
		return
	}

	// 提交offset
	if err := s.consumer.CommitMessages(ctx, msg); err != nil {
		s.log.Errorf("提交偏移量失败: %v", err)
	}

	// 推送状态给发送者
	s.pushStatusToSender(ctx, output, clientMsgID)
	// 推送给接收者（或存储离线）
	s.pushToReceiver(ctx, output, data)
}

// 判断是否为永久性错误
func isPermanentError(err error) bool {
	return false // 暂不实现
}

// pushStatusToSender 向发送者推送最终消息状态
func (s *KafkaConsumerService) pushStatusToSender(ctx context.Context, output *biz.SendMessageOutput, clientMsgID string) {
	if clientMsgID == "" {
		return
	}
	statusMsg := websocket.WSMessageResponse{
		Action:    "message_status",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"client_msg_id":   clientMsgID,
			"message_id":      output.MessageID,
			"conversation_id": output.ConversationId,
			"status":          1,
		},
	}
	respBytes, _ := json.Marshal(statusMsg)
	s.wsManager.BroadcastToUser(output.SenderID, respBytes)
}

// pushToReceiver 向接收者推送消息或存储离线
func (s *KafkaConsumerService) pushToReceiver(ctx context.Context, output *biz.SendMessageOutput, data map[string]interface{}) {
	receiverID, _ := data["receiver_id"].(string)
	convTypeFloat, _ := data["conv_type"].(float64)
	convType := int32(convTypeFloat)
	senderID, _ := data["sender_id"].(string)

	s.log.Infof("推送消息给接收者: receiver=%s, messageID=%s", receiverID, output.MessageID)

	// 构建推送消息（使用最终的消息ID）
	pushMsg := s.buildPushMessage(output.MessageID, output.ConversationId, data)

	if convType == 0 { // 单聊
		online := s.wsManager.IsUserOnline(receiverID)
		s.log.Infof("接收者在线状态: user=%s, online=%v", receiverID, online)
		if online {
			s.log.Infof("用户在线，直接推送: %s", receiverID)
			s.wsManager.BroadcastToUser(receiverID, pushMsg)
			s.updateMessageStatus(output.MessageID, 2)
		} else {
			s.log.Infof("用户离线，存储离线: %s", receiverID)
			s.storeOfflineMessage(receiverID, output.MessageID, data)
		}
	} else if convType == 1 { // 群聊
		groupID := receiverID
		members, err := s.chat.GetGroupMembers(ctx, groupID)
		if err != nil {
			s.log.Errorf("获取群成员失败: %v", err)
			return
		}
		s.log.Infof("群成员数量: %d", len(members))
		for _, memberID := range members {
			if memberID == senderID {
				continue
			}
			online := s.wsManager.IsUserOnline(memberID)
			s.log.Infof("群成员在线状态: user=%s, online=%v", memberID, online)
			if online {
				s.log.Infof("群成员在线，推送: %s", memberID)
				s.wsManager.BroadcastToUser(memberID, pushMsg)
			} else {
				s.log.Infof("群成员离线，存储离线: %s", memberID)
				s.storeOfflineMessage(memberID, output.MessageID, data)
			}
		}
	}
}

// updateMessageStatus 更新消息状态
func (s *KafkaConsumerService) updateMessageStatus(messageID string, status int32) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.messageUC.UpdateMessageStatus(ctx, messageID, status)
	if err != nil {
		s.log.Errorf("更新消息状态失败: %v", err)
	}
}

// storeOfflineMessage 存储离线消息到Redis
func (s *KafkaConsumerService) storeOfflineMessage(userID, messageID string, data map[string]interface{}) {
	offlineKey := "offline:" + userID
	// 构造要存储的离线消息（使用 time.Time 类型）
	offlineData := map[string]interface{}{
		"message_id":      messageID,
		"sender_id":       data["sender_id"],
		"receiver_id":     data["receiver_id"],
		"conversation_id": data["conversation_id"],
		"conv_type":       data["conv_type"],
		"msg_type":        data["msg_type"],
		"content":         data["content"],
		"created_at":      time.Now(), // 改为 time.Time 类型，Marshal 后为 RFC3339
	}
	msgData, _ := json.Marshal(offlineData)
	err := s.redisClient.RPush(context.Background(), offlineKey, msgData).Err()
	if err != nil {
		s.log.Errorf("存储离线消息失败: user=%s, err=%v", userID, err)
	} else {
		s.redisClient.Expire(context.Background(), offlineKey, 7*24*time.Hour)
	}
}

// buildPushMessage 构建推送给接收者的消息
func (s *KafkaConsumerService) buildPushMessage(messageID, conversationID string, data map[string]interface{}) []byte {
	response := websocket.WSMessageResponse{
		Action:    "receive_message",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"id":              messageID,
			"message_id":      messageID,
			"sender_id":       data["sender_id"],
			"receiver_id":     data["receiver_id"],
			"conversation_id": conversationID,
			"conv_type":       data["conv_type"],
			"msg_type":        data["msg_type"],
			"content":         data["content"],
			"timestamp":       time.Now().Unix(),
			"status":          1,
			"is_recalled":     false,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}
	respBytes, _ := json.Marshal(response)
	return respBytes
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

// Close 关闭消费者
func (s *KafkaConsumerService) Close() error {
	return s.consumer.Close()
}
