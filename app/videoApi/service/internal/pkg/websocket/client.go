package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"lehu-video/app/videoApi/service/internal/biz"
	"math/rand"
	"time"
)

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
		// 检查是否还是当前有效的连接
		c.Manager.RLock()
		current, ok := c.Manager.clients[c.UserID]
		isCurrent := ok && current == c
		c.Manager.RUnlock()

		if isCurrent {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := c.chat.UpdateUserOnlineStatus(ctx, c.UserID, 0, "")
			if err != nil {
				c.logger.Log(log.LevelError, "msg", "更新在线状态失败", "err", err)
			}
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
		c.Manager.unregister <- c // 确保清理
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

// handleSendMessage 处理发送消息请求（简化版：只写入Kafka）
func (c *Client) handleSendMessage(data json.RawMessage, outerClientMsgID string) {
	var msg SendMessageReq
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("parse_error", "解析消息数据失败", outerClientMsgID)
		return
	}

	// 确定最终使用的 client_msg_id
	effectiveClientMsgID := msg.ClientMsgID
	if effectiveClientMsgID == "" {
		effectiveClientMsgID = outerClientMsgID
	}
	// 如果仍然为空，生成一个随机ID（实际应由客户端生成，这里做兜底）
	if effectiveClientMsgID == "" {
		effectiveClientMsgID = fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(1000))
	}

	// 提取消息内容
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

	// 构造要写入Kafka的消息（不包含临时ID，仅用client_msg_id关联）
	messageData := map[string]interface{}{
		"client_msg_id":   effectiveClientMsgID,
		"sender_id":       c.UserID,
		"receiver_id":     msg.ReceiverID,
		"conversation_id": msg.ConversationID,
		"conv_type":       msg.ConvType,
		"msg_type":        msg.MsgType,
		"content":         content,
		"timestamp":       time.Now().Unix(),
	}
	value, _ := json.Marshal(messageData)

	// 发送到Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := c.Manager.kafkaProducer.SendMessage(ctx, []byte(c.UserID), value)
	if err != nil {
		c.logger.Log(log.LevelError, "msg", "发送消息到Kafka失败", "err", err)
		c.sendError("kafka_failed", "消息发送失败，请重试", effectiveClientMsgID)
		return
	}

	// 立即给发送者返回 message_sent（仅含client_msg_id，不含最终消息ID）
	c.sendResponse("message_sent", map[string]interface{}{
		"client_msg_id":   effectiveClientMsgID,
		"conversation_id": msg.ConversationID,
		"status":          0,
	}, effectiveClientMsgID)

	c.logger.Log(log.LevelInfo, "msg", "消息已写入Kafka",
		"client_msg_id", effectiveClientMsgID)
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

// 以下为其他处理函数（撤回、已读、输入状态等），保持不变
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
