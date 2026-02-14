package data

import (
	"context"
	"errors"
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

// CreateConversation 创建会话
func (r *conversationRepo) CreateConversation(ctx context.Context, conv *biz.Conversation) (int64, error) {
	dbConv := model.Conversation{
		ID:          conv.ID,
		Type:        int8(conv.Type),
		GroupID:     conv.GroupID,
		Name:        conv.Name,
		Avatar:      conv.Avatar,
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

// GetConversation 获取会话
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

// GetSingleChatConversation 获取单聊会话
func (r *conversationRepo) GetSingleChatConversation(ctx context.Context, userID1, userID2 int64) (*biz.Conversation, error) {
	// 查找两个用户之间的单聊会话
	// 由于我们使用唯一的(target_id)来标识单聊会话，需要确保查询顺序一致
	// 这里假设target_id是较小的那个用户ID（可以在业务层确保）

	minID := userID1
	maxID := userID2
	if userID1 > userID2 {
		minID, maxID = userID2, userID1
	}

	var dbConv model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("type = ? AND target_id = ? AND is_deleted = ?", 0, maxID, false).
		First(&dbConv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 尝试另一种顺序
		err = r.data.db.WithContext(ctx).
			Where("type = ? AND target_id = ? AND is_deleted = ?", 0, minID, false).
			First(&dbConv).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}
	if err != nil {
		return nil, err
	}

	// 还需要检查这个会话是否包含这两个用户
	// 这里简化处理，假设target_id保存的是对方用户ID
	return r.toBizConversation(&dbConv), nil
}

// GetOrCreateSingleChatConversation 获取或创建单聊会话
func (r *conversationRepo) GetOrCreateSingleChatConversation(
	ctx context.Context,
	userID1, userID2 int64,
) (*biz.Conversation, error) {

	conv, err := r.GetSingleChatConversation(ctx, userID1, userID2)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	conv = &biz.Conversation{
		Type:   biz.ConvTypeSingle,
		Name:   "",
		Avatar: "",
	}

	convID, err := r.CreateConversation(ctx, conv)
	if err != nil {
		return nil, err
	}
	conv.ID = convID

	// 添加双方成员
	_ = r.AddConversationMember(ctx, &biz.ConversationMember{
		ConversationID: convID,
		UserID:         userID1,
	})
	_ = r.AddConversationMember(ctx, &biz.ConversationMember{
		ConversationID: convID,
		UserID:         userID2,
	})

	return conv, nil
}

// GetGroupConversation 获取群聊会话
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

// GetOrCreateGroupConversation 获取或创建群聊会话
func (r *conversationRepo) GetOrCreateGroupConversation(
	ctx context.Context,
	groupID int64,
) (*biz.Conversation, error) {

	conv, err := r.GetGroupConversation(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	conv = &biz.Conversation{
		Type:    biz.ConvTypeGroup,
		GroupID: groupID,
	}

	convID, err := r.CreateConversation(ctx, conv)
	if err != nil {
		return nil, err
	}
	conv.ID = convID

	// ⚠️ 群成员添加：这里通常由 GroupService 负责
	return conv, nil
}

// UpdateConversationLastMsg 更新会话最后一条消息
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

// DeleteConversation 删除会话（软删除）
func (r *conversationRepo) DeleteConversation(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}

// AddConversationMember 添加会话成员
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

// RemoveConversationMember 移除会话成员
func (r *conversationRepo) RemoveConversationMember(ctx context.Context, conversationID, userID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}

// GetConversationMember 获取会话成员
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

// GetConversationMembers 获取会话所有成员
func (r *conversationRepo) GetConversationMembers(ctx context.Context, conversationID int64) ([]*biz.ConversationMember, error) {
	var dbMembers []*model.ConversationMember
	err := r.data.db.WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Find(&dbMembers).Error

	if err != nil {
		return nil, err
	}

	members := make([]*biz.ConversationMember, 0, len(dbMembers))
	for _, dbMember := range dbMembers {
		members = append(members, r.toBizConversationMember(dbMember))
	}

	return members, nil
}

// GetConversationMemberCount 获取会话成员数量
func (r *conversationRepo) GetConversationMemberCount(ctx context.Context, conversationID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Count(&count).Error

	return count, err
}

// UpdateMemberUnreadCount 更新成员未读计数
func (r *conversationRepo) UpdateMemberUnreadCount(ctx context.Context, conversationID, userID int64, delta int) error {
	if delta == 0 {
		return nil
	}

	var expr string
	if delta > 0 {
		expr = "unread_count + ?"
	} else {
		expr = "GREATEST(unread_count - ?, 0)" // 确保不小于0
	}

	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"unread_count": gorm.Expr(expr, abs(delta)),
			"updated_at":   time.Now(),
		}).Error
}

// ResetMemberUnreadCount 重置成员未读计数
func (r *conversationRepo) ResetMemberUnreadCount(ctx context.Context, conversationID, userID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"updated_at":   time.Now(),
		}).Error
}

