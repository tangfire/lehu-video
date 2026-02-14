package data

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/data/model"
)

type conversationRepo struct {
	data *Data
	log  *log.Helper
}

func NewConversationRepo(data *Data, logger log.Logger) biz.ConversationRepo {
	return &conversationRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *conversationRepo) CreateConversation(ctx context.Context, conv *biz.Conversation) (int64, error) {
	dbConv := model.Conversation{
		ID:          conv.ID,
		Type:        int8(conv.Type),
		GroupID:     conv.GroupID,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		LastMessage: conv.LastMessage,
		LastMsgType: conv.LastMsgType,
		LastMsgTime: conv.LastMsgTime,
		MemberCount: conv.MemberCount,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsDeleted:   false,
	}
	if err := r.data.db.WithContext(ctx).Create(&dbConv).Error; err != nil {
		return 0, err
	}
	return dbConv.ID, nil
}

func (r *conversationRepo) GetConversation(ctx context.Context, id int64) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbConv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizConversation(&dbConv), nil
}

// GetSingleChatConversation 修复：通过成员表关联查询
func (r *conversationRepo) GetSingleChatConversation(ctx context.Context, userID1, userID2 int64) (*biz.Conversation, error) {
	var conv model.Conversation
	err := r.data.db.WithContext(ctx).Raw(`
		SELECT c.* FROM conversation c
		INNER JOIN conversation_member m1 ON c.id = m1.conversation_id AND m1.user_id = ? AND m1.is_deleted = 0
		INNER JOIN conversation_member m2 ON c.id = m2.conversation_id AND m2.user_id = ? AND m2.is_deleted = 0
		WHERE c.type = 0 AND c.is_deleted = 0
		LIMIT 1
	`, userID1, userID2).Scan(&conv).Error
	if err != nil {
		return nil, err
	}
	if conv.ID == 0 {
		return nil, nil
	}
	return r.toBizConversation(&conv), nil
}

func (r *conversationRepo) GetOrCreateSingleChatConversation(ctx context.Context, userID1, userID2 int64) (*biz.Conversation, error) {
	conv, err := r.GetSingleChatConversation(ctx, userID1, userID2)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}
	// 创建新会话
	newID := int64(uuid.New().ID())
	now := time.Now()
	conv = &biz.Conversation{
		ID:          newID,
		Type:        biz.ConvTypeSingle,
		MemberCount: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := r.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	// 添加成员
	members := []*biz.ConversationMember{
		{ConversationID: newID, UserID: userID1, Type: 0, JoinTime: now, CreatedAt: now, UpdatedAt: now},
		{ConversationID: newID, UserID: userID2, Type: 0, JoinTime: now, CreatedAt: now, UpdatedAt: now},
	}
	for _, m := range members {
		if err := r.AddConversationMember(ctx, m); err != nil {
			return nil, err
		}
	}
	return conv, nil
}

