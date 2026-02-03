package model

import (
	"encoding/json"
	"time"
)

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
