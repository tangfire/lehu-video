package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"lehu-video/app/videoApi/service/internal/biz"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"lehu-video/app/videoApi/service/internal/pkg/kafka"
)

const (
	redisOnlinePrefix = "online:"     // 在线状态Key前缀
	redisPushChannel  = "push:global" // 全局推送频道（所有实例订阅）
)

// Client 表示一个WebSocket客户端连接
type Client struct {
	ID        int64
	UserID    string
	ConnID    string // 连接唯一标识，用于在线状态校验
	Ctx       context.Context
	Conn      *websocket.Conn
	SendChan  chan []byte
	Manager   *Manager
	logger    log.Logger
	messageUC *biz.MessageUsecase // *biz.MessageUsecase
	chat      biz.ChatAdapter     // biz.ChatAdapter
}

// Manager 管理所有WebSocket连接
type Manager struct {
	sync.RWMutex
	clients       map[string]*Client // userID -> Client（单设备登录）
	broadcast     chan *BroadcastMessage
	register      chan *Client
	unregister    chan *Client
	logger        log.Logger
	offlineMgr    *OfflineManager
	kafkaProducer *kafka.Producer
	redisClient   *redis.Client

	// Redis Pub/Sub 相关
	pubSub       *redis.PubSub
	pubSubCancel context.CancelFunc
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	UserIDs []string
	Message []byte
	IsGroup bool
	GroupID string
}

// NewManager 创建新的连接管理器
func NewManager(
	logger log.Logger,
	kafkaProducer *kafka.Producer,
	redisClient *redis.Client,
) *Manager {
	m := &Manager{
		clients:       make(map[string]*Client),
		broadcast:     make(chan *BroadcastMessage, 256),
		register:      make(chan *Client, 256),
		unregister:    make(chan *Client, 256),
		logger:        logger,
		offlineMgr:    NewOfflineManager(redisClient, logger),
		kafkaProducer: kafkaProducer,
		redisClient:   redisClient,
	}

	// 启动 Redis 订阅，接收跨实例推送
	m.startRedisSubscriber()

	return m
}

// IsUserOnlineGlobal 检查用户是否全局在线（通过Redis）
func (m *Manager) IsUserOnlineGlobal(ctx context.Context, userID string) bool {
	key := redisOnlinePrefix + userID
	err := m.redisClient.Get(ctx, key).Err()
	return err == nil // 如果key存在，则在线
}

