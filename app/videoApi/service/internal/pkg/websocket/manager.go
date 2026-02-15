package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"

	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/kafka"
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

// Client 表示一个WebSocket客户端连接
type Client struct {
	ID        int64
	UserID    string
	Ctx       context.Context
	Conn      *websocket.Conn
	SendChan  chan []byte
	Manager   *Manager
	logger    log.Logger
	messageUC *biz.MessageUsecase
	chat      biz.ChatAdapter
}

// Manager 管理所有WebSocket连接
type Manager struct {
	sync.RWMutex
	clients       map[string]*Client
	broadcast     chan *BroadcastMessage
	register      chan *Client
	unregister    chan *Client
	logger        log.Logger
	onlineMgr     *OnlineManager
	offlineMgr    *OfflineManager
	kafkaProducer *kafka.Producer
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	UserIDs []string
	Message []byte
	IsGroup bool
	GroupID string
}

// 获取用户连接数
func (m *Manager) GetUserConnectionCount(userID string) int {
	m.RLock()
	defer m.RUnlock()

	count := 0
	for _, client := range m.clients {
		if client.UserID == userID {
			count++
		}
	}
	return count
}

// NewManager 创建新的连接管理器
func NewManager(
	logger log.Logger,
	kafkaProducer *kafka.Producer,
	redisClient *redis.Client,
) *Manager {
	return &Manager{
		clients:       make(map[string]*Client),
		broadcast:     make(chan *BroadcastMessage, 256),
		register:      make(chan *Client, 256),
		unregister:    make(chan *Client, 256),
		logger:        logger,
		onlineMgr:     NewOnlineManager(),
		offlineMgr:    NewOfflineManager(redisClient, logger),
		kafkaProducer: kafkaProducer,
	}
}

// Start 启动管理器
func (m *Manager) Start() {
	go m.startCleanupTask()

	for {
		select {
		case client := <-m.register:
			m.Lock()

			if old, ok := m.clients[client.UserID]; ok {
				go func(old *Client) {
					time.Sleep(500 * time.Millisecond)
					old.Conn.WriteMessage(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(
							websocket.CloseNormalClosure,
							"replaced by new connection",
						),
					)
					old.Conn.Close()
				}(old)
			}

			m.clients[client.UserID] = client
			m.onlineMgr.SetUserOnline(client.UserID, "web", client.Conn.RemoteAddr().String())
			m.Unlock()
			m.logger.Log(log.LevelInfo, "msg", "用户已连接", "user_id", client.UserID)

			// 投递离线消息：启动 goroutine 避免阻塞
			go func(c *Client) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				// 定义发送函数，将消息写入 client.SendChan
				sendFunc := func(msg []byte) {
					select {
					case c.SendChan <- msg:
					default:
						m.logger.Log(log.LevelWarn, "msg", "离线消息投递失败，发送通道已满", "user_id", c.UserID)
					}
				}
				err := m.offlineMgr.DeliverOfflineMessages(ctx, c.UserID, sendFunc)
				if err != nil {
					m.logger.Log(log.LevelError, "msg", "投递离线消息失败", "user_id", c.UserID, "err", err)
				}
			}(client)

		case client := <-m.unregister:
			m.Lock()
			if _, ok := m.clients[client.UserID]; ok {
				delete(m.clients, client.UserID)
				m.onlineMgr.SetUserOffline(client.UserID, client.Conn.RemoteAddr().String())
			}
			m.Unlock()
			m.logger.Log(log.LevelInfo, "msg", "用户已断开连接", "user_id", client.UserID)

		case bm := <-m.broadcast:
			m.RLock()
			if bm.IsGroup {
				for _, userID := range bm.UserIDs {
					if client, ok := m.clients[userID]; ok {
						select {
						case client.SendChan <- bm.Message:
						default:
							m.logger.Log(log.LevelWarn, "msg", "用户发送通道已满", "user_id", userID)
						}
					}
				}
			} else {
				for _, userID := range bm.UserIDs {
					if client, ok := m.clients[userID]; ok {
						select {
						case client.SendChan <- bm.Message:
						default:
							m.logger.Log(log.LevelWarn, "msg", "用户发送通道已满", "user_id", userID)
						}
					}
				}
			}
			m.RUnlock()
		}
	}
}

// startCleanupTask 启动定时清理任务
func (m *Manager) startCleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.onlineMgr.CleanupInactiveUsers(10 * time.Minute)
		m.logger.Log(log.LevelInfo, "msg", "清理不活跃用户连接完成")
	}
}

