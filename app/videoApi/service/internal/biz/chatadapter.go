package biz

import "context"

type ChatAdapter interface {
	// 创建群聊
	CreateGroup(ctx context.Context, ownerID int64, name, notice string, addMode int32, avatar string) (int64, error)

	// 获取我创建的群聊
	LoadMyGroup(ctx context.Context, ownerID int64, pageStats *PageStats) (int64, []*Group, error)

	// 检查群聊加群方式
	CheckGroupAddMode(ctx context.Context, groupID int64) (int32, error)

	// 直接进群
	EnterGroupDirectly(ctx context.Context, userID, groupID int64) error

	// 申请加群
	ApplyJoinGroup(ctx context.Context, userID, groupID int64, applyReason string) error

	// 退群
	LeaveGroup(ctx context.Context, userID, groupID int64) error

	// 解散群聊
	DismissGroup(ctx context.Context, ownerID, groupID int64) error

	// 获取群聊信息
	GetGroupInfo(ctx context.Context, groupID int64) (*Group, error)

	// 获取我加入的群聊
	ListMyJoinedGroups(ctx context.Context, userID int64, pageStats *PageStats) (int64, []*Group, error)

	SendMessage(ctx context.Context, senderID, receiverID int64, convType, msgType int32, content *MessageContent, clientMsgID string) (int64, error)

	ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int32) ([]*Message, bool, int64, error)

	RecallMessage(ctx context.Context, messageID, userID int64) error

	MarkMessagesRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error

	DeleteConversation(ctx context.Context, userID, conversationID int64) error

	ListConversations(ctx context.Context, userID int64, pageStats *PageStats) (int64, []*Conversation, error)
}
