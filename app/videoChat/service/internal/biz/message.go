package biz

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// IDGenerator ID生成器接口
type IDGenerator interface {
	Generate() int64
}

// MessageContent 消息内容结构体
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

// Message 消息领域对象（修复版）
type Message struct {
	ID             int64           `json:"id"`
	SenderID       int64           `json:"sender_id"`
	ReceiverID     int64           `json:"receiver_id"`
	ConversationID int64           `json:"conversation_id"`
	ConvType       int32           `json:"conv_type"` // 0:单聊, 1:群聊
	MsgType        int32           `json:"msg_type"`  // 0:文本, 1:图片, 2:语音, 3:视频, 4:文件, 99:系统
	Content        *MessageContent `json:"content"`
	Status         int32           `json:"status"` // 0:发送中, 1:已发送, 2:已送达, 3:已读, 4:已撤回, 99:失败
	IsRecalled     bool            `json:"is_recalled"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	IsDeleted      bool            `json:"is_deleted"`
}

type SendMessageCommand struct {
	SenderID       int64
	ReceiverID     int64
	ConversationID int64
	ConvType       int32
	MsgType        int32
	ClientMsgID    string
	Content        *MessageContent
}

type SendMessageResult struct {
	MessageID      int64
	ConversationID int64
}

// ListMessagesQuery 查询消息列表
type ListMessagesQuery struct {
	UserID         int64
	ConversationID int64
	LastMsgID      int64
	Limit          int
}

// ListMessagesResult 查询消息列表结果
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

// MarkMessagesReadCommand 标记消息已读命令
type MarkMessagesReadCommand struct {
	UserID         int64
	ConversationID int64
	LastMsgID      int64
}

// MarkMessagesReadResult 标记消息已读结果
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
	Conversations []*ConversationView
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

type ClearMessagesCommand struct {
	UserID         int64
	ConversationID int64
}

type ClearMessagesResult struct{}

type CreateConversationCommand struct {
	UserIDs        []int64
	GroupID        int64
	ConvType       int32
	InitialMessage string
}

type CreateConversationResult struct {
	ConversationID int64
}

// 消息仓储接口
// biz/message.go - 修复MessageRepo接口
type MessageRepo interface {
	// 消息操作
	CreateMessage(ctx context.Context, message *Message) error
	GetMessageByID(ctx context.Context, id int64) (*Message, error)
	UpdateMessageStatus(ctx context.Context, id int64, status int32) error
	RecallMessage(ctx context.Context, id int64) error
	ListMessages(ctx context.Context, conversationID, lastMsgID int64, limit int) ([]*Message, bool, error)
	CountUnreadMessages(ctx context.Context, userID, targetID int64, convType int32) (int64, error)
	MarkMessagesAsRead(ctx context.Context, conversationID, userID, lastMsgID int64) error
	DeleteMessagesByConversation(ctx context.Context, userID, conversationID, targetID int64, convType int32) error
	CountTotalUnread(ctx context.Context, userID int64) (int64, error)
	CountUnreadByConversations(ctx context.Context, userID int64) (map[int64]int64, error)
	BatchUpdateMessageStatus(ctx context.Context, messageIDs []int64, status int32) error
}

type MessageUsecase struct {
	messageRepo      MessageRepo
	friendRepo       FriendRepo // 用于检查好友关系
	groupRepo        GroupRepo  // 用于检查群成员关系
	conversationRepo ConversationRepo
	idGen            IDGenerator // ID生成器
	log              *log.Helper
}

func NewMessageUsecase(messageRepo MessageRepo,
	friendRepo FriendRepo,
	groupRepo GroupRepo,
	conversationRepo ConversationRepo,
	idGen IDGenerator,
	logger log.Logger) *MessageUsecase {
	return &MessageUsecase{
		messageRepo:      messageRepo,
		friendRepo:       friendRepo,
		groupRepo:        groupRepo,
		conversationRepo: conversationRepo,
		idGen:            idGen,
		log:              log.NewHelper(logger),
	}
}

// SendMessage 发送消息（适配新会话设计）
func (uc *MessageUsecase) SendMessage(ctx context.Context, cmd *SendMessageCommand) (*SendMessageResult, error) {
	// 1. 验证消息参数
	if err := uc.validateMessage(cmd); err != nil {
		return nil, err
	}

	// 2. 检查权限
	if err := uc.checkPermission(ctx, cmd); err != nil {
		return nil, err
	}

	// 3. 获取或创建会话
	conversation, err := uc.getOrCreateConversation(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %v", err)
	}

	// 4. 创建消息
	messageID := uc.idGen.Generate()
	now := time.Now()

	message := &Message{
		ID:             messageID,
		SenderID:       cmd.SenderID,
		ReceiverID:     cmd.ReceiverID,
		ConversationID: conversation.ID,
		ConvType:       cmd.ConvType,
		MsgType:        cmd.MsgType,
		Content:        cmd.Content,
		Status:         1, // 已发送
		IsRecalled:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
		IsDeleted:      false,
	}

	// 5. 保存消息
	if err := uc.messageRepo.CreateMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("保存消息失败: %v", err)
	}

	// 6. 更新会话最后一条消息
	lastMessage := uc.getLastMessageText(cmd.MsgType, cmd.Content)
	if err := uc.conversationRepo.UpdateConversationLastMsg(ctx, conversation.ID, lastMessage, cmd.MsgType); err != nil {
		uc.log.Errorf("更新会话最后消息失败: %v", err)
	}

	// 7. 更新成员未读计数（除了发送者自己）
	members, err := uc.conversationRepo.GetConversationMembers(ctx, conversation.ID)
	if err != nil {
		uc.log.Errorf("获取会话成员失败: %v", err)
	} else {
		for _, member := range members {
			if member.UserID != cmd.SenderID {
				// 增加未读计数
				if err := uc.conversationRepo.UpdateMemberUnreadCount(ctx, conversation.ID, member.UserID, 1); err != nil {
					uc.log.Errorf("更新未读计数失败 user_id=%d: %v", member.UserID, err)
				}
			}
		}
	}

	return &SendMessageResult{
		MessageID:      messageID,
		ConversationID: conversation.ID,
	}, nil
}

// validateMessage 验证消息参数
func (uc *MessageUsecase) validateMessage(cmd *SendMessageCommand) error {
	if cmd.Content == nil {
		return fmt.Errorf("消息内容不能为空")
	}

	switch cmd.MsgType {
	case 0: // 文本
		if cmd.Content.Text == "" {
			return fmt.Errorf("文本消息内容不能为空")
		}
		if len(cmd.Content.Text) > 5000 {
			return fmt.Errorf("文本消息长度不能超过5000个字符")
		}
	case 1: // 图片
		if cmd.Content.ImageURL == "" {
			return fmt.Errorf("图片消息URL不能为空")
		}
	case 2: // 语音
		if cmd.Content.VoiceURL == "" {
			return fmt.Errorf("语音消息URL不能为空")
		}
	case 3: // 视频
		if cmd.Content.VideoURL == "" {
			return fmt.Errorf("视频消息URL不能为空")
		}
	case 4: // 文件
		if cmd.Content.FileURL == "" {
			return fmt.Errorf("文件消息URL不能为空")
		}
	}

	return nil
}

// checkPermission 检查发送权限
func (uc *MessageUsecase) checkPermission(ctx context.Context, cmd *SendMessageCommand) error {
	if cmd.ConvType == 0 { // 单聊
		isFriend, err := uc.friendRepo.CheckFriendRelation(ctx, cmd.SenderID, cmd.ReceiverID)
		if err != nil {
			return fmt.Errorf("检查好友关系失败: %v", err)
		}
		if !isFriend {
			return fmt.Errorf("你们不是好友，无法发送消息")
		}
	} else if cmd.ConvType == 1 { // 群聊
		isMember, err := uc.groupRepo.IsGroupMember(ctx, cmd.ReceiverID, cmd.SenderID)
		if err != nil {
			return fmt.Errorf("检查群成员关系失败: %v", err)
		}
		if !isMember {
			return fmt.Errorf("你不是群成员，无法发送消息")
		}
	}

	return nil
}

// getOrCreateConversation 获取或创建会话
func (uc *MessageUsecase) getOrCreateConversation(ctx context.Context, cmd *SendMessageCommand) (*Conversation, error) {
	if cmd.ConvType == 0 { // 单聊
		return uc.getOrCreateSingleChatConversation(ctx, cmd.ConversationID, cmd.SenderID, cmd.ReceiverID)
	} else if cmd.ConvType == 1 { // 群聊
		return uc.getOrCreateGroupConversation(ctx, cmd.ReceiverID)
	}

	return nil, fmt.Errorf("不支持的会话类型: %d", cmd.ConvType)
}

// getOrCreateSingleChatConversation 获取或创建单聊会话
func (uc *MessageUsecase) getOrCreateSingleChatConversation(ctx context.Context, conversationID int64, userID1, userID2 int64) (*Conversation, error) {
	// 尝试获取现有会话
	conversation, err := uc.conversationRepo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation != nil {
		return conversation, nil
	}

	// 创建新的单聊会话
	newConversationID := uc.idGen.Generate()
	now := time.Now()

	conversation = &Conversation{
		ID:          newConversationID,
		Type:        0, // 单聊
		MemberCount: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 创建会话
	if _, err := uc.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}

	// 添加用户1为成员
	if err := uc.conversationRepo.AddConversationMember(ctx, &ConversationMember{
		ConversationID: newConversationID,
		UserID:         userID1,
		Type:           0, // 普通成员
		UnreadCount:    0,
		JoinTime:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return nil, fmt.Errorf("添加成员失败: %v", err)
	}

	// 添加用户2为成员
	if err := uc.conversationRepo.AddConversationMember(ctx, &ConversationMember{
		ConversationID: newConversationID,
		UserID:         userID2,
		Type:           0, // 普通成员
		UnreadCount:    0,
		JoinTime:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return nil, fmt.Errorf("添加成员失败: %v", err)
	}

	return conversation, nil
}

// getOrCreateGroupConversation 修复后的群聊会话创建
func (uc *MessageUsecase) getOrCreateGroupConversation(ctx context.Context, groupID int64) (*Conversation, error) {
	// 尝试获取现有会话
	conversation, err := uc.conversationRepo.GetGroupConversation(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if conversation != nil {
		return conversation, nil
	}

	// 获取群信息（使用正确的方法名）
	group, err := uc.groupRepo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("获取群信息失败: %v", err)
	}

	if group == nil {
		return nil, fmt.Errorf("群聊不存在")
	}

	// 获取群成员数量
	memberCount, err := uc.groupRepo.CountGroupMembers(ctx, groupID)
	if err != nil {
		memberCount = 1 // 默认值
	}

	// 创建新的群聊会话
	conversationID := uc.idGen.Generate()
	now := time.Now()

	conversation = &Conversation{
		ID:          conversationID,
		Type:        ConvTypeGroup,
		GroupID:     groupID,
		Name:        group.Name,
		Avatar:      group.Avatar,
		MemberCount: memberCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 创建会话
	if _, err := uc.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}

	// 获取所有群成员并添加到会话
	members, err := uc.groupRepo.ListGroupMembers(ctx, groupID, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("获取群成员失败: %v", err)
	}

	for _, member := range members {
		memberType := int32(0) // 普通成员
		if member.Role == 2 {  // 群主
			memberType = 2
		} else if member.Role == 1 { // 管理员
			memberType = 1
		}

		if err := uc.conversationRepo.AddConversationMember(ctx, &ConversationMember{
			ConversationID: conversationID,
			UserID:         member.UserID,
			Type:           memberType,
			UnreadCount:    0,
			JoinTime:       now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}); err != nil {
			uc.log.Errorf("添加群成员到会话失败 user_id=%d: %v", member.UserID, err)
		}
	}

	return conversation, nil
}

// getLastMessageText 获取最后一条消息的摘要文本
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

// ListMessages 查询消息列表
func (uc *MessageUsecase) ListMessages(ctx context.Context, query *ListMessagesQuery) (*ListMessagesResult, error) {
	// 参数验证
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	// 获取消息
	messages, hasMore, err := uc.messageRepo.ListMessages(ctx, query.ConversationID, query.LastMsgID, query.Limit)
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

// MarkMessagesRead 修复标记消息已读
func (uc *MessageUsecase) MarkMessagesRead(ctx context.Context, cmd *MarkMessagesReadCommand) (*MarkMessagesReadResult, error) {
	// 更新成员最后已读消息ID
	if cmd.LastMsgID > 0 {
		if err := uc.conversationRepo.UpdateMemberLastRead(ctx, cmd.ConversationID, cmd.UserID, cmd.LastMsgID); err != nil {
			return nil, fmt.Errorf("更新已读状态失败: %v", err)
		}
	}

	// 重置未读计数
	if err := uc.conversationRepo.ResetMemberUnreadCount(ctx, cmd.ConversationID, cmd.UserID); err != nil {
		return nil, fmt.Errorf("重置未读计数失败: %v", err)
	}

	// 更新消息状态为已读
	if cmd.LastMsgID > 0 {
		// 调用正确的参数
		if err := uc.messageRepo.MarkMessagesAsRead(ctx, cmd.ConversationID, cmd.UserID, cmd.LastMsgID); err != nil {
			uc.log.Errorf("标记消息已读失败: %v", err)
		}
	}

	return &MarkMessagesReadResult{}, nil
}

func (uc *MessageUsecase) ListConversations(
	ctx context.Context,
	query *ListConversationsQuery,
) (*ListConversationsResult, error) {

	// 1️⃣ 查用户的会话成员关系（这是入口）
	members, total, err := uc.conversationRepo.ListConversationMembers(
		ctx,
		query.UserID,
		query.PageStats,
	)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return &ListConversationsResult{
			Conversations: []*ConversationView{},
			Total:         0,
		}, nil
	}

	// 2️⃣ 批量拿 conversation_id
	convIDs := make([]int64, 0, len(members))
	memberMap := make(map[int64]*ConversationMember)

	for _, m := range members {
		convIDs = append(convIDs, m.ConversationID)
		memberMap[m.ConversationID] = m
	}

	memberIdsMap, err := uc.conversationRepo.GetConversationMembersByConversationIDs(ctx, convIDs)
	if err != nil {
		return nil, err
	}

	// 3️⃣ 批量查询会话
	convs, err := uc.conversationRepo.GetConversationsByIDs(ctx, convIDs)
	if err != nil {
		return nil, err
	}

	// 4️⃣ 组装 View
	views := make([]*ConversationView, 0, len(convs))
	for _, conv := range convs {
		m := memberMap[conv.ID]
		if m == nil {
			continue
		}
		conversationMembers := memberIdsMap[conv.ID]
		memberIDs := make([]int64, 0, len(conversationMembers))
		for _, member := range conversationMembers {
			memberIDs = append(memberIDs, member.UserID)
		}

		view := &ConversationView{
			ID:          conv.ID,
			Type:        conv.Type,
			GroupID:     conv.GroupID,
			Name:        conv.Name,
			Avatar:      conv.Avatar,
			LastMessage: conv.LastMessage,
			LastMsgType: conv.LastMsgType,
			LastMsgTime: conv.LastMsgTime,
			MemberCount: conv.MemberCount,
			CreatedAt:   conv.CreatedAt,
			UpdatedAt:   conv.UpdatedAt,
			MemberIDs:   memberIDs,
			UnreadCount: int64(m.UnreadCount),
			IsPinned:    m.IsPinned,
			IsMuted:     m.IsMuted,
		}

		views = append(views, view)
	}

	// 5️⃣ 排序（非常重要）
	sort.SliceStable(views, func(i, j int) bool {
		// 置顶优先
		if views[i].IsPinned != views[j].IsPinned {
			return views[i].IsPinned
		}

		// 按最后消息时间倒序
		ti := time.Time{}
		tj := time.Time{}
		if views[i].LastMsgTime != nil {
			ti = *views[i].LastMsgTime
		}
		if views[j].LastMsgTime != nil {
			tj = *views[j].LastMsgTime
		}
		return ti.After(tj)
	})

	return &ListConversationsResult{
		Conversations: views,
		Total:         total,
	}, nil
}

// GetUnreadCount 获取未读消息数
func (uc *MessageUsecase) GetUnreadCount(ctx context.Context, query *GetUnreadCountQuery) (*GetUnreadCountResult, error) {
	total, err := uc.conversationRepo.GetUserTotalUnreadCount(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("统计总未读数失败: %v", err)
		return nil, fmt.Errorf("统计未读数失败")
	}

	convUnread, err := uc.conversationRepo.GetUserConversationUnreadCount(ctx, query.UserID)
	if err != nil {
		uc.log.Errorf("统计会话未读数失败: %v", err)
		return nil, fmt.Errorf("统计未读数失败")
	}

	return &GetUnreadCountResult{
		TotalUnread: total,
		ConvUnread:  convUnread,
	}, nil
}

// UpdateMessageStatus 修复更新消息状态
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
	message, err := uc.messageRepo.GetMessageByID(ctx, cmd.MessageID)
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
	err = uc.messageRepo.UpdateMessageStatus(ctx, cmd.MessageID, cmd.Status)
	if err != nil {
		uc.log.Errorf("更新消息状态失败: %v", err)
		return nil, fmt.Errorf("更新消息状态失败")
	}

	return &UpdateMessageStatusResult{}, nil
}

// 获取消息详情
func (uc *MessageUsecase) GetMessage(ctx context.Context, messageID int64) (*Message, error) {
	message, err := uc.messageRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		uc.log.Errorf("获取消息详情失败: %v", err)
		return nil, fmt.Errorf("获取消息失败")
	}

	if message == nil {
		return nil, fmt.Errorf("消息不存在")
	}

	return message, nil
}

// ClearMessages 修复清空聊天记录
func (uc *MessageUsecase) ClearMessages(ctx context.Context, cmd *ClearMessagesCommand) (*ClearMessagesResult, error) {
	// 1. 验证参数
	if cmd.UserID == 0 || cmd.ConversationID == 0 {
		return nil, fmt.Errorf("参数错误")
	}

	// 2. 验证用户是否属于该会话
	member, err := uc.conversationRepo.GetConversationMember(ctx, cmd.ConversationID, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("验证会话成员失败: %v", err)
	}

	if member == nil {
		return nil, fmt.Errorf("用户不属于该会话")
	}

	// 3. 获取会话信息
	conversation, err := uc.conversationRepo.GetConversation(ctx, cmd.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话信息失败: %v", err)
	}

	if conversation == nil {
		return nil, fmt.Errorf("会话不存在")
	}

	// 4. 清空该用户在该会话中的消息
	// 注意：这里应该只清空该用户的消息，而不是整个会话的消息
	// 单聊：清空自己发送的和接收的消息
	// 群聊：清空自己发送的消息
	var convType int32
	var groupID int64
	if conversation.Type == 0 { // 单聊
		// 获取对方ID
		convType = 0
	} else if conversation.Type == 1 { // 群聊
		// 群聊的目标ID就是群ID
		if conversation.GroupID == 0 {
			return nil, fmt.Errorf("群聊会话缺少群ID")
		}
		groupID = conversation.GroupID
		convType = 1
	} else {
		return nil, fmt.Errorf("不支持的会话类型")
	}

	// 5. 调用仓储层清空消息
	err = uc.messageRepo.DeleteMessagesByConversation(ctx, cmd.UserID, cmd.ConversationID, groupID, convType)
	if err != nil {
		uc.log.Errorf("清空聊天记录失败: %v", err)
		return nil, fmt.Errorf("清空聊天记录失败")
	}

	// 6. 重置未读计数（重要！）
	err = uc.conversationRepo.ResetMemberUnreadCount(ctx, cmd.ConversationID, cmd.UserID)
	if err != nil {
		uc.log.Warnf("重置未读计数失败: %v", err)
	}

	// 7. 更新会话的最后消息（可选，可以考虑将最后消息置空或设为系统提示）
	// 这里可以添加逻辑：更新会话最后消息为 "聊天记录已清空" 等系统提示

	return &ClearMessagesResult{}, nil
}

// CreateConversation 修复创建会话
func (uc *MessageUsecase) CreateConversation(ctx context.Context, cmd *CreateConversationCommand) (*CreateConversationResult, error) {
	// 检查权限已在API层处理，这里直接创建会话
	var conversation *Conversation
	var err error

	if cmd.ConvType == ConvTypeSingle {
		conversation, err = uc.conversationRepo.GetOrCreateSingleChatConversation(ctx, cmd.UserIDs[0], cmd.UserIDs[1])
	} else if cmd.ConvType == ConvTypeGroup {
		conversation, err = uc.conversationRepo.GetOrCreateGroupConversation(ctx, cmd.GroupID)
	} else {
		return nil, fmt.Errorf("不支持的会话类型")
	}

	if err != nil {
		return nil, err
	}

	// 如果有初始消息，可以创建一条系统消息或直接调用 SendMessage，但这里简化
	if cmd.InitialMessage != "" && cmd.ConvType == ConvTypeSingle {
		// 可选：创建一条初始消息（例如打招呼）
		// 可以调用 SendMessage 或直接插入一条消息
	}

	return &CreateConversationResult{
		ConversationID: conversation.ID,
	}, nil
}

// RecallMessage 撤回消息
func (uc *MessageUsecase) RecallMessage(ctx context.Context, cmd *RecallMessageCommand) (*RecallMessageResult, error) {
	// 获取消息
	message, err := uc.messageRepo.GetMessageByID(ctx, cmd.MessageID)
	if err != nil {
		uc.log.Errorf("获取消息失败: %v", err)
		return nil, fmt.Errorf("消息不存在")
	}

	if message == nil {
		return nil, fmt.Errorf("消息不存在")
	}

	// 检查权限：只能撤回自己发送的消息
	if message.SenderID != cmd.UserID {
		// 如果是群聊，检查是否是群主或管理员
		if message.ConvType == 1 { // 群聊
			isOwnerOrAdmin, err := uc.isGroupOwnerOrAdmin(ctx, message.ReceiverID, cmd.UserID)
			if err != nil || !isOwnerOrAdmin {
				return nil, fmt.Errorf("无权撤回此消息")
			}
		} else {
			return nil, fmt.Errorf("只能撤回自己发送的消息")
		}
	}

	// 检查消息是否已超过撤回时间（2分钟内）
	if time.Since(message.CreatedAt) > 2*time.Minute {
		return nil, fmt.Errorf("消息已超过撤回时间")
	}

	// 撤回消息
	err = uc.messageRepo.RecallMessage(ctx, cmd.MessageID)
	if err != nil {
		uc.log.Errorf("撤回消息失败: %v", err)
		return nil, fmt.Errorf("撤回消息失败")
	}

	return &RecallMessageResult{}, nil
}

// isGroupOwnerOrAdmin 检查是否为群主或管理员
func (uc *MessageUsecase) isGroupOwnerOrAdmin(ctx context.Context, groupID, userID int64) (bool, error) {
	// 获取群成员信息
	member, err := uc.groupRepo.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		return false, err
	}

	if member == nil {
		return false, nil
	}

	// 检查角色：群主(2)或管理员(1)
	return member.Role == 2 || member.Role == 1, nil
}

// DeleteConversation 删除会话
func (uc *MessageUsecase) DeleteConversation(ctx context.Context, cmd *DeleteConversationCommand) (*DeleteConversationResult, error) {
	// 检查会话是否存在
	member, err := uc.conversationRepo.GetConversationMember(ctx, cmd.ConversationID, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("获取会话成员失败: %v", err)
	}

	if member == nil {
		return &DeleteConversationResult{}, nil // 已经是删除状态
	}

	// 删除会话成员（软删除）
	err = uc.conversationRepo.RemoveConversationMember(ctx, cmd.ConversationID, cmd.UserID)
	if err != nil {
		uc.log.Errorf("删除会话失败: %v", err)
		return nil, fmt.Errorf("删除会话失败")
	}

	return &DeleteConversationResult{}, nil
}

// 在 message_usecase.go 中添加方法
func (uc *MessageUsecase) GetConversationView(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (*ConversationView, error) {
	return uc.conversationRepo.GetConversationView(ctx, conversationID, userID)
}