// BroadcastToUser 向指定用户发送消息
func (m *Manager) BroadcastToUser(userID string, message []byte) {
	m.logger.Log(log.LevelInfo, "msg", "准备向用户推送消息",
		"user_id", userID,
		"message_size", len(message),
		"online", m.IsUserOnline(userID))

	m.broadcast <- &BroadcastMessage{
		UserIDs: []string{userID},
		Message: message,
		IsGroup: false,
	}
}

// BroadcastToGroup 向群组所有成员发送消息
func (m *Manager) BroadcastToGroup(userIDs []string, message []byte) {
	m.logger.Log(log.LevelInfo, "msg", "准备向群组推送消息",
		"user_count", len(userIDs),
		"message_size", len(message))

	m.broadcast <- &BroadcastMessage{
		UserIDs: userIDs,
		Message: message,
		IsGroup: true,
	}
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID string) bool {
	return m.onlineMgr.IsUserOnline(userID)
}

// BatchCheckOnline 批量检查用户在线状态
func (m *Manager) BatchCheckOnline(userIDs []string) map[string]bool {
	return m.onlineMgr.GetOnlineUsers(userIDs)
}

// NewClient 创建新的客户端连接
func NewClient(
	ctx context.Context,
	userID string,
	conn *websocket.Conn,
	manager *Manager,
	messageUC *biz.MessageUsecase,
	chat biz.ChatAdapter,
	logger log.Logger,
) *Client {
	return &Client{
		ID:        time.Now().UnixNano(),
		UserID:    userID,
		Ctx:       ctx,
		Conn:      conn,
		SendChan:  make(chan []byte, 256),
		Manager:   manager,
		logger:    logger,
		messageUC: messageUC,
		chat:      chat,
	}
}

// ReadPump 读取客户端消息
func (c *Client) ReadPump() {
	defer func() {
		// 通知chat服务：用户下线
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.chat.UpdateUserOnlineStatus(ctx, c.UserID, 0, "") // 0 离线
		if err != nil {
			c.logger.Log(log.LevelError, "msg", "更新在线状态失败", "err", err)
		}
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(5120)
	c.Conn.SetPongHandler(func(string) error {
		c.Manager.onlineMgr.UpdateLastActive(c.UserID)
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Log(log.LevelError, "msg", "WebSocket读取错误", "error", err)
			}
			break
		}
		c.handleMessage(message)
	}
}

// WritePump 向客户端发送消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendChan:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.SendChan)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.SendChan)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WSMessageRequest WebSocket消息请求
type WSMessageRequest struct {
	Action      string          `json:"action"`
	Data        json.RawMessage `json:"data"`
	ClientMsgID string          `json:"client_msg_id"`
	Timestamp   int64           `json:"timestamp"`
}

// WSMessageResponse WebSocket消息响应
type WSMessageResponse struct {
	Action      string      `json:"action"`
	Data        interface{} `json:"data,omitempty"`
	ClientMsgID string      `json:"client_msg_id,omitempty"`
	Timestamp   int64       `json:"timestamp"`
	Error       string      `json:"error,omitempty"`
}

// SendMessageReq 发送消息请求结构
type SendMessageReq struct {
	ConversationID string                 `json:"conversation_id"`
	ReceiverID     string                 `json:"receiver_id"`
	ConvType       int32                  `json:"conv_type"`
	MsgType        int32                  `json:"msg_type"`
	Content        map[string]interface{} `json:"content"`
	ClientMsgID    string                 `json:"client_msg_id"`
}

// 辅助函数
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

// handleMessage 处理收到的消息
func (c *Client) handleMessage(message []byte) {
	var req WSMessageRequest
	if err := json.Unmarshal(message, &req); err != nil {
		c.sendError("invalid_message", "消息格式错误", "")
		return
	}

	switch req.Action {
	case "ping":
		c.sendPong()
	case "auth":
		c.handleAuth(req.Data)
	case "send_message":
		c.handleSendMessage(req.Data, req.ClientMsgID)
	case "recall_message":
		c.handleRecallMessage(req.Data)
	case "read_message":
		c.handleReadMessage(req.Data)
	case "typing":
		c.handleTyping(req.Data)
	default:
		c.sendError("unknown_action", "未知的操作类型", req.ClientMsgID)
	}
}

