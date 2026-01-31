package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"time"
)

// MessageContent 消息内容结构体
type MessageContent struct {
	Text          string `json:"text,omitempty"`           // 文本内容
	ImageURL      string `json:"image_url,omitempty"`      // 图片URL地址
	ImageWidth    int64  `json:"image_width,omitempty"`    // 图片宽度(像素)
	ImageHeight   int64  `json:"image_height,omitempty"`   // 图片高度(像素)
	VoiceURL      string `json:"voice_url,omitempty"`      // 语音消息URL地址
	VoiceDuration int64  `json:"voice_duration,omitempty"` // 语音时长(单位:毫秒)
	VideoURL      string `json:"video_url,omitempty"`      // 视频URL地址
	VideoCover    string `json:"video_cover,omitempty"`    // 视频封面图片URL
	VideoDuration int64  `json:"video_duration,omitempty"` // 视频时长(单位:毫秒)
	FileURL       string `json:"file_url,omitempty"`       // 文件URL地址
	FileName      string `json:"file_name,omitempty"`      // 文件名(包含扩展名)
	FileSize      int64  `json:"file_size,omitempty"`      // 文件大小(单位:字节)
	Extra         string `json:"extra,omitempty"`          // 扩展字段，可用于存储自定义数据
}

// Message 消息结构体
type Message struct {
	ID         int64           `json:"id"`                                // 消息唯一标识ID
	SenderID   int64           `json:"sender_id"`                         // 发送者用户ID
	ReceiverID int64           `json:"receiver_id"`                       // 接收者ID
	ConvType   int8            `json:"conv_type"`                         // 会话类型: 0=单聊, 1=群聊
	MsgType    int8            `json:"msg_type"`                          // 消息类型: 0=文本, 1=图片, 2=语音, 3=视频, 4=文件
	Content    *MessageContent `json:"content"`                           // 消息内容
	Status     int8            `json:"status"`                            // 消息状态: 0=发送中, 1=已发送, 2=已送达, 3=已读
	IsRecalled bool            `json:"is_recalled"`                       // 消息是否被撤回
	CreatedAt  time.Time       `json:"created_at"`                        // 消息创建时间
	UpdatedAt  time.Time       `json:"updated_at"`                        // 消息最后更新时间
	DeletedAt  gorm.DeletedAt  `json:"deleted_at,omitempty" gorm:"index"` // 消息删除时间(软删除标识)
}

// 将 Message 转换为 JSON 字符串（用于存储或传输）
func (m *Message) ToJSON() (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// 从 JSON 字符串解析 Message
func (m *Message) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), m)
}

// Conversation 会话结构体
type Conversation struct {
	ID          int64
	UserID      int64
	Type        int32
	TargetID    *int64 // 注意这里用指针，匹配 data 层实现
	GroupID     *int64 // 注意这里用指针
	Name        string
	Avatar      string
	LastMessage string
	LastMsgType *int32     // 匹配 data 层
	LastMsgTime *time.Time // 匹配 data 层使用的 time.Unix 转换
	UnreadCount int32
	MemberCount int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 消息相关输入输出（修复版）
type SendMessageInput struct {
	ReceiverID  int64
	ConvType    int32
	MsgType     int32
	Content     *MessageContent
	ClientMsgID string
}

type SendMessageOutput struct {
	MessageID      int64
	ConversationId int64
}

type ListMessagesInput struct {
	ConversationID int64 // 重点：对应 proto 中的 conversation_id
	LastMsgID      int64
	Limit          int32
}

type ListMessagesOutput struct {
	Messages  []*Message
	HasMore   bool
	LastMsgID int64
}

type RecallMessageInput struct {
	MessageID int64
}

type MarkMessagesReadInput struct {
	ConversationID int64 // 对应 proto 的 conversation_id
	LastMsgID      int64
}

type ListConversationsInput struct {
	PageStats *PageStats
}

type ListConversationsOutput struct {
	Conversations []*Conversation
	Total         int64
}

type DeleteConversationInput struct {
	ConversationID int64
}

type CreateConversationInput struct {
	TargetID       int64
	ConvType       int32
	InitialMessage string
}

type CreateConversationOutput struct {
	ConversationID int64
}

// MessageUsecase（修复版）
type MessageUsecase struct {
	chat ChatAdapter
	log  *log.Helper
}

func NewMessageUsecase(chat ChatAdapter, logger log.Logger) *MessageUsecase {
	return &MessageUsecase{
		chat: chat,
		log:  log.NewHelper(logger),
	}
}

func (uc *MessageUsecase) SendMessage(ctx context.Context, input *SendMessageInput) (*SendMessageOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	// 参数验证
	if input.Content == nil {
		return nil, errors.New("消息内容不能为空")
	}

	// 根据消息类型验证内容
	switch input.MsgType {
	case 0: // 文本
		if input.Content.Text == "" {
			return nil, errors.New("文本消息内容不能为空")
		}
		if len(input.Content.Text) > 5000 {
			return nil, errors.New("文本消息长度不能超过5000个字符")
		}
	case 1: // 图片
		if input.Content.ImageURL == "" {
			return nil, errors.New("图片消息URL不能为空")
		}
	case 2: // 语音
		if input.Content.VoiceURL == "" {
			return nil, errors.New("语音消息URL不能为空")
		}
	case 3: // 视频
		if input.Content.VideoURL == "" {
			return nil, errors.New("视频消息URL不能为空")
		}
	case 4: // 文件
		if input.Content.FileURL == "" {
			return nil, errors.New("文件消息URL不能为空")
		}
	}

	// 发送消息
	messageID, conversationId, err := uc.chat.SendMessage(ctx, userID, input.ReceiverID, input.ConvType, input.MsgType, input.Content, input.ClientMsgID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("发送消息失败: %v", err)
		return nil, err
	}

	return &SendMessageOutput{
		MessageID:      messageID,
		ConversationId: conversationId,
	}, nil
}

// ListMessages 获取消息列表（修复版）
func (uc *MessageUsecase) ListMessages(ctx context.Context, input *ListMessagesInput) (*ListMessagesOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	// 注意：这里需要chat适配器支持按会话ID查询消息
	messages, hasMore, lastMsgID, err := uc.chat.ListMessages(ctx, userID, input.ConversationID, input.LastMsgID, input.Limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取消息列表失败: %v", err)
		return nil, errors.New("获取消息失败")
	}

	return &ListMessagesOutput{
		Messages:  messages,
		HasMore:   hasMore,
		LastMsgID: lastMsgID,
	}, nil
}

func (uc *MessageUsecase) RecallMessage(ctx context.Context, input *RecallMessageInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.RecallMessage(ctx, input.MessageID, userID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("撤回消息失败: %v", err)
		return errors.New("撤回消息失败")
	}

	return nil
}

func (uc *MessageUsecase) MarkMessagesRead(ctx context.Context, input *MarkMessagesReadInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	// 调用 chatAdapter
	err = uc.chat.MarkMessagesRead(ctx, userID, input.ConversationID, input.LastMsgID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("标记消息已读失败: %v", err)
		return errors.New("标记消息已读失败")
	}

	return nil
}

func (uc *MessageUsecase) ListConversations(ctx context.Context, input *ListConversationsInput) (*ListConversationsOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	total, conversations, err := uc.chat.ListConversations(ctx, userID, input.PageStats)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取会话列表失败: %v", err)
		return nil, errors.New("获取会话列表失败")
	}

	return &ListConversationsOutput{
		Conversations: conversations,
		Total:         total,
	}, nil
}

