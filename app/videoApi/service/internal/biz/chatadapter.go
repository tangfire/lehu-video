package biz

import "context"

// ChatAdapter 接口
type ChatAdapter interface {
	// 消息相关
	SendMessage(ctx context.Context, senderID, receiverID int64, convType, msgType int32, content *MessageContent, clientMsgID string) (int64, error)
	ListMessages(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64, limit int32) ([]*Message, bool, int64, error)
	RecallMessage(ctx context.Context, messageID, userID int64) error
	MarkMessagesRead(ctx context.Context, userID, targetID int64, convType int32, lastMsgID int64) error
	GetUnreadCount(ctx context.Context, userID int64) (int64, map[int64]int64, error)
	ListConversations(ctx context.Context, userID int64, pageStats *PageStats) (int64, []*Conversation, error)
	DeleteConversation(ctx context.Context, userID, conversationID int64) error
	ClearMessages(ctx context.Context, userID, targetID int64, convType int32) error
	UpdateMessageStatus(ctx context.Context, messageID int64, status int32) error                       // 新增：更新消息状态
	GetMessageByID(ctx context.Context, messageID int64) (*Message, error)                              // 新增：获取消息详情
	GetConversation(ctx context.Context, userID, targetID int64, convType int32) (*Conversation, error) // 新增：获取会话详情

	// 好友相关
	SearchUsers(ctx context.Context, keyword string, page, size int32) (int64, []*UserInfo, error)
	SendFriendApply(ctx context.Context, applicantID, receiverID int64, applyReason string) (int64, error)
	HandleFriendApply(ctx context.Context, applyID, handlerID int64, accept bool) error
	ListFriendApplies(ctx context.Context, userID int64, page, size int32, status *int32) (int64, []*FriendApply, error)
	ListFriends(ctx context.Context, userID int64, page, size int32, groupName *string) (int64, []*FriendInfo, error)
	DeleteFriend(ctx context.Context, userID, friendID int64) error
	UpdateFriendRemark(ctx context.Context, userID, friendID int64, remark string) error
	SetFriendGroup(ctx context.Context, userID, friendID int64, groupName string) error
	CheckFriendRelation(ctx context.Context, userID, targetID int64) (bool, int32, error)
	GetUserOnlineStatus(ctx context.Context, userID int64) (int32, string, error)
	BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]int32, error)

	// 群聊相关
	CreateGroup(ctx context.Context, ownerID int64, name, notice string, addMode int32, avatar string) (int64, error)
	LoadMyGroup(ctx context.Context, ownerID int64, pageStats *PageStats) (int64, []*Group, error)
	CheckGroupAddMode(ctx context.Context, groupID int64) (int32, error)
	EnterGroupDirectly(ctx context.Context, userID, groupID int64) error
	ApplyJoinGroup(ctx context.Context, userID, groupID int64, applyReason string) error
	GetGroupInfo(ctx context.Context, groupID int64) (*Group, error)
	ListMyJoinedGroups(ctx context.Context, userID int64, pageStats *PageStats) (int64, []*Group, error)
	LeaveGroup(ctx context.Context, userID, groupID int64) error
	DismissGroup(ctx context.Context, ownerID, groupID int64) error

	// 新增：获取群成员列表
	GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error)

	// 新增：检查用户关系
	CheckUserRelation(ctx context.Context, userID, targetID int64, convType int32) (bool, error)
}
