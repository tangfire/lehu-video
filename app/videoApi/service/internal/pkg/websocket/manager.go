package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/kafka"
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

			// 投递离线消息
			go func(c *Client) {
				m.logger.Log(log.LevelInfo, "启动离线消息投递", "user_id", c.UserID) // 新增
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
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
							m.logger.Log(log.LevelInfo, "消息已写入SendChan", "user_id", userID, "message_size", len(bm.Message))
						default:
							m.logger.Log(log.LevelWarn, "用户发送通道已满", "user_id", userID)
						}
					} else {
						m.logger.Log(log.LevelWarn, "用户不在clients中，无法推送", "user_id", userID, "online_status", m.onlineMgr.IsUserOnline(userID))
					}
				}
			} else {
				for _, userID := range bm.UserIDs {
					if client, ok := m.clients[userID]; ok {
						select {
						case client.SendChan <- bm.Message:
							m.logger.Log(log.LevelInfo, "消息已写入SendChan", "user_id", userID, "message_size", len(bm.Message))
						default:
							m.logger.Log(log.LevelWarn, "用户发送通道已满", "user_id", userID)
						}
					} else {
						m.logger.Log(log.LevelWarn, "用户不在clients中，无法推送", "user_id", userID, "online_status", m.onlineMgr.IsUserOnline(userID))
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
	m.logger.Log(log.LevelInfo, "准备向用户推送消息",
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
	m.logger.Log(log.LevelInfo, "准备向群组推送消息",
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