func (uc *MessageUsecase) DeleteConversation(ctx context.Context, input *DeleteConversationInput) error {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	err = uc.chat.DeleteConversation(ctx, userID, input.ConversationID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("删除会话失败: %v", err)
		return errors.New("删除会话失败")
	}

	return nil
}

// 新增：更新消息状态
func (uc *MessageUsecase) UpdateMessageStatus(ctx context.Context, messageID int64, status int32) error {
	// 验证状态值
	if status < 0 || status > 99 {
		return errors.New("无效的消息状态")
	}

	// 这里通过chat适配器调用chat服务更新状态
	err := uc.chat.UpdateMessageStatus(ctx, messageID, status)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("更新消息状态失败: %v", err)
		return errors.New("更新消息状态失败")
	}

	return nil
}

// 新增：获取会话详情
func (uc *MessageUsecase) GetConversation(ctx context.Context, targetID int64, convType int32) (*Conversation, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	conversation, err := uc.chat.GetConversation(ctx, userID, targetID, convType)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取会话详情失败: %v", err)
		return nil, errors.New("获取会话失败")
	}

	return conversation, nil
}

func (uc *MessageUsecase) GetUnreadCount(ctx context.Context, userID int64) (int64, map[int64]int64, error) {
	total, results, err := uc.chat.GetUnreadCount(ctx, userID)
	if err != nil {
		return 0, nil, err
	}
	return total, results, nil
}

func (uc *MessageUsecase) ClearMessages(ctx context.Context, userID, conversationId int64) error {
	err := uc.chat.ClearMessages(ctx, userID, conversationId)
	if err != nil {
		return err
	}
	return nil
}

// 添加创建会话方法
func (uc *MessageUsecase) CreateConversation(ctx context.Context, input *CreateConversationInput) (*CreateConversationOutput, error) {
	userID, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, errors.New("获取用户信息失败")
	}

	// 检查权限
	if input.ConvType == 0 { // 单聊
		// 检查是否是好友关系
		if uc.chat != nil {
			// 检查是否有 CheckFriendRelation 方法
			isFriend, _, err := uc.chat.CheckFriendRelation(ctx, userID, input.TargetID)
			if err != nil {
				return nil, fmt.Errorf("检查好友关系失败: %v", err)
			}
			if !isFriend {
				return nil, errors.New("你们不是好友，无法创建会话")
			}
		}
	} else if input.ConvType == 1 { // 群聊
		// 检查是否是群成员
		if uc.chat != nil {
			// 检查是否有 CheckUserRelation 方法
			checker, ok := uc.chat.(interface {
				CheckUserRelation(ctx context.Context, userID, targetID int64, convType int32) (bool, error)
			})
			if ok {
				isMember, err := checker.CheckUserRelation(ctx, userID, input.TargetID, input.ConvType)
				if err != nil {
					return nil, fmt.Errorf("检查群成员关系失败: %v", err)
				}
				if !isMember {
					return nil, errors.New("你不是群成员，无法创建会话")
				}
			}
		}
	}

	// 如果是单聊，检查是否已经存在会话
	if input.ConvType == 0 {
		existingConv, err := uc.chat.GetConversation(ctx, userID, input.TargetID, input.ConvType)
		if err == nil && existingConv != nil {
			return &CreateConversationOutput{
				ConversationID: existingConv.ID,
			}, nil
		}
	}

	// 通过 chat 适配器创建会话
	if uc.chat != nil {
		conversationID, err := uc.chat.CreateConversation(ctx, userID, input.TargetID, input.ConvType, input.InitialMessage)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("创建会话失败: %v", err)
			return nil, fmt.Errorf("创建会话失败: %v", err)
		}

		return &CreateConversationOutput{
			ConversationID: conversationID,
		}, nil
	}

	return nil, errors.New("聊天服务不可用")
}
