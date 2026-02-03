package data

import (
	"context"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
	"time"
)

// 创建会话
func (r *chatAdapterImpl) CreateConversation(ctx context.Context, userID, receiverID, groupID string, convType int32, initialMessage string) (string, error) {
	req := &chat.CreateConversationReq{
		UserIds:        []string{userID, receiverID},
		GroupId:        groupID,
		ConvType:       chat.ConversationType(convType),
		InitialMessage: initialMessage,
	}
	resp, err := r.message.CreateConversation(ctx, req)
	if err != nil {
		return "0", err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return "0", err
	}
	return resp.ConversationId, nil
}

func (r *chatAdapterImpl) ListConversations(ctx context.Context, userID string, pageStats *biz.PageStats) (int64, []*biz.Conversation, error) {
	req := &chat.ListConversationsReq{
		UserId: userID,
		PageStats: &chat.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
			Sort: pageStats.Sort,
		},
	}

	resp, err := r.message.ListConversations(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	var conversations []*biz.Conversation
	for _, c := range resp.Conversations {
		conversations = append(conversations, &biz.Conversation{
			ID:          c.Conversation.Id,
			Type:        int32(c.Conversation.Type),
			GroupID:     &c.Conversation.GroupId,
			Name:        c.Conversation.Name,
			Avatar:      c.Conversation.Avatar,
			LastMessage: c.Conversation.LastMessage,
			LastMsgType: (*int32)(c.Conversation.LastMsgType),
			LastMsgTime: parseUnixPointer(c.Conversation.LastMsgTime),
			MemberCount: int32(c.Conversation.MemberCount),
			MemberIDs:   c.Conversation.MemberIds,
			UnreadCount: c.UnreadCount,
			IsPinned:    c.IsPinned,
			IsMuted:     c.IsMuted,
			CreatedAt:   parseTime(c.Conversation.CreatedAt),
			UpdatedAt:   parseTime(c.Conversation.UpdatedAt),
		})
	}

	return int64(resp.PageStats.Total), conversations, nil
}

func (r *chatAdapterImpl) GetConversationDetail(ctx context.Context, conversationID, userID string) (*biz.Conversation, error) {
	resp, err := r.message.GetConversation(ctx, &chat.GetConversationReq{
		ConversationId: conversationID,
		UserId:         userID,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	conv := resp.ConversationView.Conversation
	return &biz.Conversation{
		ID:          conv.Id,
		Type:        int32(conv.Type),
		GroupID:     &conv.GroupId,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		LastMessage: conv.LastMessage,
		LastMsgType: (*int32)(conv.LastMsgType),
		LastMsgTime: parseUnixPointer(conv.LastMsgTime),
		UnreadCount: resp.ConversationView.UnreadCount,
		MemberCount: int32(conv.MemberCount),
		MemberIDs:   conv.MemberIds,
		CreatedAt:   parseTime(conv.CreatedAt),
		UpdatedAt:   parseTime(conv.UpdatedAt),
	}, nil

}

func parseUnixPointer(ts *int64) *time.Time {
	if ts == nil || *ts == 0 {
		return nil
	}
	t := time.Unix(*ts, 0)
	return &t
}