// handleAuth 处理认证
func (c *Client) handleAuth(data json.RawMessage) {
	c.sendResponse("auth_success", map[string]interface{}{
		"user_id":   c.UserID,
		"timestamp": time.Now().Unix(),
	}, "")
}

// handleSendMessage 处理发送消息请求
func (c *Client) handleSendMessage(data json.RawMessage, outerClientMsgID string) {
	var msg SendMessageReq
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析消息数据失败", outerClientMsgID)
		return
	}

	effectiveClientMsgID := msg.ClientMsgID
	if effectiveClientMsgID == "" {
		effectiveClientMsgID = outerClientMsgID
	}

	messageID := generateMessageID()

	content := &biz.MessageContent{
		Text:          getString(msg.Content, "text"),
		ImageURL:      getString(msg.Content, "image_url"),
		ImageWidth:    getInt64(msg.Content, "image_width"),
		ImageHeight:   getInt64(msg.Content, "image_height"),
		VoiceURL:      getString(msg.Content, "voice_url"),
		VoiceDuration: getInt64(msg.Content, "voice_duration"),
		VideoURL:      getString(msg.Content, "video_url"),
		VideoCover:    getString(msg.Content, "video_cover"),
		VideoDuration: getInt64(msg.Content, "video_duration"),
		FileURL:       getString(msg.Content, "file_url"),
		FileName:      getString(msg.Content, "file_name"),
		FileSize:      getInt64(msg.Content, "file_size"),
		Extra:         getString(msg.Content, "extra"),
	}

	messageData := map[string]interface{}{
		"message_id":      messageID,
		"sender_id":       c.UserID,
		"receiver_id":     msg.ReceiverID,
		"conversation_id": msg.ConversationID,
		"conv_type":       msg.ConvType,
		"msg_type":        msg.MsgType,
		"content":         content,
		"client_msg_id":   effectiveClientMsgID,
		"timestamp":       time.Now().Unix(),
	}
	value, _ := json.Marshal(messageData)

	// 发送到 Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := c.Manager.kafkaProducer.SendMessage(ctx, []byte(c.UserID), value)
	if err != nil {
		c.logger.Log(log.LevelError, "msg", "发送消息到Kafka失败", "err", err)
		c.sendError("kafka_failed", "消息发送失败，请重试", effectiveClientMsgID)
		return
	}

	// 给发送者返回 message_sent
	c.sendResponse("message_sent", map[string]interface{}{
		"message_id":      messageID,
		"conversation_id": msg.ConversationID,
		"status":          0,
		"client_msg_id":   effectiveClientMsgID,
	}, effectiveClientMsgID)

	// ========== 新增：立即推送给接收者 ==========
	pushMsg := c.buildReceiveMessage(messageID, msg)

	if msg.ConvType == 0 { // 单聊
		if c.Manager.IsUserOnline(msg.ReceiverID) {
			c.Manager.BroadcastToUser(msg.ReceiverID, pushMsg)
			c.sendDeliveryConfirm(messageID, msg.ReceiverID)
		} else {
			c.storeOfflineMessage(msg.ReceiverID, messageID, msg)
		}
	} else if msg.ConvType == 1 { // 群聊
		c.handleGroupMessage(msg.ReceiverID, messageID, msg, pushMsg)
	}
	// =======================================

	c.logger.Log(log.LevelInfo, "msg", "消息已写入Kafka",
		"message_id", messageID,
		"client_msg_id", effectiveClientMsgID)
}

func generateMessageID() string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(1000))
}

func (c *Client) updateMessageStatus(messageID string, status int32) {
	if c.chat != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.chat.UpdateMessageStatus(ctx, messageID, status)
		if err != nil {
			c.logger.Log(log.LevelError, "msg", "更新消息状态失败",
				"message_id", messageID,
				"status", status,
				"error", err)
		}
	}
}