func (r *conversationRepo) GetGroupConversation(ctx context.Context, groupID int64) (*biz.Conversation, error) {
	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("type = ? AND group_id = ? AND is_deleted = ?", 1, groupID, false).
		First(&dbConv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizConversation(&dbConv), nil
}

func (r *conversationRepo) GetOrCreateGroupConversation(ctx context.Context, groupID int64) (*biz.Conversation, error) {
	conv, err := r.GetGroupConversation(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}
	// 创建新群会话
	newID := int64(uuid.New().ID())
	now := time.Now()
	conv = &biz.Conversation{
		ID:          newID,
		Type:        biz.ConvTypeGroup,
		GroupID:     groupID,
		MemberCount: 1, // 至少群主，后续会添加所有成员
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := r.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (r *conversationRepo) UpdateConversationLastMsg(ctx context.Context, conversationID int64, lastMessage string, lastMsgType int32) error {
	now := time.Now()
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message":  lastMessage,
			"last_msg_type": int8(lastMsgType),
			"last_msg_time": now,
			"updated_at":    now,
		}).Error
}

func (r *conversationRepo) DeleteConversation(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}

func (r *conversationRepo) AddConversationMember(ctx context.Context, member *biz.ConversationMember) error {
	dbMember := model.ConversationMember{
		ConversationID: member.ConversationID,
		UserID:         member.UserID,
		Type:           int8(member.Type),
		UnreadCount:    member.UnreadCount,
		LastReadMsgID:  member.LastReadMsgID,
		IsPinned:       member.IsPinned,
		IsMuted:        member.IsMuted,
		JoinTime:       member.JoinTime,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		IsDeleted:      false,
	}
	return r.data.db.WithContext(ctx).Create(&dbMember).Error
}

func (r *conversationRepo) RemoveConversationMember(ctx context.Context, conversationID, userID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}

func (r *conversationRepo) GetConversationMember(ctx context.Context, conversationID, userID int64) (*biz.ConversationMember, error) {
	var dbMember model.ConversationMember
	err := r.data.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ? AND is_deleted = ?", conversationID, userID, false).
		First(&dbMember).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizConversationMember(&dbMember), nil
}

func (r *conversationRepo) GetConversationMembers(ctx context.Context, conversationID int64) ([]*biz.ConversationMember, error) {
	var dbMembers []*model.ConversationMember
	err := r.data.db.WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Find(&dbMembers).Error
	if err != nil {
		return nil, err
	}
	members := make([]*biz.ConversationMember, 0, len(dbMembers))
	for _, m := range dbMembers {
		members = append(members, r.toBizConversationMember(m))
	}
	return members, nil
}

func (r *conversationRepo) GetConversationMemberCount(ctx context.Context, conversationID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Count(&count).Error
	return count, err
}

func (r *conversationRepo) UpdateMemberUnreadCount(ctx context.Context, conversationID, userID int64, delta int) error {
	if delta == 0 {
		return nil
	}
	expr := "unread_count + ?"
	if delta < 0 {
		expr = "GREATEST(unread_count - ?, 0)"
	}
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"unread_count": gorm.Expr(expr, abs(delta)),
			"updated_at":   time.Now(),
		}).Error
}

func (r *conversationRepo) ResetMemberUnreadCount(ctx context.Context, conversationID, userID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"updated_at":   time.Now(),
		}).Error
}

func (r *conversationRepo) UpdateMemberLastRead(ctx context.Context, conversationID, userID, lastReadMsgID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"last_read_msg_id": lastReadMsgID,
			"updated_at":       time.Now(),
		}).Error
}

func (r *conversationRepo) UpdateMemberSettings(ctx context.Context, conversationID, userID int64, isPinned, isMuted bool) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"is_pinned":  isPinned,
			"is_muted":   isMuted,
			"updated_at": time.Now(),
		}).Error
}

func (r *conversationRepo) GetUserTotalUnreadCount(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Select("COALESCE(SUM(unread_count), 0)").
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Scan(&total).Error
	return total, err
}

func (r *conversationRepo) GetUserConversationUnreadCount(ctx context.Context, userID int64) (map[int64]int64, error) {
	type Result struct {
		ConversationID int64
		UnreadCount    int64
	}
	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Select("conversation_id, unread_count").
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int64]int64, len(results))
	for _, r := range results {
		m[r.ConversationID] = r.UnreadCount
	}
	return m, nil
}

