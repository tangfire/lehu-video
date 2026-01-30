package websocket

import (
	"sync"
	"time"
)

// OnlineManager 在线状态管理器
type OnlineManager struct {
	sync.RWMutex
	onlineUsers map[int64]*UserOnlineStatus // userID -> 在线状态
}

// UserOnlineStatus 用户在线状态
type UserOnlineStatus struct {
	UserID     int64     `json:"user_id"`
	IsOnline   bool      `json:"is_online"`
	LastActive time.Time `json:"last_active"`
	DeviceType string    `json:"device_type"` // web/mobile/desktop
	ConnID     string    `json:"conn_id"`     // 连接ID，用于区分不同设备的连接
}

// NewOnlineManager 创建在线状态管理器
func NewOnlineManager() *OnlineManager {
	return &OnlineManager{
		onlineUsers: make(map[int64]*UserOnlineStatus),
	}
}

// SetUserOnline 设置用户在线
func (m *OnlineManager) SetUserOnline(userID int64, deviceType, connID string) {
	m.Lock()
	defer m.Unlock()

	m.onlineUsers[userID] = &UserOnlineStatus{
		UserID:     userID,
		IsOnline:   true,
		LastActive: time.Now(),
		DeviceType: deviceType,
		ConnID:     connID,
	}
}

// SetUserOffline 设置用户离线
func (m *OnlineManager) SetUserOffline(userID int64, connID string) {
	m.Lock()
	defer m.Unlock()

	if status, exists := m.onlineUsers[userID]; exists {
		// 如果是同一个连接的设备下线
		if status.ConnID == connID {
			delete(m.onlineUsers, userID)
		}
	}
}

// IsUserOnline 检查用户是否在线
func (m *OnlineManager) IsUserOnline(userID int64) bool {
	m.RLock()
	defer m.RUnlock()

	status, exists := m.onlineUsers[userID]
	return exists && status.IsOnline
}

// GetOnlineUsers 获取在线用户列表
func (m *OnlineManager) GetOnlineUsers(userIDs []int64) map[int64]bool {
	m.RLock()
	defer m.RUnlock()

	result := make(map[int64]bool)
	for _, userID := range userIDs {
		status, exists := m.onlineUsers[userID]
		result[userID] = exists && status.IsOnline
	}

	return result
}

// UpdateLastActive 更新最后活跃时间
func (m *OnlineManager) UpdateLastActive(userID int64) {
	m.Lock()
	defer m.Unlock()

	if status, exists := m.onlineUsers[userID]; exists {
		status.LastActive = time.Now()
	}
}

// CleanupInactiveUsers 清理不活跃的用户连接
func (m *OnlineManager) CleanupInactiveUsers(timeout time.Duration) {
	m.Lock()
	defer m.Unlock()

	now := time.Now()
	for userID, status := range m.onlineUsers {
		if now.Sub(status.LastActive) > timeout {
			delete(m.onlineUsers, userID)
		}
	}
}
