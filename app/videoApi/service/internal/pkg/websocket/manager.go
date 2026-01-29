package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"lehu-video/app/videoApi/service/internal/biz"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 生产环境应该验证来源
		},
	}
)

// Client 表示一个WebSocket客户端连接
type Client struct {
	ID        int64
	UserID    int64
	Conn      *websocket.Conn
	SendChan  chan []byte
	Manager   *Manager
	logger    log.Logger
	messageUC *biz.MessageUsecase // 添加消息用例
}

// Manager 管理所有WebSocket连接
type Manager struct {
	sync.RWMutex
	clients    map[int64]*Client      // userID -> Client
	broadcast  chan *BroadcastMessage // 广播消息通道
	register   chan *Client           // 注册通道
	unregister chan *Client           // 注销通道
	logger     log.Logger
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	UserIDs []int64
	Message []byte
	IsGroup bool
	GroupID int64
}

// NewManager 创建新的连接管理器
func NewManager(logger log.Logger) *Manager {
	return &Manager{
		clients:    make(map[int64]*Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		logger:     logger,
	}
}

// Start 启动管理器
func (m *Manager) Start() {
	for {
		select {
		case client := <-m.register:
			m.Lock()
			m.clients[client.UserID] = client
			m.Unlock()
			m.logger.Log(log.LevelInfo, "msg", "用户已连接", "user_id", client.UserID)

		case client := <-m.unregister:
			m.Lock()
			if _, ok := m.clients[client.UserID]; ok {
				delete(m.clients, client.UserID)
				close(client.SendChan)
			}
			m.Unlock()
			m.logger.Log(log.LevelInfo, "msg", "用户已断开连接", "user_id", client.UserID)

		case bm := <-m.broadcast:
			m.RLock()
			if bm.IsGroup {
				// 群组广播
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
				// 单播或指定用户广播
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

// BroadcastToUser 向指定用户发送消息
func (m *Manager) BroadcastToUser(userID int64, message []byte) {
	m.broadcast <- &BroadcastMessage{
		UserIDs: []int64{userID},
		Message: message,
		IsGroup: false,
	}
}

// BroadcastToGroup 向群组所有成员发送消息
func (m *Manager) BroadcastToGroup(userIDs []int64, message []byte) {
	m.broadcast <- &BroadcastMessage{
		UserIDs: userIDs,
		Message: message,
		IsGroup: true,
	}
}

// NewClient 创建新的客户端连接
func NewClient(userID int64, conn *websocket.Conn, manager *Manager, messageUC *biz.MessageUsecase, logger log.Logger) *Client {
	return &Client{
		ID:        time.Now().UnixNano(),
		UserID:    userID,
		Conn:      conn,
		SendChan:  make(chan []byte, 256),
		Manager:   manager,
		logger:    logger,
		messageUC: messageUC,
	}
}

// ReadPump 读取客户端消息
func (c *Client) ReadPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(5120) // 5KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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

		// 处理消息
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
				// 通道已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 如果有更多消息，一并发送
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

// handleMessage 处理收到的消息
func (c *Client) handleMessage(message []byte) {
	var req WSMessageRequest
	if err := json.Unmarshal(message, &req); err != nil {
		c.sendError("invalid_message", "消息格式错误", "")
		return
	}

	// 根据action处理不同类型的消息
	switch req.Action {
	case "ping":
		c.sendPong()
	case "send_message":
		c.handleSendMessage(req.Data, req.ClientMsgID)
	case "recall_message":
		c.handleRecallMessage(req.Data)
	case "read_message":
		c.handleReadMessage(req.Data)
	default:
		c.sendError("unknown_action", "未知的操作类型", req.ClientMsgID)
	}
}

// handleSendMessage 处理发送消息请求
func (c *Client) handleSendMessage(data json.RawMessage, clientMsgID string) {
	var msg struct {
		ReceiverID int64                  `json:"receiver_id"`
		ConvType   int32                  `json:"conv_type"`
		MsgType    int32                  `json:"msg_type"`
		Content    map[string]interface{} `json:"content"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析消息数据失败", clientMsgID)
		return
	}

	// 构建消息内容
	content := &biz.MessageContent{}

	// 根据消息类型提取内容
	switch msg.MsgType {
	case 0: // 文本
		if text, ok := msg.Content["text"].(string); ok {
			content.Text = text
		}
	case 1: // 图片
		if imageURL, ok := msg.Content["image_url"].(string); ok {
			content.ImageURL = imageURL
		}
	case 2: // 语音
		if voiceURL, ok := msg.Content["voice_url"].(string); ok {
			content.VoiceURL = voiceURL
		}
	case 3: // 视频
		if videoURL, ok := msg.Content["video_url"].(string); ok {
			content.VideoURL = videoURL
		}
	case 4: // 文件
		if fileURL, ok := msg.Content["file_url"].(string); ok {
			content.FileURL = fileURL
		}
	}

	// 调用MessageUsecase发送消息
	if c.messageUC != nil {
		input := &biz.SendMessageInput{
			ReceiverID:  msg.ReceiverID,
			ConvType:    msg.ConvType,
			MsgType:     msg.MsgType,
			Content:     content,
			ClientMsgID: clientMsgID,
		}

		// 使用context.Background()，实际应该从请求中获取
		ctx := context.Background()
		output, err := c.messageUC.SendMessage(ctx, input)
		if err != nil {
			c.sendError("send_failed", "发送消息失败: "+err.Error(), clientMsgID)
			return
		}

		// 发送成功响应
		c.sendResponse("message_sent", map[string]interface{}{
			"message_id": output.MessageID,
			"status":     "sent",
		}, clientMsgID)

		// 如果接收者在线，推送消息
		if msg.ConvType == 0 { // 单聊
			c.Manager.BroadcastToUser(msg.ReceiverID, c.buildReceiveMessage(output.MessageID, msg))
		} else if msg.ConvType == 1 { // 群聊
			// TODO: 获取群成员列表，然后推送
			// 这里需要调用群聊服务获取成员列表
		}
	} else {
		c.sendError("service_unavailable", "消息服务不可用", clientMsgID)
	}
}

// buildReceiveMessage 构建接收消息
func (c *Client) buildReceiveMessage(messageID int64, msg struct {
	ReceiverID int64                  `json:"receiver_id"`
	ConvType   int32                  `json:"conv_type"`
	MsgType    int32                  `json:"msg_type"`
	Content    map[string]interface{} `json:"content"`
}) []byte {
	response := WSMessageResponse{
		Action:    "receive_message",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message_id": messageID,
			"sender_id":  c.UserID,
			"conv_type":  msg.ConvType,
			"msg_type":   msg.MsgType,
			"content":    msg.Content,
			"timestamp":  time.Now().Unix(),
		},
	}

	respBytes, _ := json.Marshal(response)
	return respBytes
}

func (c *Client) handleRecallMessage(data json.RawMessage) {
	var msg struct {
		MessageID int64 `json:"message_id"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析撤回消息数据失败", "")
		return
	}

	// TODO: 实现撤回消息逻辑
	c.sendResponse("message_recalled", map[string]interface{}{
		"message_id": msg.MessageID,
	}, "")
}

func (c *Client) handleReadMessage(data json.RawMessage) {
	var msg struct {
		MessageID int64 `json:"message_id"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析已读消息数据失败", "")
		return
	}

	// TODO: 实现标记消息已读逻辑
	c.sendResponse("message_read", map[string]interface{}{
		"message_id": msg.MessageID,
	}, "")
}

// sendPong 发送pong响应
func (c *Client) sendPong() {
	response := WSMessageResponse{
		Action:    "pong",
		Timestamp: time.Now().Unix(),
	}
	respBytes, _ := json.Marshal(response)
	c.SendChan <- respBytes
}

// sendResponse 发送成功响应
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

// sendError 发送错误响应
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
	logger    log.Logger
}

func NewHandler(manager *Manager, messageUC *biz.MessageUsecase, logger log.Logger) *Handler {
	return &Handler{
		manager:   manager,
		messageUC: messageUC,
		logger:    logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 从请求中获取token
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "未认证", http.StatusUnauthorized)
		return
	}

	// TODO: 从JWT token中解析用户ID
	// 这里简化处理，假设token就是userID的字符串形式
	userID, err := strconv.ParseInt(token, 10, 64)
	if err != nil || userID == 0 {
		http.Error(w, "无效的token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Log(log.LevelError, "msg", "WebSocket升级失败", "error", err)
		return
	}

	client := NewClient(userID, conn, h.manager, h.messageUC, h.logger)
	h.manager.register <- client

	go client.WritePump()
	go client.ReadPump()
}