func (c *Client) handleGroupMessage(groupID, messageID string, msg SendMessageReq, pushMsg []byte) {
	if c.chat != nil {
		gmGetter, ok := c.chat.(interface {
			GetGroupMembers(ctx context.Context, groupID string) ([]string, error)
		})
		if !ok {
			c.logger.Log(log.LevelError, "msg", "chat适配器不支持GetGroupMembers方法")
			return
		}

		members, err := gmGetter.GetGroupMembers(c.Ctx, groupID)
		if err != nil {
			c.logger.Log(log.LevelError, "msg", "获取群成员失败", "group_id", groupID, "error", err)
			return
		}

		onlineCount := 0
		offlineCount := 0

		for _, memberID := range members {
			if memberID == c.UserID {
				continue
			}
			isOnline := c.Manager.IsUserOnline(memberID)
			if isOnline {
				onlineCount++
				c.Manager.BroadcastToUser(memberID, pushMsg)
				c.sendDeliveryConfirm(messageID, memberID)
			} else {
				offlineCount++
				c.storeOfflineMessage(memberID, messageID, msg)
			}
		}

		c.sendResponse("group_message_stat", map[string]interface{}{
			"message_id":    messageID,
			"group_id":      groupID,
			"online_count":  onlineCount,
			"offline_count": offlineCount,
			"total_members": len(members) - 1,
		}, "")
	}
}

func (c *Client) storeOfflineMessage(receiverID, messageID string, msg SendMessageReq) {
	offlineMsg := &OfflineMessage{
		MessageID:      messageID,
		SenderID:       c.UserID,
		ReceiverID:     receiverID,
		ConversationID: msg.ConversationID,
		ConvType:       msg.ConvType,
		MsgType:        msg.MsgType,
		Content:        json.RawMessage{},
		CreatedAt:      time.Now(),
	}
	if contentBytes, err := json.Marshal(msg.Content); err == nil {
		offlineMsg.Content = contentBytes
	}
	c.Manager.offlineMgr.StoreOfflineMessage(receiverID, offlineMsg)
	c.sendOfflineNotification(messageID, msg)
}

func (c *Client) sendDeliveryConfirm(messageID string, receiverID string) {
	if c.chat != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := c.chat.UpdateMessageStatus(ctx, cast.ToString(messageID), 2)
			if err != nil {
				c.logger.Log(log.LevelError, "msg", "更新消息状态失败", "message_id", messageID, "error", err)
			}
		}()
	}

	response := WSMessageResponse{
		Action: "message_delivered",
		Data: map[string]interface{}{
			"message_id":  messageID,
			"receiver_id": receiverID,
			"timestamp":   time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
	respBytes, _ := json.Marshal(response)
	c.SendChan <- respBytes
}

func (c *Client) sendOfflineNotification(messageID string, msg SendMessageReq) {
	response := WSMessageResponse{
		Action: "message_offline",
		Data: map[string]interface{}{
			"message_id":  messageID,
			"receiver_id": msg.ReceiverID,
			"timestamp":   time.Now().Unix(),
			"note":        "对方当前不在线，消息将在其上线后推送",
		},
		Timestamp: time.Now().Unix(),
	}
	respBytes, _ := json.Marshal(response)
	c.SendChan <- respBytes
}

func (c *Client) buildReceiveMessage(messageID string, msg SendMessageReq) []byte {
	response := WSMessageResponse{
		Action:    "receive_message",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"id":              messageID,
			"message_id":      messageID,
			"sender_id":       c.UserID,
			"receiver_id":     msg.ReceiverID,
			"conversation_id": msg.ConversationID,
			"conv_type":       msg.ConvType,
			"msg_type":        msg.MsgType,
			"content":         msg.Content,
			"timestamp":       time.Now().Unix(),
			"status":          1,
			"is_recalled":     false,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}
	respBytes, _ := json.Marshal(response)
	return respBytes
}

func (c *Client) handleRecallMessage(data json.RawMessage) {
	var msg struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析撤回消息数据失败", "")
		return
	}
	if c.messageUC != nil {
		input := &biz.RecallMessageInput{MessageID: msg.MessageID}
		err := c.messageUC.RecallMessage(c.Ctx, input)
		if err != nil {
			c.sendError("recall_failed", "撤回消息失败: "+err.Error(), "")
			return
		}
		c.sendResponse("message_recalled", map[string]interface{}{
			"message_id": msg.MessageID,
			"status":     "recalled",
		}, "")
		c.notifyRecall(msg.MessageID)
	} else {
		c.sendError("service_unavailable", "消息服务不可用", "")
	}
}

