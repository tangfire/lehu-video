package data

import (
	"context"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
	"time"
)

// 创建会话
func (r *chatAdapterImpl) CreateConversation(ctx context.Context, userID, targetID string, convType int32, initialMessage string) (string, error) {
	req := &chat.CreateConversationReq{
		UserId:         userID,
		TargetId:       targetID,
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

// 获取会话详情
func (r *chatAdapterImpl) GetConversation(ctx context.Context, userID, targetID string, convType int32) (*biz.Conversation, error) {
	req := &chat.GetConversationReq{
		UserId:   userID,
		TargetId: targetID,
		ConvType: chat.ConversationType(convType),
	}

	resp, err := r.message.GetConversation(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}
	c := resp.Conversation
	if c == nil {
		return nil, nil
	}

	var lastMsgTime *time.Time
	if c.LastMsgTime != nil && *c.LastMsgTime > 0 {
		t := time.Unix(*c.LastMsgTime, 0)
		lastMsgTime = &t
	}

	var targetIDPtr *string
	if c.TargetId != nil {
		targetIDPtr = c.TargetId
	}

	var groupIDPtr *string
	if c.GroupId != nil {
		groupIDPtr = c.GroupId
	}

	var lastMsgTypePtr *int32
	if c.LastMsgType != nil && *c.LastMsgType != 0 {
		msgType := int32(*c.LastMsgType)
		lastMsgTypePtr = &msgType
	}

	return &biz.Conversation{
		ID:          c.Id,
		Type:        int32(c.Type),
		TargetID:    targetIDPtr,
		GroupID:     groupIDPtr,
		Name:        c.Name,
		Avatar:      c.Avatar,
		LastMessage: c.LastMessage,
		LastMsgType: lastMsgTypePtr,
		LastMsgTime: lastMsgTime,
		MemberCount: int32(c.MemberCount),
		CreatedAt:   parseTime(c.CreatedAt),
		UpdatedAt:   parseTime(c.UpdatedAt),
	}, nil
}
