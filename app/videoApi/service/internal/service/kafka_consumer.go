package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	segkafka "github.com/segmentio/kafka-go" // 给 segmentio/kafka-go 起别名

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
		s.consumer.CommitMessages(ctx, msg)
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
			s.consumer.CommitMessages(ctx, msg)
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

	output, err := s.messageUC.SendMessage(ctx, input)
	if err != nil {
		s.log.Errorf("发送消息失败: %v", err)
		if clientMsgID != "" {
			s.redisClient.Del(ctx, "idempotent:"+clientMsgID)
		}
		return
	}

	if err := s.consumer.CommitMessages(ctx, msg); err != nil {
		s.log.Errorf("提交偏移量失败: %v", err)
	}

	s.pushStatusToSender(ctx, output, data)
	s.pushToReceiver(ctx, output, data)
}

func (s *KafkaConsumerService) pushStatusToSender(ctx context.Context, output *biz.SendMessageOutput, data map[string]interface{}) {
	senderID, _ := data["sender_id"].(string)
	clientMsgID, _ := data["client_msg_id"].(string)
	if senderID == "" {
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
	s.wsManager.BroadcastToUser(senderID, respBytes)
}

func (s *KafkaConsumerService) pushToReceiver(ctx context.Context, output *biz.SendMessageOutput, data map[string]interface{}) {
	receiverID, _ := data["receiver_id"].(string)
	convTypeFloat, _ := data["conv_type"].(float64)
	convType := int32(convTypeFloat)
	senderID, _ := data["sender_id"].(string)

	pushMsg := s.buildPushMessage(output.MessageID, output.ConversationId, data)

	if convType == 0 {
		s.wsManager.BroadcastToUser(receiverID, pushMsg)
	} else if convType == 1 {
		groupID := receiverID
		members, err := s.chat.GetGroupMembers(ctx, groupID)
		if err != nil {
			s.log.Errorf("获取群成员失败: %v", err)
			return
		}
		for _, memberID := range members {
			if memberID == senderID {
				continue
			}
			if s.wsManager.IsUserOnline(memberID) {
				s.wsManager.BroadcastToUser(memberID, pushMsg)
			} else {
				s.storeOfflineMessage(memberID, output.MessageID, data)
			}
		}
	}
}

func (s *KafkaConsumerService) storeOfflineMessage(userID, messageID string, data map[string]interface{}) {
	offlineKey := "offline:" + userID
	msgData, _ := json.Marshal(data)
	err := s.redisClient.RPush(context.Background(), offlineKey, msgData).Err()
	if err != nil {
		s.log.Errorf("存储离线消息失败: user=%s, err=%v", userID, err)
	}
	s.redisClient.Expire(context.Background(), offlineKey, 7*24*time.Hour)
}

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

// Close 关闭消费者，释放资源
func (s *KafkaConsumerService) Close() error {
	return s.consumer.Close()
}
