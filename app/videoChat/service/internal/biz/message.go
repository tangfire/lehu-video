package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// IDGenerator ID生成器接口
type IDGenerator interface {
	Generate() int64
}

// 消息内容
type MessageContent struct {
	Text          string
	ImageURL      string
	ImageWidth    int64
	ImageHeight   int64
	VoiceURL      string
	VoiceDuration int64
	VideoURL      string
	VideoCover    string
	VideoDuration int64
	FileURL       string
	FileName      string
	FileSize      int64
	Extra         string
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
	LastMsgTime time.Time // 统一使用time.Time
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
	PageStats *PageStats
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

// 新增：更新消息状态Command
type UpdateMessageStatusCommand struct {
	MessageID  int64
	Status     int32
	OperatorID int64
}

type UpdateMessageStatusResult struct{}

// 新增：清空消息Command
type ClearMessagesCommand struct {
	UserID   int64
	TargetID int64
	ConvType int32
}

type ClearMessagesResult struct{}

type CreateConversationCommand struct {
	UserID         int64
	TargetID       int64
	ConvType       int32
	InitialMessage string
}

type CreateConversationResult struct {
	ConversationID int64
}

// 消息仓储接口
type MessageRepo interface {
	// 消息操作
	CreateMessage(ctx context.Context, message *Message) error
	GetMessageByID(ctx context.Context, id int64) (*Message, error)
	UpdateMessageStatus(ctx context.Context, id int64, status int32) error
	RecallMessage(ctx context.Context, id int64) error
	ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int) ([]*Message, bool, error)
	CountUnreadMessages(ctx context.Context, userID, targetID int64, convType int32) (int64, error)
	MarkMessagesAsRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error

	// 会话操作
	CreateOrUpdateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*Conversation, error)
	GetConversationByID(ctx context.Context, id int64) (*Conversation, error)
	DeleteConversation(ctx context.Context, id int64) error
	ListConversations(ctx context.Context, userID int64, offset, limit int) ([]*Conversation, error)
	CountConversations(ctx context.Context, userID int64) (int64, error)

	// 统计总未读数
	CountTotalUnread(ctx context.Context, userID int64) (int64, error)
	CountUnreadByConversations(ctx context.Context, userID int64) (map[int64]int64, error)

	// 新增方法
	DeleteMessagesByConversation(ctx context.Context, userID, targetID int64, convType int32) error // 新增：清空聊天记录
}

// 单聊会话ID生成器
func generateSingleChatConversationID(userID1, userID2 int64) int64 {
	// 确保小ID在前，大ID在后
	minID := userID1
	maxID := userID2
	if userID1 > userID2 {
		minID, maxID = userID2, userID1
	}

	// 使用位运算生成唯一ID
	// 高32位存储小ID，低32位存储大ID
	return (minID << 32) | (maxID & 0xFFFFFFFF)
}

// 从会话ID解析用户ID
func parseSingleChatUserIDs(conversationID int64) (int64, int64) {
	userID1 := conversationID >> 32
	userID2 := conversationID & 0xFFFFFFFF
	return userID1, userID2
}

// 生成群聊会话ID（每个用户独立）
func generateGroupConversationID(userID, groupID int64) int64 {
	// 使用雪花算法或其他分布式ID生成
	// 这里简单使用哈希组合
	return (groupID << 32) | (userID & 0xFFFFFFFF)
}

type MessageUsecase struct {
	repo       MessageRepo
	friendRepo FriendRepo  // 用于检查好友关系
	groupRepo  GroupRepo   // 用于检查群成员关系
	idGen      IDGenerator // ID生成器
	log        *log.Helper
}

