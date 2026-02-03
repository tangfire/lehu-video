// biz/conversation.go - 统一会话模型
package biz

import (
	"context"
	"time"
)

// 会话类型
const (
	ConvTypeSingle = 0 // 单聊
	ConvTypeGroup  = 1 // 群聊
)

// Conversation 统一会话领域对象
type Conversation struct {
	ID          int64      `json:"id"`
	Type        int32      `json:"type"`     // 0:单聊, 1:群聊
	GroupID     int64      `json:"group_id"` // 群ID（群聊时）
	Name        string     `json:"name"`     // 会话名称
	Avatar      string     `json:"avatar"`   // 会话头像
	LastMessage string     `json:"last_message"`
	LastMsgType *int32     `json:"last_msg_type"`
	LastMsgTime *time.Time `json:"last_msg_time"`
	MemberCount int64      `json:"member_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	IsDeleted   bool       `json:"is_deleted"`
}

// ConversationMember 会话成员领域对象
type ConversationMember struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	UserID         int64     `json:"user_id"`
	Type           int32     `json:"type"` // 0:普通, 1:管理员, 2:群主
	UnreadCount    int32     `json:"unread_count"`
	LastReadMsgID  int64     `json:"last_read_msg_id"`
	IsPinned       bool      `json:"is_pinned"`
	IsMuted        bool      `json:"is_muted"`
	JoinTime       time.Time `json:"join_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	IsDeleted      bool      `json:"is_deleted"`
}

type ConversationView struct {
	// 会话基础信息
	ID      int64
	Type    int32
	GroupID int64
	Name    string
	Avatar  string

	LastMessage string
	LastMsgType *int32
	LastMsgTime *time.Time

	MemberCount int64
	MemberIDs   []int64

	CreatedAt time.Time
	UpdatedAt time.Time

	// ===== 成员态（关键）=====
	UnreadCount int64
	IsPinned    bool
	IsMuted     bool
}

// 会话仓储接口
type ConversationRepo interface {
	// 会话操作
	CreateConversation(ctx context.Context, conv *Conversation) (int64, error)
	GetConversation(ctx context.Context, id int64) (*Conversation, error)
	// 获取带用户状态的会话视图
	GetConversationView(
		ctx context.Context,
		conversationID int64,
		userID int64,
	) (*ConversationView, error)

	GetSingleChatConversation(ctx context.Context, userID1, userID2 int64) (*Conversation, error)
	GetOrCreateSingleChatConversation(ctx context.Context, userID1, userID2 int64) (*Conversation, error)
	GetGroupConversation(ctx context.Context, groupID int64) (*Conversation, error)
	GetOrCreateGroupConversation(ctx context.Context, groupID int64) (*Conversation, error)
	UpdateConversationLastMsg(ctx context.Context, conversationID int64, lastMessage string, lastMsgType int32) error
	DeleteConversation(ctx context.Context, id int64) error

	// 成员操作
	AddConversationMember(ctx context.Context, member *ConversationMember) error
	RemoveConversationMember(ctx context.Context, conversationID, userID int64) error
	GetConversationMember(ctx context.Context, conversationID, userID int64) (*ConversationMember, error)
	GetConversationMembers(ctx context.Context, conversationID int64) ([]*ConversationMember, error)
	GetConversationMemberCount(ctx context.Context, conversationID int64) (int64, error)
	UpdateMemberUnreadCount(ctx context.Context, conversationID, userID int64, delta int) error
	ResetMemberUnreadCount(ctx context.Context, conversationID, userID int64) error
	UpdateMemberLastRead(ctx context.Context, conversationID, userID, lastReadMsgID int64) error
	UpdateMemberSettings(ctx context.Context, conversationID, userID int64, isPinned, isMuted bool) error

	// 用户相关查询
	GetUserTotalUnreadCount(ctx context.Context, userID int64) (int64, error)
	GetUserConversationUnreadCount(ctx context.Context, userID int64) (map[int64]int64, error)
	GetConversationMembersByConversationIDs(
		ctx context.Context,
		conversationIDs []int64,
	) (map[int64][]*ConversationMember, error)

	// 会话成员列表（分页）
	ListConversationMembers(
		ctx context.Context,
		userID int64,
		page *PageStats,
	) ([]*ConversationMember, int64, error)

	// 批量查会话
	GetConversationsByIDs(
		ctx context.Context,
		ids []int64,
	) ([]*Conversation, error)
}