// UpdateMemberLastRead 更新成员最后已读消息ID
func (r *conversationRepo) UpdateMemberLastRead(ctx context.Context, conversationID, userID, lastReadMsgID int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"last_read_msg_id": lastReadMsgID,
			"updated_at":       time.Now(),
		}).Error
}

// UpdateMemberSettings 更新成员设置
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

// GetUserTotalUnreadCount 获取用户总未读数
func (r *conversationRepo) GetUserTotalUnreadCount(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.data.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Select("SUM(unread_count)").
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Scan(&total).Error

	if err != nil {
		return 0, err
	}

	return total, nil
}

// GetUserConversationUnreadCount 获取用户各会话未读数
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

	resultMap := make(map[int64]int64)
	for _, res := range results {
		resultMap[res.ConversationID] = res.UnreadCount
	}

	return resultMap, nil
}

func (r *conversationRepo) ListConversationMembers(
	ctx context.Context,
	userID int64,
	page *biz.PageStats,
) ([]*biz.ConversationMember, int64, error) {

	db := r.data.db.WithContext(ctx)

	// 分页参数
	offset := 0
	limit := 20
	if page != nil {
		if page.Page > 0 && page.PageSize > 0 {
			offset = (page.Page - 1) * page.PageSize
			limit = page.PageSize
		}
	}

	// 1️⃣ 查成员列表
	var dbMembers []*model.ConversationMember
	err := db.
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("is_pinned DESC, updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&dbMembers).Error

	if err != nil {
		return nil, 0, err
	}

	// 2️⃣ 查总数
	var total int64
	err = db.
		Model(&model.ConversationMember{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error

	if err != nil {
		return nil, 0, err
	}

	// 3️⃣ 转 biz
	members := make([]*biz.ConversationMember, 0, len(dbMembers))
	for _, m := range dbMembers {
		members = append(members, r.toBizConversationMember(m))
	}

	return members, total, nil
}

func (r *conversationRepo) GetConversationsByIDs(
	ctx context.Context,
	ids []int64,
) ([]*biz.Conversation, error) {

	if len(ids) == 0 {
		return []*biz.Conversation{}, nil
	}

	var dbConvs []*model.Conversation
	err := r.data.db.WithContext(ctx).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Find(&dbConvs).Error

	if err != nil {
		return nil, err
	}

	convs := make([]*biz.Conversation, 0, len(dbConvs))
	for _, dbConv := range dbConvs {
		convs = append(convs, r.toBizConversation(dbConv))
	}

	return convs, nil
}

// 辅助函数
// toBizConversation 将数据库会话转换为业务会话（修复版）
func (r *conversationRepo) toBizConversation(dbConv *model.Conversation) *biz.Conversation {
	conv := &biz.Conversation{
		ID:          dbConv.ID,
		Type:        int32(dbConv.Type),
		GroupID:     dbConv.GroupID,
		Name:        dbConv.Name,
		Avatar:      dbConv.Avatar,
		LastMessage: dbConv.LastMessage,
		LastMsgType: dbConv.LastMsgType,
		LastMsgTime: dbConv.LastMsgTime,
		MemberCount: dbConv.MemberCount,
		CreatedAt:   dbConv.CreatedAt,
		UpdatedAt:   dbConv.UpdatedAt,
		IsDeleted:   dbConv.IsDeleted,
	}

	// 处理指针字段

	if dbConv.GroupID != 0 {
		groupID := dbConv.GroupID
		conv.GroupID = groupID
	}

	if dbConv.LastMsgType != nil {
		lastMsgType := *dbConv.LastMsgType
		conv.LastMsgType = &lastMsgType
	}

	if dbConv.LastMsgTime != nil {
		conv.LastMsgTime = dbConv.LastMsgTime
	}

	return conv
}

// toBizConversationMember 将数据库会话成员转换为业务会话成员（修复版）
func (r *conversationRepo) toBizConversationMember(dbMember *model.ConversationMember) *biz.ConversationMember {
	return &biz.ConversationMember{
		ID:             dbMember.ID,
		ConversationID: dbMember.ConversationID,
		UserID:         dbMember.UserID,
		Type:           int32(dbMember.Type),
		UnreadCount:    dbMember.UnreadCount,
		LastReadMsgID:  dbMember.LastReadMsgID,
		IsPinned:       dbMember.IsPinned,
		IsMuted:        dbMember.IsMuted,
		JoinTime:       dbMember.JoinTime,
		CreatedAt:      dbMember.CreatedAt,
		UpdatedAt:      dbMember.UpdatedAt,
		IsDeleted:      dbMember.IsDeleted,
	}
}

// GetConversationMembersByConversationIDs 批量查询会话成员
func (r *conversationRepo) GetConversationMembersByConversationIDs(
	ctx context.Context,
	conversationIDs []int64,
) (map[int64][]*biz.ConversationMember, error) {
	if len(conversationIDs) == 0 {
		return make(map[int64][]*biz.ConversationMember), nil
	}

	// 使用 where in 查询所有相关成员
	var dbMembers []*model.ConversationMember
	err := r.data.db.WithContext(ctx).
		Where("conversation_id IN ? AND is_deleted = ?", conversationIDs, false).
		Order("conversation_id, id ASC"). // 可以按需排序
		Find(&dbMembers).Error

	if err != nil {
		return nil, err
	}

	// 构造 map: conversationID -> []*ConversationMember
	result := make(map[int64][]*biz.ConversationMember)

	for _, dbMember := range dbMembers {
		bizMember := r.toBizConversationMember(dbMember)
		result[bizMember.ConversationID] = append(result[bizMember.ConversationID], bizMember)
	}

	return result, nil
}

// GetConversationView 获取带用户状态的会话视图
func (r *conversationRepo) GetConversationView(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (*biz.ConversationView, error) {
	// 1. 获取会话基础信息
	conv, err := r.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, nil
	}

	// 2. 获取会话成员列表
	members, err := r.GetConversationMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// 3. 获取当前用户在会话中的状态
	member, err := r.GetConversationMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	// 4. 提取成员ID列表
	memberIDs := make([]int64, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.UserID)
	}

	// 5. 构建 ConversationView
	view := &biz.ConversationView{
		// 会话基础信息
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

	// 6. 添加用户状态（如果用户是会话成员）
	if member != nil {
		view.UnreadCount = int64(member.UnreadCount)
		view.IsPinned = member.IsPinned
		view.IsMuted = member.IsMuted
	}

	return view, nil
}

// UpdateConversationMemberCount 更新会话成员数量（增加或减少）
func (r *conversationRepo) UpdateConversationMemberCount(ctx context.Context, conversationID int64, delta int) error {
	if delta == 0 {
		return nil
	}

	var expr string
	if delta > 0 {
		expr = "member_count + ?"
	} else {
		expr = "GREATEST(member_count - ?, 0)" // 确保不小于0
	}

	return r.data.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"member_count": gorm.Expr(expr, abs(delta)),
			"updated_at":   time.Now(),
		}).Error
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
