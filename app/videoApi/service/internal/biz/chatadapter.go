package biz

import "context"

// ChatAdapter 接口
type ChatAdapter interface {
	// 消息相关
	SendMessage(ctx context.Context, senderID, receiverID string, convType, msgType int32, content *MessageContent, clientMsgID string) (string, string, error)
	ListMessages(ctx context.Context, userID, conversationID, lastMsgID string, limit int32) ([]*Message, bool, string, error)
	RecallMessage(ctx context.Context, messageID, userID string) error
	MarkMessagesRead(ctx context.Context, userID, conversationID, lastMsgID string) error
	GetUnreadCount(ctx context.Context, userID string) (int64, map[string]int64, error)
	ListConversations(ctx context.Context, userID string, pageStats *PageStats) (int64, []*Conversation, error)
	DeleteConversation(ctx context.Context, userID, conversationID string) error
	ClearMessages(ctx context.Context, userID, conversationID string) error
	UpdateMessageStatus(ctx context.Context, messageID string, status int32) error // 新增：更新消息状态
	GetMessageByID(ctx context.Context, messageID string) (*Message, error)        // 新增：获取消息详情

	// 好友相关
	SearchUsers(ctx context.Context, keyword string, page, size int32) (int64, []*UserInfo, error)
	SendFriendApply(ctx context.Context, applicantID, receiverID, applyReason string) (string, error)
	HandleFriendApply(ctx context.Context, applyID, handlerID string, accept bool) error
	ListFriendApplies(ctx context.Context, userID string, status *int32, pageStats *PageStats) (int64, []*FriendApply, error)
	ListFriends(ctx context.Context, userID string, groupName *string, pageStats *PageStats) (int64, []*FriendInfo, error)
	DeleteFriend(ctx context.Context, userID, friendID string) error
	UpdateFriendRemark(ctx context.Context, userID, friendID, remark string) error
	SetFriendGroup(ctx context.Context, userID, friendID, groupName string) error
	CheckFriendRelation(ctx context.Context, userID, targetID string) (bool, int32, error)
	GetUserOnlineStatus(ctx context.Context, userID string) (int32, string, error)
	BatchGetUserOnlineStatus(ctx context.Context, userIDs []string) (map[string]int32, error)

	// 群聊相关
	CreateGroup(ctx context.Context, ownerID, name, notice string, addMode int32, avatar string) (string, error)
	LoadMyGroup(ctx context.Context, ownerID string, pageStats *PageStats) (int64, []*Group, error)
	CheckGroupAddMode(ctx context.Context, groupID string) (int32, error)
	EnterGroupDirectly(ctx context.Context, userID, groupID string) error
	ApplyJoinGroup(ctx context.Context, userID, groupID, applyReason string) error
	GetGroupInfo(ctx context.Context, groupID string) (*Group, error)
	ListMyJoinedGroups(ctx context.Context, userID string, pageStats *PageStats) (int64, []*Group, error)
	LeaveGroup(ctx context.Context, userID, groupID string) error
	DismissGroup(ctx context.Context, ownerID string, groupID string) error

	// 新增：获取群成员列表
	GetGroupMembers(ctx context.Context, groupID string) ([]string, error)

	// 创建会话
	CreateConversation(ctx context.Context, userID, targetID string, convType int32, initialMessage string) (string, error)
	GetConversation(ctx context.Context, userID, targetID string, convType int32) (*Conversation, error) // 新增：获取会话详情
}