// startRedisSubscriber 启动 Redis Pub/Sub 订阅，用于接收其他实例的推送
func (m *Manager) startRedisSubscriber() {
	ctx, cancel := context.WithCancel(context.Background())
	m.pubSubCancel = cancel

	pubSub := m.redisClient.Subscribe(ctx, redisPushChannel)
	m.pubSub = pubSub

	go func() {
		ch := pubSub.Channel()
		for msg := range ch {
			var payload struct {
				UserID  string `json:"user_id"`
				Message []byte `json:"message"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				m.logger.Log(log.LevelError, "msg", "解析Redis推送消息失败", "err", err)
				continue
			}
			// 只推送给本地在线的用户
			m.RLock()
			client, ok := m.clients[payload.UserID]
			m.RUnlock()
			if ok {
				select {
				case client.SendChan <- payload.Message:
				default:
					m.logger.Log(log.LevelWarn, "用户发送通道已满", "user_id", payload.UserID)
				}
			}
		}
	}()
}

// Start 启动管理器（处理注册、注销、广播）
func (m *Manager) Start() {
	for {
		select {
		case client := <-m.register:
			m.Lock()
			// 如果用户已有连接，关闭旧连接（单设备登录）
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
			m.Unlock()

			// 设置Redis在线状态（存储连接ID，用于竞态校验）
			ctx := context.Background()
			if err := m.setUserOnline(ctx, client.UserID, client.ConnID); err != nil {
				m.logger.Log(log.LevelError, "msg", "设置Redis在线状态失败", "err", err)
			}

			m.logger.Log(log.LevelInfo, "msg", "用户已连接", "user_id", client.UserID)

			// 投递离线消息
			go func(c *Client) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				sendFunc := func(msg []byte) {
					select {
					case c.SendChan <- msg:
					default:
						m.logger.Log(log.LevelWarn, "离线消息投递失败，发送通道已满", "user_id", c.UserID)
					}
				}
				if err := m.offlineMgr.DeliverOfflineMessages(ctx, c.UserID, sendFunc); err != nil {
					m.logger.Log(log.LevelError, "msg", "投递离线消息失败", "user_id", c.UserID, "err", err)
				}
			}(client)

		case client := <-m.unregister:
			m.Lock()
			current, exists := m.clients[client.UserID]
			if exists && current == client {
				// 当前连接是map中存储的，正常删除
				delete(m.clients, client.UserID)
				// 删除Redis在线状态（连接ID匹配）
				ctx := context.Background()
				if err := m.setUserOfflineIfMatch(ctx, client.UserID, client.ConnID); err != nil {
					m.logger.Log(log.LevelError, "msg", "删除Redis在线状态失败", "err", err)
				}
			} else if !exists {
				// map中没有该用户，说明是最后一个连接，直接删除Redis key（无条件）
				ctx := context.Background()
				key := redisOnlinePrefix + client.UserID
				if err := m.redisClient.Del(ctx, key).Err(); err != nil && err != redis.Nil {
					m.logger.Log(log.LevelError, "msg", "删除Redis在线状态失败", "err", err)
				}
			}
			// 如果exists但current != client，说明被新连接替换，不操作
			m.Unlock()
			m.logger.Log(log.LevelInfo, "msg", "用户已断开连接", "user_id", client.UserID)

		case bm := <-m.broadcast:
			m.RLock()
			for _, userID := range bm.UserIDs {
				if client, ok := m.clients[userID]; ok {
					select {
					case client.SendChan <- bm.Message:
					default:
						m.logger.Log(log.LevelWarn, "用户发送通道已满", "user_id", userID)
					}
				}
				// 注意：这里不再处理离线，离线由业务逻辑在Kafka消费者中处理
			}
			m.RUnlock()
		}
	}
}

// setUserOnline 设置用户在线，过期时间35秒
func (m *Manager) setUserOnline(ctx context.Context, userID, connID string) error {
	key := redisOnlinePrefix + userID
	return m.redisClient.Set(ctx, key, connID, 35*time.Second).Err()
}

// refreshUserOnline 刷新过期时间
func (m *Manager) refreshUserOnline(ctx context.Context, userID string) error {
	key := redisOnlinePrefix + userID
	return m.redisClient.Expire(ctx, key, 35*time.Second).Err()
}

// setUserOfflineIfMatch 如果Redis中存储的连接ID与当前一致，则删除
func (m *Manager) setUserOfflineIfMatch(ctx context.Context, userID, connID string) error {
	key := redisOnlinePrefix + userID
	val, err := m.redisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil // 已不存在，无需处理
	}
	if err != nil {
		return err
	}
	if val == connID {
		return m.redisClient.Del(ctx, key).Err()
	}
	// 连接ID不匹配，说明是旧连接的延迟操作，忽略
	return nil
}

// PushToUser 推送消息给指定用户（支持跨实例）
func (m *Manager) PushToUser(userID string, message []byte) {
	// 1. 先尝试本地推送
	if client, ok := m.clients[userID]; ok {
		select {
		case client.SendChan <- message:
			return
		default:
			m.logger.Log(log.LevelWarn, "本地推送失败，通道已满，尝试跨实例推送", "user_id", userID)
		}
	}

	// 2. 本地不在线或推送失败，通过Redis发布给其他实例
	go func() {
		payload := struct {
			UserID  string `json:"user_id"`
			Message []byte `json:"message"`
		}{
			UserID:  userID,
			Message: message,
		}
		data, _ := json.Marshal(payload)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.redisClient.Publish(ctx, redisPushChannel, data).Err(); err != nil {
			m.logger.Log(log.LevelError, "msg", "Redis发布推送消息失败", "err", err)
		}
	}()
}

// BroadcastToUser 兼容旧接口，内部调用 PushToUser
func (m *Manager) BroadcastToUser(userID string, message []byte) {
	m.PushToUser(userID, message)
}

// BroadcastToGroup 向群组所有成员发送消息（需要外部传入成员列表）
func (m *Manager) BroadcastToGroup(userIDs []string, message []byte) {
	for _, userID := range userIDs {
		m.PushToUser(userID, message)
	}
}

// IsUserOnline 检查用户是否在线（仅检查本地连接）
func (m *Manager) IsUserOnline(userID string) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.clients[userID]
	return ok
}

// GetUserConnectionCount 获取用户连接数（单设备登录，返回0或1）
func (m *Manager) GetUserConnectionCount(userID string) int {
	m.RLock()
	defer m.RUnlock()
	if _, ok := m.clients[userID]; ok {
		return 1
	}
	return 0
}

// Stop 停止管理器，释放资源
func (m *Manager) Stop() {
	if m.pubSubCancel != nil {
		m.pubSubCancel()
	}
	if m.pubSub != nil {
		_ = m.pubSub.Close()
	}
}
