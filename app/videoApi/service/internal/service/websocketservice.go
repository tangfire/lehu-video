package service

import (
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/websocket"
)

type WebSocketService struct {
	wsManager *websocket.Manager
	wsHandler *websocket.Handler
	logger    log.Logger
}

func NewWebSocketService(messageUC *biz.MessageUsecase, logger log.Logger) *WebSocketService {
	// 创建WebSocket管理器
	wsManager := websocket.NewManager(logger)

	// 创建WebSocket处理器
	wsHandler := websocket.NewHandler(wsManager, messageUC, logger)

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
func (s *WebSocketService) BroadcastToUser(userID int64, message []byte) {
	s.wsManager.BroadcastToUser(userID, message)
}

// BroadcastToGroup 向群组所有成员发送消息
func (s *WebSocketService) BroadcastToGroup(userIDs []int64, message []byte) {
	s.wsManager.BroadcastToGroup(userIDs, message)
}
