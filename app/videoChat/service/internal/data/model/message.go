package model

import (
	"encoding/json"
	"time"
)

// 消息表
type Message struct {
	ID         int64           `gorm:"column:id;primaryKey;comment:消息ID"`
	SenderID   int64           `gorm:"column:sender_id;not null;comment:发送者ID"`
	ReceiverID int64           `gorm:"column:receiver_id;not null;comment:接收者ID"`
	ConvType   int8            `gorm:"column:conv_type;not null;comment:会话类型 0:单聊 1:群聊"`
	MsgType    int8            `gorm:"column:msg_type;not null;comment:消息类型 0:文本 1:图片 2:语音 3:视频 4:文件 99:系统"`
	Content    json.RawMessage `gorm:"column:content;type:json;comment:消息内容"`
	Status     int8            `gorm:"column:status;default:0;comment:消息状态 0:发送中 1:已发送 2:已送达 3:已读 4:已撤回 99:失败"`
	IsRecalled bool            `gorm:"column:is_recalled;default:0;comment:是否已撤回"`
	CreatedAt  time.Time       `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted  bool            `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (Message) TableName() string {
	return "message"
}

// 会话表
type Conversation struct {
	ID          int64     `gorm:"column:id;primaryKey;comment:会话ID"`
	UserID      int64     `gorm:"column:user_id;not null;comment:用户ID"`
	Type        int8      `gorm:"column:type;not null;comment:会话类型 0:单聊 1:群聊"`
	TargetID    int64     `gorm:"column:target_id;not null;comment:对方ID（用户ID或群ID）"`
	LastMessage string    `gorm:"column:last_message;type:text;comment:最后一条消息内容"`
	LastMsgType int8      `gorm:"column:last_msg_type;comment:最后一条消息类型"`
	LastMsgTime time.Time `gorm:"column:last_msg_time;comment:最后一条消息时间"`
	UnreadCount int       `gorm:"column:unread_count;default:0;comment:未读消息数"`
	IsPinned    bool      `gorm:"column:is_pinned;default:0;comment:是否置顶"`
	IsMuted     bool      `gorm:"column:is_muted;default:0;comment:是否免打扰"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted   bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (Conversation) TableName() string {
	return "conversation"
}

// 消息已读记录表
type MessageRead struct {
	ID        int64     `gorm:"column:id;primaryKey;comment:记录ID"`
	MessageID int64     `gorm:"column:message_id;not null;comment:消息ID"`
	UserID    int64     `gorm:"column:user_id;not null;comment:用户ID"`
	ReadAt    time.Time `gorm:"column:read_at;not null;comment:阅读时间"`
	CreatedAt time.Time `gorm:"column:created_at;not null;comment:创建时间"`
}

func (MessageRead) TableName() string {
	return "message_read"
}

// 用户消息设置表
type UserMessageSetting struct {
	ID        int64     `gorm:"column:id;primaryKey;comment:设置ID"`
	UserID    int64     `gorm:"column:user_id;not null;comment:用户ID"`
	TargetID  int64     `gorm:"column:target_id;not null;comment:对方ID（用户ID或群ID）"`
	ConvType  int8      `gorm:"column:conv_type;not null;comment:会话类型"`
	IsMuted   bool      `gorm:"column:is_muted;default:0;comment:是否免打扰"`
	CreatedAt time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted bool      `gorm:"column:is_deleted;default:0;NOT NULL"`
}

func (UserMessageSetting) TableName() string {
	return "user_message_setting"
}
