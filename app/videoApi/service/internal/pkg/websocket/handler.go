package websocket

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/gorilla/websocket"

	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	sharedauth "lehu-video/pkg/auth"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// Handler HTTP处理器
type Handler struct {
	manager   *Manager
	messageUC *biz.MessageUsecase
	chat      biz.ChatAdapter
	logger    log.Logger
	jwtSecret string
}

func NewHandler(
	manager *Manager,
	messageUC *biz.MessageUsecase,
	chat biz.ChatAdapter,
	logger log.Logger,
	jwtSecret string,
) *Handler {
	return &Handler{
		manager:   manager,
		messageUC: messageUC,
		chat:      chat,
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

// ServeHTTP 处理 WebSocket 升级请求
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}
	}
	if tokenStr == "" {
		h.logger.Log(log.LevelWarn, "msg", "未授权的访问: 缺失 token")
		http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
		return
	}

	claim, err := parseJWT(tokenStr, h.jwtSecret)
	if err != nil {
		h.logger.Log(log.LevelError, "msg", "Token 验证失败", "err", err)
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Log(log.LevelError, "msg", "WebSocket 升级失败", "err", err)
		return
	}

	ctx := context.Background()
	ctx = kjwt.NewContext(ctx, claim)

	client := NewClient(ctx, claim.UserId, conn, h.manager, h.messageUC, h.chat, h.logger)

	// 异步更新最后上线时间
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := h.manager.updateUserLastOnlineTime(updateCtx, claim.UserId); err != nil {
			h.logger.Log(log.LevelWarn, "msg", "更新最后上线时间失败", "err", err)
		}
	}()

	h.manager.register <- client

	go client.WritePump()
	client.ReadPump()
}

func parseJWT(tokenStr string, secret string) (*claims.Claims, error) {
	claim, err := sharedauth.ParseToken(tokenStr, secret)
	if err != nil {
		return nil, err
	}
	return claim, nil
}
