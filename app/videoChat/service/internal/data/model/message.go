package model

import (
	"encoding/json"
	"time"
)

// Conversation 会话主表
type Conversation struct {
	ID          int64      `gorm:"column:id;primaryKey;comment:会话ID"`
	Type        int8       `gorm:"column:type;not null;comment:会话类型 0:单聊 1:群聊"`
	TargetID    *int64     `gorm:"column:target_id;comment:对方用户ID（仅单聊有效）"`
	GroupID     *int64     `gorm:"column:group_id;comment:群ID（仅群聊有效）"`
	Name        string     `gorm:"column:name;size:100;default:'';comment:会话名称"`
	Avatar      string     `gorm:"column:avatar;size:500;default:'';comment:会话头像"`
	LastMessage string     `gorm:"column:last_message;type:text;comment:最后一条消息内容"`
	LastMsgType *int32     `gorm:"column:last_msg_type;comment:最后一条消息类型"`
	LastMsgTime *time.Time `gorm:"column:last_msg_time;comment:最后一条消息时间"`
	MemberCount int64      `gorm:"column:member_count;default:1;comment:成员数量"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted   bool       `gorm:"column:is_deleted;default:0;NOT NULL;comment:是否删除"`
}

func (Conversation) TableName() string {
	return "conversation"
}

// ConversationMember 会话成员表
type ConversationMember struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID"`
	ConversationID int64     `gorm:"column:conversation_id;not null;comment:会话ID"`
	UserID         int64     `gorm:"column:user_id;not null;comment:用户ID"`
	Type           int8      `gorm:"column:type;not null;default:0;comment:成员类型 0:普通 1:管理员 2:群主"`
	UnreadCount    int32     `gorm:"column:unread_count;default:0;comment:未读消息数"`
	LastReadMsgID  int64     `gorm:"column:last_read_msg_id;default:0;comment:最后已读消息ID"`
	IsPinned       bool      `gorm:"column:is_pinned;default:0;comment:是否置顶"`
	IsMuted        bool      `gorm:"column:is_muted;default:0;comment:是否免打扰"`
	JoinTime       time.Time `gorm:"column:join_time;default:CURRENT_TIMESTAMP;comment:加入时间"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted      bool      `gorm:"column:is_deleted;default:0;NOT NULL;comment:是否删除"`
}

func (ConversationMember) TableName() string {
	return "conversation_member"
}

// Message 消息表（更新）
type Message struct {
	ID             int64           `gorm:"column:id;primaryKey;comment:消息ID"`
	SenderID       int64           `gorm:"column:sender_id;not null;comment:发送者ID"`
	ReceiverID     int64           `gorm:"column:receiver_id;not null;comment:接收者ID（用户ID或群ID）"`
	ConversationID int64           `gorm:"column:conversation_id;comment:会话ID"`
	ConvType       int8            `gorm:"column:conv_type;not null;comment:会话类型 0:单聊 1:群聊"`
	MsgType        int8            `gorm:"column:msg_type;not null;comment:消息类型 0:文本 1:图片 2:语音 3:视频 4:文件 99:系统"`
	Content        json.RawMessage `gorm:"column:content;type:json;comment:消息内容"`
	Status         int8            `gorm:"column:status;default:0;comment:消息状态 0:发送中 1:已发送 2:已送达 3:已读 4:已撤回 99:失败"`
	IsRecalled     bool            `gorm:"column:is_recalled;default:0;comment:是否已撤回"`
	CreatedAt      time.Time       `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt      time.Time       `gorm:"column:updated_at;not null;comment:更新时间"`
	IsDeleted      bool            `gorm:"column:is_deleted;default:0;NOT NULL;comment:是否删除"`
}

func (Message) TableName() string {
	return "message"
}
