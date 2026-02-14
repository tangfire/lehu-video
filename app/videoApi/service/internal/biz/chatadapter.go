package biz

import "context"

// ChatAdapter 接口
type ChatAdapter interface {
	// 消息相关
	SendMessage(ctx context.Context, conversationID, senderID, receiverID string, convType, msgType int32, content *MessageContent, clientMsgID string) (string, string, error)
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
	SendFriendApply(ctx context.Context, applicantID, receiverID, applyReason string) (string, error)
	HandleFriendApply(ctx context.Context, applyID, handlerID string, accept bool) error
	ListFriendApplies(ctx context.Context, userID string, status *int32, pageStats *PageStats) (int64, []*FriendApply, error)
	ListFriends(ctx context.Context, userID string, groupName *string, pageStats *PageStats) (int64, []*FriendRelation, error)
	DeleteFriend(ctx context.Context, userID, friendID string) error
	UpdateFriendRemark(ctx context.Context, userID, friendID, remark string) error
	SetFriendGroup(ctx context.Context, userID, friendID, groupName string) error
	CheckFriendRelation(ctx context.Context, userID, targetID string) (bool, int32, error)

	// 在线状态相关（新增）
	GetUserOnlineStatus(ctx context.Context, userID string) (*UserSocialInfo, error)
	BatchGetUserOnlineStatus(ctx context.Context, userIDs []string) (map[string]int32, error) // 简化返回
	UpdateUserOnlineStatus(ctx context.Context, userID string, status int32, deviceType string) error

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
	GetGroupMembers(ctx context.Context, groupID string) ([]string, error)

	// 创建会话
	CreateConversation(ctx context.Context, userID, receiverID, groupID string, convType int32, initialMessage string) (string, error)
	GetConversationDetail(ctx context.Context, conversationID, userID string) (*Conversation, error)

	// 关系相关
	GetUserRelation(ctx context.Context, userID, targetUserID string) (*UserRelationInfo, error)
	BatchGetUserRelations(ctx context.Context, userID string, targetUserIDs []string) (map[string]*UserRelationInfo, error)

	HandleGroupApply(ctx context.Context, applyID, handlerID string, accept bool, replyMsg string) error
}
