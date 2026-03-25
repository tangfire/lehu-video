package service

import (
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	core "lehu-video/api/videoCore/service/v1"
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
	userClient core.UserServiceClient,
	logger log.Logger,
) *WebSocketService {
	// 创建 WebSocket 管理器，传入必要的依赖
	wsManager := websocket.NewManager(logger, kafkaProducer, redisClient, chat, userClient)

	// 创建 WebSocket 处理器
	wsHandler := websocket.NewHandler(wsManager, messageUC, chat, logger, "fireshine")

	// 启动 WebSocket 管理器
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