func NewMessageUsecase(repo MessageRepo, friendRepo FriendRepo, groupRepo GroupRepo, idGen IDGenerator, logger log.Logger) *MessageUsecase {
	return &MessageUsecase{
		repo:       repo,
		friendRepo: friendRepo,
		groupRepo:  groupRepo,
		idGen:      idGen,
		log:        log.NewHelper(logger),
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

	// 检查权限
	if cmd.ConvType == 0 { // 单聊
		isFriend, _, err := uc.friendRepo.CheckFriendRelation(ctx, cmd.SenderID, cmd.ReceiverID)
		if err != nil {
			return nil, fmt.Errorf("检查好友关系失败")
		}
		if !isFriend {
			return nil, fmt.Errorf("你们不是好友，无法发送消息")
		}
	} else if cmd.ConvType == 1 { // 群聊
		isMember, err := uc.groupRepo.IsGroupMember(ctx, cmd.ReceiverID, cmd.SenderID)
		if err != nil {
			return nil, fmt.Errorf("检查群成员关系失败")
		}
		if !isMember {
			return nil, fmt.Errorf("你不是群成员，无法发送消息")
		}
	}

	// 使用ID生成器生成消息ID
	messageID := uc.idGen.Generate()

	// 创建消息
	now := time.Now()
	message := &Message{
		ID:         messageID,
		SenderID:   cmd.SenderID,
		ReceiverID: cmd.ReceiverID,
		ConvType:   cmd.ConvType,
		MsgType:    cmd.MsgType,
		Content:    cmd.Content,
		Status:     1, // 已发送
		IsRecalled: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 保存消息
	err := uc.repo.CreateMessage(ctx, message)
	if err != nil {
		uc.log.Errorf("保存消息失败: %v", err)
		return nil, fmt.Errorf("发送消息失败")
	}

	// 生成会话ID
	var senderConversationID int64
	var receiverConversationID int64
	lastMessageText := uc.getLastMessageText(cmd.MsgType, cmd.Content)

	if cmd.ConvType == 0 { // 单聊
		// 单聊使用统一的会话ID
		senderConversationID = generateSingleChatConversationID(cmd.SenderID, cmd.ReceiverID)
		receiverConversationID = senderConversationID // 两个用户共享同一会话ID

		// 为发送者创建或更新会话
		senderConv := &Conversation{
			ID:          senderConversationID,
			UserID:      cmd.SenderID,
			Type:        cmd.ConvType,
			TargetID:    cmd.ReceiverID,
			LastMessage: lastMessageText,
			LastMsgType: cmd.MsgType,
			LastMsgTime: now,
			UnreadCount: 0, // 发送者自己未读为0
			IsPinned:    false,
			IsMuted:     false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err = uc.repo.CreateOrUpdateConversation(ctx, senderConv)
		if err != nil {
			uc.log.Errorf("更新发送者会话失败: %v", err)
		}

		// 为接收者创建或更新会话
		receiverConv := &Conversation{
			ID:          receiverConversationID,
			UserID:      cmd.ReceiverID,
			Type:        cmd.ConvType,
			TargetID:    cmd.SenderID,
			LastMessage: lastMessageText,
			LastMsgType: cmd.MsgType,
			LastMsgTime: now,
			UnreadCount: 1, // 接收者未读+1
			IsPinned:    false,
			IsMuted:     false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err = uc.repo.CreateOrUpdateConversation(ctx, receiverConv)
		if err != nil {
			uc.log.Errorf("更新接收者会话失败: %v", err)
		}

	} else if cmd.ConvType == 1 { // 群聊
		// 群聊：获取所有群成员，为每个成员更新会话
		members, err := uc.groupRepo.GetGroupMembers(ctx, cmd.ReceiverID)
		if err != nil {
			uc.log.Errorf("获取群成员失败: %v", err)
			return &SendMessageResult{MessageID: messageID}, nil
		}

		for _, memberID := range members {
			// 为每个成员生成独立的会话ID
			memberConversationID := generateGroupConversationID(memberID, cmd.ReceiverID)

			unreadCount := 0
			if memberID == cmd.SenderID {
				unreadCount = 0 // 发送者自己未读为0
			} else {
				unreadCount = 1 // 其他成员未读+1
			}

			memberConv := &Conversation{
				ID:          memberConversationID,
				UserID:      memberID,
				Type:        cmd.ConvType,
				TargetID:    cmd.ReceiverID,
				LastMessage: lastMessageText,
				LastMsgType: cmd.MsgType,
				LastMsgTime: now,
				UnreadCount: unreadCount,
				IsPinned:    false,
				IsMuted:     false,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			err = uc.repo.CreateOrUpdateConversation(ctx, memberConv)
			if err != nil {
				uc.log.Errorf("更新群成员会话失败 user_id=%d: %v", memberID, err)
			}
		}
	}

	return &SendMessageResult{
		MessageID: message.ID,
	}, nil
}

func (uc *MessageUsecase) getLastMessageText(msgType int32, content *MessageContent) string {
	switch msgType {
	case 0: // 文本
		if len(content.Text) > 50 {
			return content.Text[:50] + "..."
		}
		return content.Text
	case 1: // 图片
		return "[图片]"
	case 2: // 语音
		return "[语音]"
	case 3: // 视频
		return "[视频]"
	case 4: // 文件
		return "[文件] " + content.FileName
	default:
		return "[消息]"
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

	// 使用改进的ListMessages方法，返回是否还有更多
	messages, hasMore, err := uc.repo.ListMessages(ctx, query.UserID, query.TargetID, query.ConvType, query.LastMsgID, query.Limit)
	if err != nil {
		uc.log.Errorf("查询消息列表失败: %v", err)
		return nil, fmt.Errorf("查询消息失败")
	}

	var lastMsgID int64
	if len(messages) > 0 {
		lastMsgID = messages[len(messages)-1].ID
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
	// 参数验证
	if cmd.LastMsgID < 0 {
		cmd.LastMsgID = 0
	}

	// ✅ 群聊：不需要标记已读，直接返回成功
	if cmd.ConvType == 1 {
		// 群聊不标记已读，直接返回成功
		return &MarkMessagesReadResult{}, nil
	}

	// 单聊：标记消息已读
	err := uc.repo.MarkMessagesAsRead(ctx, cmd.UserID, cmd.TargetID, cmd.ConvType, cmd.LastMsgID)
	if err != nil {
		uc.log.Errorf("标记消息已读失败: %v", err)
		return nil, fmt.Errorf("标记消息已读失败")
	}

	// 更新会话未读计数为0
	conv, err := uc.repo.GetConversation(ctx, cmd.UserID, cmd.TargetID, cmd.ConvType)
	if err == nil && conv != nil {
		conv.UnreadCount = 0
		conv.UpdatedAt = time.Now()
		_ = uc.repo.CreateOrUpdateConversation(ctx, conv)
	}

	return &MarkMessagesReadResult{}, nil
}

func (uc *MessageUsecase) GetUnreadCount(ctx context.Context, query *GetUnreadCountQuery) (*GetUnreadCountResult, error) {
	// 获取总未读数（只统计单聊）
	total, err := uc.repo.CountTotalUnread(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("统计总未读数失败: %v", err)
		return nil, fmt.Errorf("统计未读数失败")
	}

	// 获取会话未读数（只统计单聊）
	convUnread, err := uc.repo.CountUnreadByConversations(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("统计会话未读数失败: %v", err)
		return nil, fmt.Errorf("统计未读数失败")
	}

	return &GetUnreadCountResult{
		TotalUnread: total,
		ConvUnread:  convUnread,
	}, nil
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

	conversations, err := uc.repo.ListConversations(ctx, query.UserID, offset, query.PageStats.PageSize)
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

// 更新消息状态
func (uc *MessageUsecase) UpdateMessageStatus(ctx context.Context, cmd *UpdateMessageStatusCommand) (*UpdateMessageStatusResult, error) {
	// 验证参数
	if cmd.MessageID <= 0 {
		return nil, fmt.Errorf("消息ID不能为空")
	}

	// 验证状态值
	if cmd.Status < 0 || (cmd.Status > 4 && cmd.Status != 99) {
		return nil, fmt.Errorf("无效的消息状态")
	}

	// 检查消息是否存在
	message, err := uc.repo.GetMessageByID(ctx, cmd.MessageID)
	if err != nil {
		uc.log.Errorf("查询消息失败: %v", err)
		return nil, fmt.Errorf("消息不存在")
	}

	if message == nil {
		return nil, fmt.Errorf("消息不存在")
	}

	// 检查权限：只能更新自己相关消息的状态
	if cmd.OperatorID > 0 {
		if message.SenderID != cmd.OperatorID && message.ReceiverID != cmd.OperatorID {
			return nil, fmt.Errorf("无权更新此消息状态")
		}
	}

	// 更新消息状态
	err = uc.repo.UpdateMessageStatus(ctx, cmd.MessageID, cmd.Status)
	if err != nil {
		uc.log.Errorf("更新消息状态失败: %v", err)
		return nil, fmt.Errorf("更新消息状态失败")
	}

	return &UpdateMessageStatusResult{}, nil
}

// 获取消息详情
func (uc *MessageUsecase) GetMessage(ctx context.Context, messageID int64) (*Message, error) {
	message, err := uc.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		uc.log.Errorf("获取消息详情失败: %v", err)
		return nil, fmt.Errorf("获取消息失败")
	}

	if message == nil {
		return nil, fmt.Errorf("消息不存在")
	}

	return message, nil
}

// 获取会话详情
func (uc *MessageUsecase) GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*Conversation, error) {
	conversation, err := uc.repo.GetConversation(ctx, userID, targetID, convType)
	if err != nil {
		uc.log.Errorf("获取会话详情失败: %v", err)
		return nil, fmt.Errorf("获取会话失败")
	}

	return conversation, nil
}

// 清空聊天记录
func (uc *MessageUsecase) ClearMessages(ctx context.Context, userID, targetID int64, convType int32) (*ClearMessagesResult, error) {
	// 检查权限：只能清空自己的聊天记录
	// 这里可以添加额外的权限检查，比如检查是否是好友或群成员

	err := uc.repo.DeleteMessagesByConversation(ctx, userID, targetID, convType)
	if err != nil {
		uc.log.Errorf("清空聊天记录失败: %v", err)
		return nil, fmt.Errorf("清空聊天记录失败")
	}

	// 更新会话
	conv, err := uc.repo.GetConversation(ctx, userID, targetID, convType)
	if err == nil && conv != nil {
		// 清空最后一条消息和未读计数
		conv.LastMessage = ""
		conv.UnreadCount = 0
		conv.UpdatedAt = time.Now()
		_ = uc.repo.CreateOrUpdateConversation(ctx, conv)
	}

	return &ClearMessagesResult{}, nil
}

// CreateConversation 方法修复
func (uc *MessageUsecase) CreateConversation(ctx context.Context, cmd *CreateConversationCommand) (*CreateConversationResult, error) {
	// 检查权限
	if cmd.ConvType == 0 { // 单聊
		isFriend, _, err := uc.friendRepo.CheckFriendRelation(ctx, cmd.UserID, cmd.TargetID)
		if err != nil {
			return nil, fmt.Errorf("检查好友关系失败: %v", err)
		}
		if !isFriend {
			return nil, fmt.Errorf("你们不是好友，无法创建会话")
		}
	} else if cmd.ConvType == 1 { // 群聊
		isMember, err := uc.groupRepo.IsGroupMember(ctx, cmd.TargetID, cmd.UserID)
		if err != nil {
			return nil, fmt.Errorf("检查群成员关系失败: %v", err)
		}
		if !isMember {
			return nil, fmt.Errorf("你不是群成员，无法创建会话")
		}
	}

	// 生成会话ID
	var conversationID int64
	if cmd.ConvType == 0 {
		// 单聊：使用统一ID
		conversationID = generateSingleChatConversationID(cmd.UserID, cmd.TargetID)
	} else {
		// 群聊：用户独立的ID
		conversationID = generateGroupConversationID(cmd.UserID, cmd.TargetID)
	}

	// 检查是否已存在会话
	existingConv, err := uc.repo.GetConversation(ctx, cmd.UserID, cmd.TargetID, cmd.ConvType)
	if err == nil && existingConv != nil {
		return &CreateConversationResult{
			ConversationID: existingConv.ID,
		}, nil
	}

	// 创建会话
	now := time.Now()
	conversation := &Conversation{
		ID:          conversationID,
		UserID:      cmd.UserID,
		Type:        cmd.ConvType,
		TargetID:    cmd.TargetID,
		LastMessage: cmd.InitialMessage,
		LastMsgType: 0, // 文本
		LastMsgTime: now,
		UnreadCount: 0,
		IsPinned:    false,
		IsMuted:     false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = uc.repo.CreateOrUpdateConversation(ctx, conversation)
	if err != nil {
		uc.log.Errorf("创建会话失败: %v", err)
		return nil, fmt.Errorf("创建会话失败")
	}

	return &CreateConversationResult{
		ConversationID: conversationID,
	}, nil
}