func (r *conversationRepo) ListConversationMembers(ctx context.Context, userID int64, page *biz.PageStats) ([]*biz.ConversationMember, int64, error) {
	db := r.data.db.WithContext(ctx)
	offset, limit := 0, 20
	if page != nil {
		if page.Page > 0 && page.PageSize > 0 {
			offset = (page.Page - 1) * page.PageSize
			limit = page.PageSize
		}
	}
	var total int64
	if err := db.Model(&model.ConversationMember{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var dbMembers []*model.ConversationMember
	err := db.
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("is_pinned DESC, updated_at DESC").
		Offset(offset).Limit(limit).
		Find(&dbMembers).Error
	if err != nil {
		return nil, 0, err
	}
	members := make([]*biz.ConversationMember, 0, len(dbMembers))
	for _, m := range dbMembers {
		members = append(members, r.toBizConversationMember(m))
	}
	return members, total, nil
}

func (r *conversationRepo) GetConversationsByIDs(ctx context.Context, ids []int64) ([]*biz.Conversation, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var dbConvs []*model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Find(&dbConvs).Error
	if err != nil {
		return nil, err
	}
	convs := make([]*biz.Conversation, 0, len(dbConvs))
	for _, c := range dbConvs {
		convs = append(convs, r.toBizConversation(c))
	}
	return convs, nil
}

func (r *conversationRepo) GetConversationMembersByConversationIDs(ctx context.Context, convIDs []int64) (map[int64][]*biz.ConversationMember, error) {
	if len(convIDs) == 0 {
		return map[int64][]*biz.ConversationMember{}, nil
	}
	var dbMembers []*model.ConversationMember
	err := r.data.db.WithContext(ctx).
		Where("conversation_id IN ? AND is_deleted = ?", convIDs, false).
		Find(&dbMembers).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64][]*biz.ConversationMember)
	for _, m := range dbMembers {
		bizM := r.toBizConversationMember(m)
		result[bizM.ConversationID] = append(result[bizM.ConversationID], bizM)
	}
	return result, nil
}

func (r *conversationRepo) GetConversationView(ctx context.Context, conversationID, userID int64) (*biz.ConversationView, error) {
	conv, err := r.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, nil
	}
	members, err := r.GetConversationMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	memberIDs := make([]int64, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.UserID)
	}
	member, _ := r.GetConversationMember(ctx, conversationID, userID)
	view := &biz.ConversationView{
		ID:          conv.ID,
		Type:        conv.Type,
		GroupID:     conv.GroupID,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		LastMessage: conv.LastMessage,
		LastMsgType: conv.LastMsgType,
		LastMsgTime: conv.LastMsgTime,
		MemberCount: conv.MemberCount,
		MemberIDs:   memberIDs,
		CreatedAt:   conv.CreatedAt,
		UpdatedAt:   conv.UpdatedAt,
	}
	if member != nil {
		view.UnreadCount = int64(member.UnreadCount)
		view.IsPinned = member.IsPinned
		view.IsMuted = member.IsMuted
	}
	return view, nil
}

func (r *conversationRepo) UpdateConversationMemberCount(ctx context.Context, conversationID int64, delta int) error {
	if delta == 0 {
		return nil
	}
	expr := "member_count + ?"
	if delta < 0 {
		expr = "GREATEST(member_count - ?, 0)"
	}
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"member_count": gorm.Expr(expr, abs(delta)),
			"updated_at":   time.Now(),
		}).Error
}

// 辅助方法
func (r *conversationRepo) toBizConversation(db *model.Conversation) *biz.Conversation {
	return &biz.Conversation{
		ID:          db.ID,
		Type:        int32(db.Type),
		GroupID:     db.GroupID,
		Name:        db.Name,
		Avatar:      db.Avatar,
		LastMessage: db.LastMessage,
		LastMsgType: db.LastMsgType,
		LastMsgTime: db.LastMsgTime,
		MemberCount: db.MemberCount,
		CreatedAt:   db.CreatedAt,
		UpdatedAt:   db.UpdatedAt,
		IsDeleted:   db.IsDeleted,
	}
}

func (r *conversationRepo) toBizConversationMember(db *model.ConversationMember) *biz.ConversationMember {
	return &biz.ConversationMember{
		ID:             db.ID,
		ConversationID: db.ConversationID,
		UserID:         db.UserID,
		Type:           int32(db.Type),
		UnreadCount:    db.UnreadCount,
		LastReadMsgID:  db.LastReadMsgID,
		IsPinned:       db.IsPinned,
		IsMuted:        db.IsMuted,
		JoinTime:       db.JoinTime,
		CreatedAt:      db.CreatedAt,
		UpdatedAt:      db.UpdatedAt,
		IsDeleted:      db.IsDeleted,
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
