package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"net/http"
	"strings"
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
			return true
		},
	}
)

// Client 表示一个WebSocket客户端连接
type Client struct {
	ID        int64
	UserID    int64
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
	clients    map[int64]*Client
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	logger     log.Logger
	onlineMgr  *OnlineManager
	offlineMgr *OfflineManager
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	UserIDs []int64
	Message []byte
	IsGroup bool
	GroupID int64
}

// 获取用户连接数
func (m *Manager) GetUserConnectionCount(userID int64) int {
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
func NewManager(logger log.Logger, messageUC *biz.MessageUsecase, chat biz.ChatAdapter) *Manager {
	return &Manager{
		clients:    make(map[int64]*Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		logger:     logger,
		onlineMgr:  NewOnlineManager(),
		offlineMgr: NewOfflineManager(logger, messageUC, chat),
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
					time.Sleep(500 * time.Millisecond) // 延迟半秒
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

			go m.offlineMgr.DeliverOfflineMessages(client.UserID, client)

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

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID int64) bool {
	return m.onlineMgr.IsUserOnline(userID)
}

// BatchCheckOnline 批量检查用户在线状态
func (m *Manager) BatchCheckOnline(userIDs []int64) map[int64]bool {
	return m.onlineMgr.GetOnlineUsers(userIDs)
}

// NewClient 创建新的客户端连接
func NewClient(
	ctx context.Context,
	userID int64,
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

func parseJWTFromRequest(r *http.Request, secret string) (*claims.Claims, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, errors.New("missing Authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, errors.New("invalid Authorization format")
	}

	tokenStr := parts[1]
	token, err := jwtv5.ParseWithClaims(tokenStr, &claims.Claims{}, func(token *jwtv5.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	claim, ok := token.Claims.(*claims.Claims)
	if !ok {
		return nil, errors.New("claims type error")
	}

	return claim, nil
}

// ReadPump 读取客户端消息
func (c *Client) ReadPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(5120)
	//c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Manager.onlineMgr.UpdateLastActive(c.UserID)
		//c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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

// SendMessageReq 发送消息请求结构
type SendMessageReq struct {
	ReceiverID int64                  `json:"receiver_id"`
	ConvType   int32                  `json:"conv_type"`
	MsgType    int32                  `json:"msg_type"`
	Content    map[string]interface{} `json:"content"`
	GroupID    int64                  `json:"group_id,omitempty"`
}

// handleSendMessage 处理发送消息请求
func (c *Client) handleSendMessage(data json.RawMessage, clientMsgID string) {
	var msg SendMessageReq
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析消息数据失败", clientMsgID)
		return
	}

	// 验证接收者
	if msg.ReceiverID <= 0 {
		c.sendError("invalid_receiver", "无效的接收者ID", clientMsgID)
		return
	}

	// 检查用户关系（仅在单聊时检查好友关系）
	if msg.ConvType == 0 && c.chat != nil {
		checker, ok := c.chat.(interface {
			CheckFriendRelation(ctx context.Context, userID, targetID int64) (bool, int32, error)
		})

		if ok {
			isFriend, _, err := checker.CheckFriendRelation(c.Ctx, c.UserID, msg.ReceiverID)
			if err != nil {
				c.sendError("relation_check_failed", "检查好友关系失败", clientMsgID)
				return
			}
			if !isFriend {
				c.sendError("not_friend", "你们不是好友，无法发送消息", clientMsgID)
				return
			}
		}
	} else if msg.ConvType == 1 && c.chat != nil {
		// 群聊检查群成员关系
		checker, ok := c.chat.(interface {
			CheckUserRelation(ctx context.Context, userID, targetID int64, convType int32) (bool, error)
		})

		if ok {
			isMember, err := checker.CheckUserRelation(c.Ctx, c.UserID, msg.ReceiverID, msg.ConvType)
			if err != nil {
				c.sendError("relation_check_failed", "检查群成员关系失败", clientMsgID)
				return
			}
			if !isMember {
				c.sendError("not_member", "你不是群成员，无法发送消息", clientMsgID)
				return
			}
		}
	}

	// 构建消息内容
	content := &biz.MessageContent{}

	// 根据消息类型提取内容
	if msg.Content != nil {
		switch msg.MsgType {
		case 0: // 文本
			if text, ok := msg.Content["text"].(string); ok {
				content.Text = text
			}
		case 1: // 图片
			if imageURL, ok := msg.Content["image_url"].(string); ok {
				content.ImageURL = imageURL
			}
			if width, ok := msg.Content["image_width"].(float64); ok {
				content.ImageWidth = int64(width)
			}
			if height, ok := msg.Content["image_height"].(float64); ok {
				content.ImageHeight = int64(height)
			}
		case 2: // 语音
			if voiceURL, ok := msg.Content["voice_url"].(string); ok {
				content.VoiceURL = voiceURL
			}
			if duration, ok := msg.Content["voice_duration"].(float64); ok {
				content.VoiceDuration = int64(duration)
			}
		case 3: // 视频
			if videoURL, ok := msg.Content["video_url"].(string); ok {
				content.VideoURL = videoURL
			}
			if cover, ok := msg.Content["video_cover"].(string); ok {
				content.VideoCover = cover
			}
			if duration, ok := msg.Content["video_duration"].(float64); ok {
				content.VideoDuration = int64(duration)
			}
		case 4: // 文件
			if fileURL, ok := msg.Content["file_url"].(string); ok {
				content.FileURL = fileURL
			}
			if fileName, ok := msg.Content["file_name"].(string); ok {
				content.FileName = fileName
			}
			if fileSize, ok := msg.Content["file_size"].(float64); ok {
				content.FileSize = int64(fileSize)
			}
		}

		// 处理额外字段
		if extra, ok := msg.Content["extra"].(string); ok {
			content.Extra = extra
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

		output, err := c.messageUC.SendMessage(c.Ctx, input)
		if err != nil {
			c.sendError("send_failed", "发送消息失败: "+err.Error(), clientMsgID)
			return
		}

		// 发送成功响应
		c.sendResponse("message_sent", map[string]interface{}{
			"message_id":      output.MessageID,
			"conversation_id": output.ConversationId,
			"status":          "sent",
			"client_msg_id":   clientMsgID, // 确保返回客户端消息ID
		}, clientMsgID)

		// 构建推送消息
		pushMsg := c.buildReceiveMessage(output.MessageID, msg)

		// 处理消息推送（根据会话类型）
		if msg.ConvType == 0 { // 单聊
			// 立即更新消息状态为已送达（因为通过WebSocket直接推送）
			go c.updateMessageStatus(output.MessageID, 2) // DELIVERED

			// 推送给接收方
			c.Manager.BroadcastToUser(msg.ReceiverID, pushMsg)

		} else if msg.ConvType == 1 { // 群聊
			// 获取群成员并推送
			c.handleGroupMessage(msg.ReceiverID, output.MessageID, msg, pushMsg)
		}
	} else {
		c.sendError("service_unavailable", "消息服务不可用", clientMsgID)
	}
}

// updateMessageStatus 更新消息状态
func (c *Client) updateMessageStatus(messageID int64, status int32) {
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

// handleGroupMessage 处理群聊消息推送
func (c *Client) handleGroupMessage(groupID, messageID int64, msg SendMessageReq, pushMsg []byte) {
	// 获取群成员列表
	if c.chat != nil {
		gmGetter, ok := c.chat.(interface {
			GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error)
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

		// 统计在线/离线成员
		onlineCount := 0
		offlineCount := 0

		for _, memberID := range members {
			if memberID == c.UserID {
				continue // 跳过发送者自己
			}

			isOnline := c.Manager.IsUserOnline(memberID)
			if isOnline {
				onlineCount++
				// 推送给在线成员
				c.Manager.BroadcastToUser(memberID, pushMsg)
				// 发送者可以看到送达状态
				c.sendDeliveryConfirm(messageID, memberID)
			} else {
				offlineCount++
				// 存储为离线消息
				c.storeOfflineMessage(memberID, messageID, msg)
			}
		}

		// 发送群消息统计
		c.sendResponse("group_message_stat", map[string]interface{}{
			"message_id":    messageID,
			"group_id":      groupID,
			"online_count":  onlineCount,
			"offline_count": offlineCount,
			"total_members": len(members) - 1,
		}, "")
	}
}

// storeOfflineMessage 存储离线消息
func (c *Client) storeOfflineMessage(receiverID, messageID int64, msg SendMessageReq) {
	// 构建离线消息
	offlineMsg := &OfflineMessage{
		MessageID:  messageID,
		SenderID:   c.UserID,
		ReceiverID: receiverID,
		ConvType:   msg.ConvType,
		MsgType:    msg.MsgType,
		Content:    json.RawMessage{}, // 需要序列化
		CreatedAt:  time.Now(),
	}

	// 序列化内容
	if contentBytes, err := json.Marshal(msg.Content); err == nil {
		offlineMsg.Content = contentBytes
	}

	// 存储到离线管理器
	c.Manager.offlineMgr.StoreOfflineMessage(receiverID, offlineMsg)

	// 发送离线通知
	c.sendOfflineNotification(messageID, msg)
}

// 发送送达确认
func (c *Client) sendDeliveryConfirm(messageID int64, receiverID int64) {
	// 更新消息状态为已送达
	if c.chat != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := c.chat.UpdateMessageStatus(ctx, messageID, 2) // 2 = DELIVERED
			if err != nil {
				c.logger.Log(log.LevelError, "msg", "更新消息状态失败", "message_id", messageID, "error", err)
			}
		}()
	}

	// 向发送方推送送达通知
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

// 发送离线通知
func (c *Client) sendOfflineNotification(messageID int64, msg SendMessageReq) {
	// 向发送方推送离线通知
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

// buildReceiveMessage 构建接收消息
func (c *Client) buildReceiveMessage(messageID int64, msg SendMessageReq) []byte {
	response := WSMessageResponse{
		Action:    "receive_message",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"message_id":  messageID,
			"sender_id":   c.UserID,
			"receiver_id": msg.ReceiverID,
			"conv_type":   msg.ConvType,
			"msg_type":    msg.MsgType,
			"content":     msg.Content,
			"timestamp":   time.Now().Unix(),
		},
	}

	if msg.ConvType == 1 {
		response.Data.(map[string]interface{})["group_id"] = msg.GroupID
	}

	respBytes, _ := json.Marshal(response)
	return respBytes
}

// 处理消息撤回
func (c *Client) handleRecallMessage(data json.RawMessage) {
	var msg struct {
		MessageID int64 `json:"message_id"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析撤回消息数据失败", "")
		return
	}

	// 调用业务层撤回消息
	if c.messageUC != nil {
		input := &biz.RecallMessageInput{
			MessageID: msg.MessageID,
		}

		err := c.messageUC.RecallMessage(c.Ctx, input)
		if err != nil {
			c.sendError("recall_failed", "撤回消息失败: "+err.Error(), "")
			return
		}

		c.sendResponse("message_recalled", map[string]interface{}{
			"message_id": msg.MessageID,
			"status":     "recalled",
		}, "")

		// 通知相关方消息已被撤回
		c.notifyRecall(msg.MessageID)
	} else {
		c.sendError("service_unavailable", "消息服务不可用", "")
	}
}

// notifyRecall 通知消息撤回
func (c *Client) notifyRecall(messageID int64) {
	// 获取消息详情
	if c.chat != nil {
		message, err := c.chat.GetMessageByID(c.Ctx, messageID)
		if err != nil {
			c.logger.Log(log.LevelError, "msg", "获取消息详情失败", "message_id", messageID, "error", err)
			return
		}

		if message == nil {
			return
		}

		// 构建撤回通知
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

		// 单聊：通知对方
		if message.ConvType == 0 {
			c.Manager.BroadcastToUser(message.ReceiverID, respBytes)
		} else if message.ConvType == 1 {
			// 群聊：通知所有群成员
			gmGetter, ok := c.chat.(interface {
				GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error)
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

// 处理已读消息
func (c *Client) handleReadMessage(data json.RawMessage) {
	var msg struct {
		ConversationID int64 `json:"conversation_id"` // 之前是 TargetID
		MessageID      int64 `json:"message_id"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析已读消息数据失败", "")
		return
	}

	// 调用业务层标记消息已读
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

		// 通知发送方消息已读
		c.notifyMessageRead(msg.MessageID, msg.ConversationID)
	} else {
		c.sendError("service_unavailable", "消息服务不可用", "")
	}
}

// notifyMessageRead 通知消息已读
func (c *Client) notifyMessageRead(messageID, conversationID int64) {
	if c.chat != nil {
		message, err := c.chat.GetMessageByID(c.Ctx, messageID)
		if err != nil || message == nil {
			return
		}

		// 单聊判断还是用 message.ConvType
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

// handleTyping 处理正在输入状态
func (c *Client) handleTyping(data json.RawMessage) {
	var msg struct {
		ReceiverID int64  `json:"receiver_id"`
		ConvType   int32  `json:"conv_type"`
		IsTyping   bool   `json:"is_typing"`
		Text       string `json:"text,omitempty"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析输入状态数据失败", "")
		return
	}

	// 构建正在输入通知
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

	// 发送给接收方
	if msg.ConvType == 0 { // 单聊
		c.Manager.BroadcastToUser(msg.ReceiverID, respBytes)
	}
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
	// 1. 获取 Token (优先从查询参数获取，因为浏览器 WebSocket API 不支持自定义 Header)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// 备选：尝试从 Authorization Header 获取
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

	// 2. 解析并验证 JWT Token
	claim, err := parseJWT(tokenStr, h.jwtSecret)
	if err != nil {
		h.logger.Log(log.LevelError, "msg", "Token 验证失败", "err", err)
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// 3. 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Log(log.LevelError, "msg", "WebSocket 升级失败", "err", err)
		return
	}

	// 4. 构造上下文 (将用户信息存入 context 供后续业务使用)
	ctx := context.Background()
	ctx = kjwt.NewContext(ctx, claim)

	// 5. 创建 Client 实例
	// 注意：userID 从 claim 中获取，确保是 int64 类型
	client := NewClient(ctx, claim.UserId, conn, h.manager, h.messageUC, h.chat, h.logger)

	// 6. 将新连接注册到 Manager
	// Manager.Start 协程会处理此通道消息，并自动踢掉该用户之前的旧连接
	h.manager.register <- client

	// 7. 【关键】启动读写循环
	// WritePump 必须在后台协程运行，用于向客户端发送数据
	go client.WritePump()

	// ReadPump 在当前协程运行，它是一个阻塞循环，直到连接断开
	// 这会阻止 ServeHTTP 函数返回，从而维持连接生命周期
	client.ReadPump()
}

// 辅助函数：解析 JWT
func parseJWT(tokenStr string, secret string) (*claims.Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenStr, &claims.Claims{}, func(token *jwtv5.Token) (interface{}, error) {
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
