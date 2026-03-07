package websocket

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
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

	// 更新持久化在线状态（数据库）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := h.chat.UpdateUserOnlineStatus(ctx, claim.UserId, 1, "web")
		if err != nil {
			h.logger.Log(log.LevelError, "msg", "更新持久化在线状态失败", "err", err)
		}
	}()

	h.manager.register <- client

	go client.WritePump()
	client.ReadPump()
}

func parseJWT(tokenStr string, secret string) (*claims.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	if c, ok := token.Claims.(*claims.Claims); ok {
		return c, nil
	}
	return nil, errors.New("invalid claims type")
}
