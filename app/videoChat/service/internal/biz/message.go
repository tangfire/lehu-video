package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// 消息内容
type MessageContent struct {
	Text          string `json:"text,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
	ImageWidth    int64  `json:"image_width,omitempty"`
	ImageHeight   int64  `json:"image_height,omitempty"`
	VoiceURL      string `json:"voice_url,omitempty"`
	VoiceDuration int64  `json:"voice_duration,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
	VideoCover    string `json:"video_cover,omitempty"`
	VideoDuration int64  `json:"video_duration,omitempty"`
	FileURL       string `json:"file_url,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	FileSize      int64  `json:"file_size,omitempty"`
	Extra         string `json:"extra,omitempty"`
}

// 消息领域对象
type Message struct {
	ID         int64
	SenderID   int64
	ReceiverID int64
	ConvType   int32 // 0:单聊, 1:群聊
	MsgType    int32 // 0:文本, 1:图片, 2:语音, 3:视频, 4:文件, 99:系统
	Content    *MessageContent
	Status     int32 // 0:发送中, 1:已发送, 2:已送达, 3:已读, 4:已撤回, 99:失败
	IsRecalled bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// 会话领域对象
type Conversation struct {
	ID          int64
	UserID      int64
	Type        int32
	TargetID    int64
	LastMessage string
	LastMsgType int32
	LastMsgTime time.Time
	UnreadCount int
	IsPinned    bool
	IsMuted     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Commands and Queries
type SendMessageCommand struct {
	SenderID    int64
	ReceiverID  int64
	ConvType    int32
	MsgType     int32
	Content     *MessageContent
	ClientMsgID string
}

type SendMessageResult struct {
	MessageID int64
}

type ListMessagesQuery struct {
	UserID    int64
	TargetID  int64
	ConvType  int32
	LastMsgID int64
	Limit     int
}

type ListMessagesResult struct {
	Messages  []*Message
	HasMore   bool
	LastMsgID int64
}

type RecallMessageCommand struct {
	MessageID int64
	UserID    int64
}

type RecallMessageResult struct{}

type MarkMessagesReadCommand struct {
	UserID    int64
	TargetID  int64
	ConvType  int32
	LastMsgID int64
}

type MarkMessagesReadResult struct{}

type GetUnreadCountQuery struct {
	UserID int64
}

type GetUnreadCountResult struct {
	TotalUnread int64
	ConvUnread  map[int64]int64
}

type ListConversationsQuery struct {
	UserID    int64
	PageStats PageStats
}

type ListConversationsResult struct {
	Conversations []*Conversation
	Total         int64
}

type DeleteConversationCommand struct {
	UserID         int64
	ConversationID int64
}

type DeleteConversationResult struct{}

type ClearMessagesCommand struct {
	UserID   int64
	TargetID int64
	ConvType int32
}

type ClearMessagesResult struct{}

// WebSocket消息
type WSMessage struct {
	Action      string
	Message     *Message
	Timestamp   int64
	ClientMsgID string
}

// 消息仓储接口
type MessageRepo interface {
	// 消息操作
	CreateMessage(ctx context.Context, message *Message) error
	GetMessageByID(ctx context.Context, id int64) (*Message, error)
	UpdateMessageStatus(ctx context.Context, id int64, status int32) error
	RecallMessage(ctx context.Context, id int64) error
	ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int) ([]*Message, error)
	CountUnreadMessages(ctx context.Context, userID, targetID int64, convType int32) (int64, error)
	MarkMessagesAsRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error

	// 会话操作
	CreateOrUpdateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*Conversation, error)
	GetConversationByID(ctx context.Context, id int64) (*Conversation, error)
	DeleteConversation(ctx context.Context, id int64) error
	ListConversations(ctx context.Context, userID int64, offset, limit int) ([]*Conversation, error)
	CountConversations(ctx context.Context, userID int64) (int64, error)

	// 消息已读记录
	CreateMessageRead(ctx context.Context, messageID, userID int64) error

	// 用户消息设置
	GetUserMessageSetting(ctx context.Context, userID, targetID int64, convType int32) (*UserMessageSetting, error)
	UpdateUserMessageSetting(ctx context.Context, setting *UserMessageSetting) error
}

// 用户消息设置领域对象
type UserMessageSetting struct {
	ID        int64
	UserID    int64
	TargetID  int64
	ConvType  int32
	IsMuted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MessageUsecase struct {
	repo MessageRepo
	log  *log.Helper
}

func NewMessageUsecase(repo MessageRepo, logger log.Logger) *MessageUsecase {
	return &MessageUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *MessageUsecase) SendMessage(ctx context.Context, cmd *SendMessageCommand) (*SendMessageResult, error) {
	// 验证参数
	if cmd.Content == nil {
		return nil, fmt.Errorf("消息内容不能为空")
	}

	// 验证消息类型
	switch cmd.MsgType {
	case 0: // 文本
		if cmd.Content.Text == "" {
			return nil, fmt.Errorf("文本消息内容不能为空")
		}
		if len(cmd.Content.Text) > 5000 {
			return nil, fmt.Errorf("文本消息长度不能超过5000个字符")
		}
	case 1: // 图片
		if cmd.Content.ImageURL == "" {
			return nil, fmt.Errorf("图片消息URL不能为空")
		}
	case 2: // 语音
		if cmd.Content.VoiceURL == "" {
			return nil, fmt.Errorf("语音消息URL不能为空")
		}
	case 3: // 视频
		if cmd.Content.VideoURL == "" {
			return nil, fmt.Errorf("视频消息URL不能为空")
		}
	case 4: // 文件
		if cmd.Content.FileURL == "" {
			return nil, fmt.Errorf("文件消息URL不能为空")
		}
	}

	// 创建消息
	message := &Message{
		SenderID:   cmd.SenderID,
		ReceiverID: cmd.ReceiverID,
		ConvType:   cmd.ConvType,
		MsgType:    cmd.MsgType,
		Content:    cmd.Content,
		Status:     0, // 发送中
		IsRecalled: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	message.ID = int64(uuid.New().ID())

	// 保存消息
	err := uc.repo.CreateMessage(ctx, message)
	if err != nil {
		uc.log.Errorf("保存消息失败: %v", err)
		return nil, fmt.Errorf("发送消息失败")
	}

	// 创建或更新会话
	conv := &Conversation{
		UserID:      cmd.SenderID,
		Type:        cmd.ConvType,
		TargetID:    cmd.ReceiverID,
		LastMessage: uc.getLastMessageText(cmd.MsgType, cmd.Content),
		LastMsgType: cmd.MsgType,
		LastMsgTime: time.Now(),
		UnreadCount: 0,
		UpdatedAt:   time.Now(),
	}

	// 如果是群聊，为所有群成员创建会话
	if cmd.ConvType == 1 { // 群聊
		// TODO: 获取群成员列表，为每个成员创建会话（不包括发送者）
		// 这里需要调用GroupRepo来获取群成员
	} else {
		// 单聊：为发送者创建会话
		err = uc.repo.CreateOrUpdateConversation(ctx, conv)
		if err != nil {
			uc.log.Errorf("更新会话失败: %v", err)
			// 不返回错误，消息已发送成功
		}

		// 为接收者创建会话
		conv.UserID = cmd.ReceiverID
		conv.UnreadCount = 1
		err = uc.repo.CreateOrUpdateConversation(ctx, conv)
		if err != nil {
			uc.log.Errorf("更新接收者会话失败: %v", err)
		}
	}

	return &SendMessageResult{
		MessageID: message.ID,
	}, nil
}

func (uc *MessageUsecase) getLastMessageText(msgType int32, content *MessageContent) string {
	switch msgType {
	case 0: // 文本
		return content.Text
	case 1: // 图片
		return "[图片]"
	case 2: // 语音
		return "[语音]"
	case 3: // 视频
		return "[视频]"
	case 4: // 文件
		return "[文件] " + content.FileName
	case 99: // 系统消息
		return "[系统消息]"
	default:
		return "[未知消息]"
	}
}

func (uc *MessageUsecase) ListMessages(ctx context.Context, query *ListMessagesQuery) (*ListMessagesResult, error) {
	// 参数验证
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	messages, err := uc.repo.ListMessages(ctx, query.UserID, query.TargetID, query.ConvType, query.LastMsgID, query.Limit)
	if err != nil {
		uc.log.Errorf("查询消息列表失败: %v", err)
		return nil, fmt.Errorf("查询消息失败")
	}

	var lastMsgID int64
	hasMore := false
	if len(messages) > 0 {
		lastMsgID = messages[len(messages)-1].ID
		if len(messages) == query.Limit {
			hasMore = true
		}
	}

	return &ListMessagesResult{
		Messages:  messages,
		HasMore:   hasMore,
		LastMsgID: lastMsgID,
	}, nil
}

func (uc *MessageUsecase) RecallMessage(ctx context.Context, cmd *RecallMessageCommand) (*RecallMessageResult, error) {
	// 获取消息
	message, err := uc.repo.GetMessageByID(ctx, cmd.MessageID)
	if err != nil {
		uc.log.Errorf("查询消息失败: %v", err)
		return nil, fmt.Errorf("消息不存在")
	}

	if message == nil {
		return nil, fmt.Errorf("消息不存在")
	}

	// 检查权限（只能撤回自己发送的消息）
	if message.SenderID != cmd.UserID {
		return nil, fmt.Errorf("只能撤回自己发送的消息")
	}

	// 检查时间（超过2分钟不能撤回）
	if time.Since(message.CreatedAt) > 2*time.Minute {
		return nil, fmt.Errorf("消息已超过撤回时间限制")
	}

	// 撤回消息
	err = uc.repo.RecallMessage(ctx, cmd.MessageID)
	if err != nil {
		uc.log.Errorf("撤回消息失败: %v", err)
		return nil, fmt.Errorf("撤回消息失败")
	}

	return &RecallMessageResult{}, nil
}

func (uc *MessageUsecase) MarkMessagesRead(ctx context.Context, cmd *MarkMessagesReadCommand) (*MarkMessagesReadResult, error) {
	err := uc.repo.MarkMessagesAsRead(ctx, cmd.UserID, cmd.TargetID, cmd.ConvType, cmd.LastMsgID)
	if err != nil {
		uc.log.Errorf("标记消息已读失败: %v", err)
		return nil, fmt.Errorf("标记消息已读失败")
	}

	// 更新会话未读计数
	conv, err := uc.repo.GetConversation(ctx, cmd.UserID, cmd.TargetID, cmd.ConvType)
	if err == nil && conv != nil {
		conv.UnreadCount = 0
		conv.UpdatedAt = time.Now()
		_ = uc.repo.CreateOrUpdateConversation(ctx, conv)
	}

	return &MarkMessagesReadResult{}, nil
}

func (uc *MessageUsecase) ListConversations(ctx context.Context, query *ListConversationsQuery) (*ListConversationsResult, error) {
	// 分页参数验证
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}

	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize

	conversations, err := uc.repo.ListConversations(ctx, query.UserID, int(offset), int(query.PageStats.PageSize))
	if err != nil {
		uc.log.Errorf("查询会话列表失败: %v", err)
		return nil, fmt.Errorf("查询会话列表失败")
	}

	total, err := uc.repo.CountConversations(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("统计会话数量失败: %v", err)
		return nil, fmt.Errorf("统计会话数量失败")
	}

	return &ListConversationsResult{
		Conversations: conversations,
		Total:         total,
	}, nil
}

func (uc *MessageUsecase) DeleteConversation(ctx context.Context, cmd *DeleteConversationCommand) (*DeleteConversationResult, error) {
	// 获取会话
	conv, err := uc.repo.GetConversationByID(ctx, cmd.ConversationID)
	if err != nil {
		uc.log.Errorf("查询会话失败: %v", err)
		return nil, fmt.Errorf("会话不存在")
	}

	if conv == nil {
		return nil, fmt.Errorf("会话不存在")
	}

	// 检查权限（只能删除自己的会话）
	if conv.UserID != cmd.UserID {
		return nil, fmt.Errorf("无权删除此会话")
	}

	// 删除会话
	err = uc.repo.DeleteConversation(ctx, cmd.ConversationID)
	if err != nil {
		uc.log.Errorf("删除会话失败: %v", err)
		return nil, fmt.Errorf("删除会话失败")
	}

	return &DeleteConversationResult{}, nil
}
