package service

import (
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/kafka"
	"lehu-video/app/videoApi/service/internal/pkg/websocket"
)

type WebSocketService struct {
	wsManager *websocket.Manager
	wsHandler *websocket.Handler
	logger    log.Logger
}

// NewWebSocketService 需要传入 kafkaProducer 和 redisClient
func NewWebSocketService(
	messageUC *biz.MessageUsecase,
	chat biz.ChatAdapter,
	kafkaProducer *kafka.Producer,
	redisClient *redis.Client,
	logger log.Logger,
) *WebSocketService {
	// 创建WebSocket管理器，传入必要的依赖
	wsManager := websocket.NewManager(logger, kafkaProducer, redisClient)

	// 创建WebSocket处理器
	wsHandler := websocket.NewHandler(wsManager, messageUC, chat, logger, "fireshine")

	// 启动WebSocket管理器
	go wsManager.Start()

	return &WebSocketService{
		wsManager: wsManager,
		wsHandler: wsHandler,
		logger:    logger,
	}
}

func (s *WebSocketService) GetHandler() http.Handler {
	return s.wsHandler
}

// BroadcastToUser 向指定用户发送消息
func (s *WebSocketService) BroadcastToUser(userID string, message []byte) {
	s.wsManager.BroadcastToUser(userID, message)
}

// BroadcastToGroup 向群组所有成员发送消息
func (s *WebSocketService) BroadcastToGroup(userIDs []string, message []byte) {
	s.wsManager.BroadcastToGroup(userIDs, message)
}

// IsUserOnline 检查用户是否在线
func (s *WebSocketService) IsUserOnline(userID string) bool {
	return s.wsManager.IsUserOnline(userID)
}

// BatchCheckOnline 批量检查用户在线状态
func (s *WebSocketService) BatchCheckOnline(userIDs []string) map[string]bool {
	return s.wsManager.BatchCheckOnline(userIDs)
}