func (c *Client) notifyRecall(messageID string) {
	if c.chat != nil {
		message, err := c.chat.GetMessageByID(c.Ctx, messageID)
		if err != nil || message == nil {
			return
		}
		recallMsg := WSMessageResponse{
			Action:    "message_recalled",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"message_id":  messageID,
				"sender_id":   message.SenderID,
				"receiver_id": message.ReceiverID,
				"conv_type":   message.ConvType,
				"recalled_by": c.UserID,
				"timestamp":   time.Now().Unix(),
			},
		}
		respBytes, _ := json.Marshal(recallMsg)
		if message.ConvType == 0 {
			c.Manager.BroadcastToUser(message.ReceiverID, respBytes)
		} else if message.ConvType == 1 {
			gmGetter, ok := c.chat.(interface {
				GetGroupMembers(ctx context.Context, groupID string) ([]string, error)
			})
			if ok {
				members, err := gmGetter.GetGroupMembers(c.Ctx, message.ReceiverID)
				if err == nil {
					for _, memberID := range members {
						if memberID != c.UserID {
							c.Manager.BroadcastToUser(memberID, respBytes)
						}
					}
				}
			}
		}
	}
}

func (c *Client) handleReadMessage(data json.RawMessage) {
	var msg struct {
		ConversationID string `json:"conversation_id"`
		MessageID      string `json:"message_id"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析已读消息数据失败", "")
		return
	}
	if c.messageUC != nil {
		input := &biz.MarkMessagesReadInput{
			ConversationID: msg.ConversationID,
			LastMsgID:      msg.MessageID,
		}
		err := c.messageUC.MarkMessagesRead(c.Ctx, input)
		if err != nil {
			c.sendError("read_failed", "标记消息已读失败: "+err.Error(), "")
			return
		}
		c.sendResponse("message_read", map[string]interface{}{
			"conversation_id": msg.ConversationID,
			"message_id":      msg.MessageID,
			"status":          "read",
		}, "")
		c.notifyMessageRead(msg.MessageID, msg.ConversationID)
	} else {
		c.sendError("service_unavailable", "消息服务不可用", "")
	}
}

func (c *Client) notifyMessageRead(messageID, conversationID string) {
	if c.chat != nil {
		message, err := c.chat.GetMessageByID(c.Ctx, messageID)
		if err != nil || message == nil {
			return
		}
		if message.ConvType == 0 && message.SenderID != c.UserID {
			readMsg := WSMessageResponse{
				Action:    "message_read_ack",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"message_id":      messageID,
					"reader_id":       c.UserID,
					"reader_name":     "用户",
					"timestamp":       time.Now().Unix(),
					"conversation_id": conversationID,
				},
			}
			respBytes, _ := json.Marshal(readMsg)
			c.Manager.BroadcastToUser(message.SenderID, respBytes)
		}
	}
}

func (c *Client) handleTyping(data json.RawMessage) {
	var msg struct {
		ReceiverID string `json:"receiver_id"`
		ConvType   int32  `json:"conv_type"`
		IsTyping   bool   `json:"is_typing"`
		Text       string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析输入状态数据失败", "")
		return
	}
	response := WSMessageResponse{
		Action:    "user_typing",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"sender_id":   c.UserID,
			"receiver_id": msg.ReceiverID,
			"conv_type":   msg.ConvType,
			"is_typing":   msg.IsTyping,
			"text":        msg.Text,
			"timestamp":   time.Now().Unix(),
		},
	}
	respBytes, _ := json.Marshal(response)
	if msg.ConvType == 0 {
		c.Manager.BroadcastToUser(msg.ReceiverID, respBytes)
	}
}

func (c *Client) sendPong() {
	response := WSMessageResponse{
		Action:    "pong",
		Timestamp: time.Now().Unix(),
	}
	respBytes, _ := json.Marshal(response)
	c.SendChan <- respBytes
}

func (c *Client) sendResponse(action string, data interface{}, clientMsgID string) {
	response := WSMessageResponse{
		Action:      action,
		Data:        data,
		ClientMsgID: clientMsgID,
		Timestamp:   time.Now().Unix(),
	}
	respBytes, _ := json.Marshal(response)
	c.SendChan <- respBytes
}

func (c *Client) sendError(action, errorMsg, clientMsgID string) {
	response := WSMessageResponse{
		Action:      action,
		ClientMsgID: clientMsgID,
		Timestamp:   time.Now().Unix(),
		Error:       errorMsg,
	}
	respBytes, _ := json.Marshal(response)
	c.SendChan <- respBytes
}

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

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := h.chat.UpdateUserOnlineStatus(ctx, claim.UserId, 1, "web")
		if err != nil {
			h.logger.Log(log.LevelError, "msg", "更新在线状态失败", "err", err)
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
