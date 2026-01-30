package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/data/model"
)

type groupRepo struct {
	data *Data
	log  *log.Helper
}

func NewGroupRepo(data *Data, logger log.Logger) biz.GroupRepo {
	return &groupRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *groupRepo) GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error) {
	var memberIDs []int64

	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Select("user_id").
		Where("group_id = ? AND is_deleted = ?", groupID, false).
		Order("role DESC, join_time ASC").
		Pluck("user_id", &memberIDs).Error

	if err != nil {
		return nil, err
	}

	return memberIDs, nil
}

func (r *groupRepo) CreateGroup(ctx context.Context, group *biz.Group) error {
	dbGroup := model.GroupInfo{
		Id:        group.ID,
		Name:      group.Name,
		Notice:    group.Notice,
		MemberCnt: group.MemberCnt,
		OwnerId:   group.OwnerID,
		AddMode:   int8(group.AddMode),
		Avatar:    group.Avatar,
		Status:    int8(group.Status),
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
		IsDeleted: false,
	}

	return r.data.db.WithContext(ctx).Create(&dbGroup).Error
}

func (r *groupRepo) GetGroupByID(ctx context.Context, id int64) (*biz.Group, error) {
	var dbGroup model.GroupInfo
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbGroup).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 不再从 members 字段反序列化，而是从 GroupMember 表查询
	return &biz.Group{
		ID:        dbGroup.Id,
		Name:      dbGroup.Name,
		Notice:    dbGroup.Notice,
		MemberCnt: dbGroup.MemberCnt,
		OwnerID:   dbGroup.OwnerId,
		AddMode:   int32(dbGroup.AddMode),
		Avatar:    dbGroup.Avatar,
		Status:    int32(dbGroup.Status),
		CreatedAt: dbGroup.CreatedAt,
		UpdatedAt: dbGroup.UpdatedAt,
	}, nil
}

func (r *groupRepo) GetGroupWithMembers(ctx context.Context, id int64) (*biz.Group, []int64, error) {
	// 获取群组基本信息
	group, err := r.GetGroupByID(ctx, id)
	if err != nil || group == nil {
		return nil, nil, err
	}

	// 获取成员ID列表
	memberIDs, err := r.GetGroupMemberIDs(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return group, memberIDs, nil
}

func (r *groupRepo) GetGroupMemberIDs(ctx context.Context, groupID int64) ([]int64, error) {
	var memberIDs []int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND is_deleted = ?", groupID, false).
		Pluck("user_id", &memberIDs).Error

	return memberIDs, err
}

func (r *groupRepo) GetGroupByOwnerAndID(ctx context.Context, ownerID, id int64) (*biz.Group, error) {
	var dbGroup model.GroupInfo
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND owner_id = ? AND is_deleted = ?", id, ownerID, false).
		First(&dbGroup).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.Group{
		ID:        dbGroup.Id,
		Name:      dbGroup.Name,
		Notice:    dbGroup.Notice,
		MemberCnt: dbGroup.MemberCnt,
		OwnerID:   dbGroup.OwnerId,
		AddMode:   int32(dbGroup.AddMode),
		Avatar:    dbGroup.Avatar,
		Status:    int32(dbGroup.Status),
		CreatedAt: dbGroup.CreatedAt,
		UpdatedAt: dbGroup.UpdatedAt,
	}, nil
}

func (r *groupRepo) UpdateGroup(ctx context.Context, group *biz.Group) error {
	return r.data.db.WithContext(ctx).
		Model(&model.GroupInfo{}).
		Where("id = ?", group.ID).
		Updates(map[string]interface{}{
			"name":       group.Name,
			"notice":     group.Notice,
			"member_cnt": group.MemberCnt,
			"add_mode":   group.AddMode,
			"avatar":     group.Avatar,
			"status":     group.Status,
			"updated_at": time.Now(),
		}).Error
}

func (r *groupRepo) DeleteGroup(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.GroupInfo{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

func (r *groupRepo) ListGroupsByOwner(ctx context.Context, ownerID int64, offset, limit int) ([]*biz.Group, error) {
	var dbGroups []*model.GroupInfo

	query := r.data.db.WithContext(ctx).
		Where("owner_id = ? AND is_deleted = ?", ownerID, false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbGroups).Error
	if err != nil {
		return nil, err
	}

	groups := make([]*biz.Group, 0, len(dbGroups))
	for _, dbGroup := range dbGroups {
		groups = append(groups, &biz.Group{
			ID:        dbGroup.Id,
			Name:      dbGroup.Name,
			Notice:    dbGroup.Notice,
			MemberCnt: dbGroup.MemberCnt,
			OwnerID:   dbGroup.OwnerId,
			AddMode:   int32(dbGroup.AddMode),
			Avatar:    dbGroup.Avatar,
			Status:    int32(dbGroup.Status),
			CreatedAt: dbGroup.CreatedAt,
			UpdatedAt: dbGroup.UpdatedAt,
		})
	}

	return groups, nil
}

func (r *groupRepo) CountGroupsByOwner(ctx context.Context, ownerID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupInfo{}).
		Where("owner_id = ? AND is_deleted = ?", ownerID, false).
		Count(&count).Error

	return count, err
}

func (r *groupRepo) CreateGroupMember(ctx context.Context, member *biz.GroupMember) error {
	dbMember := model.GroupMember{
		Id:        member.ID,
		UserId:    member.UserID,
		GroupId:   member.GroupID,
		Role:      int8(member.Role),
		JoinTime:  member.JoinTime,
		IsDeleted: false,
	}

	return r.data.db.WithContext(ctx).Create(&dbMember).Error
}

func (r *groupRepo) GetGroupMember(ctx context.Context, groupID, userID int64) (*biz.GroupMember, error) {
	var dbMember model.GroupMember
	err := r.data.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ? AND is_deleted = ?", groupID, userID, false).
		First(&dbMember).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.GroupMember{
		ID:       dbMember.Id,
		UserID:   dbMember.UserId,
		GroupID:  dbMember.GroupId,
		Role:     int32(dbMember.Role),
		JoinTime: dbMember.JoinTime,
	}, nil
}

func (r *groupRepo) UpdateGroupMember(ctx context.Context, member *biz.GroupMember) error {
	return r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("id = ?", member.ID).
		Updates(map[string]interface{}{
			"role":       member.Role,
			"is_deleted": false,
		}).Error
}

func (r *groupRepo) DeleteGroupMember(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

func (r *groupRepo) ListGroupMembers(ctx context.Context, groupID int64, offset, limit int) ([]*biz.GroupMember, error) {
	var dbMembers []*model.GroupMember

	query := r.data.db.WithContext(ctx).
		Where("group_id = ? AND is_deleted = ?", groupID, false).
		Order("role DESC, join_time ASC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbMembers).Error
	if err != nil {
		return nil, err
	}

	members := make([]*biz.GroupMember, 0, len(dbMembers))
	for _, dbMember := range dbMembers {
		members = append(members, &biz.GroupMember{
			ID:       dbMember.Id,
			UserID:   dbMember.UserId,
			GroupID:  dbMember.GroupId,
			Role:     int32(dbMember.Role),
			JoinTime: dbMember.JoinTime,
		})
	}

	return members, nil
}

func (r *groupRepo) CountGroupMembers(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND is_deleted = ?", groupID, false).
		Count(&count).Error

	return count, err
}

func (r *groupRepo) IsGroupMember(ctx context.Context, groupID, userID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ? AND is_deleted = ?", groupID, userID, false).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *groupRepo) IsGroupOwner(ctx context.Context, groupID, userID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ? AND role = ? AND is_deleted = ?",
			groupID, userID, 2, false).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *groupRepo) IsGroupAdmin(ctx context.Context, groupID, userID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ? AND role = ? AND is_deleted = ?",
			groupID, userID, 1, false).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *groupRepo) CreateGroupApply(ctx context.Context, apply *biz.GroupApply) error {
	dbApply := model.GroupApply{
		Id:          apply.ID,
		UserId:      apply.UserID,
		GroupId:     apply.GroupID,
		ApplyReason: apply.ApplyReason,
		Status:      int8(apply.Status),
		HandlerId:   apply.HandlerID,
		ReplyMsg:    apply.ReplyMsg,
		CreatedAt:   apply.CreatedAt,
		UpdatedAt:   apply.UpdatedAt,
		IsDeleted:   false,
	}

	return r.data.db.WithContext(ctx).Create(&dbApply).Error
}

func (r *groupRepo) GetGroupApply(ctx context.Context, id int64) (*biz.GroupApply, error) {
	var dbApply model.GroupApply
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbApply).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.GroupApply{
		ID:          dbApply.Id,
		UserID:      dbApply.UserId,
		GroupID:     dbApply.GroupId,
		ApplyReason: dbApply.ApplyReason,
		Status:      int32(dbApply.Status),
		HandlerID:   dbApply.HandlerId,
		ReplyMsg:    dbApply.ReplyMsg,
		CreatedAt:   dbApply.CreatedAt,
		UpdatedAt:   dbApply.UpdatedAt,
	}, nil
}

func (r *groupRepo) UpdateGroupApply(ctx context.Context, apply *biz.GroupApply) error {
	return r.data.db.WithContext(ctx).
		Model(&model.GroupApply{}).
		Where("id = ?", apply.ID).
		Updates(map[string]interface{}{
			"status":     apply.Status,
			"handler_id": apply.HandlerID,
			"reply_msg":  apply.ReplyMsg,
			"updated_at": time.Now(),
		}).Error
}

func (r *groupRepo) ListPendingApplies(ctx context.Context, groupID int64, offset, limit int) ([]*biz.GroupApply, error) {
	var dbApplies []*model.GroupApply

	query := r.data.db.WithContext(ctx).
		Where("group_id = ? AND status = ? AND is_deleted = ?", groupID, 0, false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbApplies).Error
	if err != nil {
		return nil, err
	}

	applies := make([]*biz.GroupApply, 0, len(dbApplies))
	for _, dbApply := range dbApplies {
		applies = append(applies, &biz.GroupApply{
			ID:          dbApply.Id,
			UserID:      dbApply.UserId,
			GroupID:     dbApply.GroupId,
			ApplyReason: dbApply.ApplyReason,
			Status:      int32(dbApply.Status),
			HandlerID:   dbApply.HandlerId,
			ReplyMsg:    dbApply.ReplyMsg,
			CreatedAt:   dbApply.CreatedAt,
			UpdatedAt:   dbApply.UpdatedAt,
		})
	}

	return applies, nil
}

func (r *groupRepo) CountPendingApplies(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupApply{}).
		Where("group_id = ? AND status = ? AND is_deleted = ?", groupID, 0, false).
		Count(&count).Error

	return count, err
}

// 优化查询，使用 JOIN 提高性能
func (r *groupRepo) ListJoinedGroups(ctx context.Context, userID int64, offset, limit int) ([]*biz.Group, error) {
	var dbGroups []*model.GroupInfo

	// 使用 JOIN 查询用户加入的群聊
	query := r.data.db.WithContext(ctx).
		Model(&model.GroupInfo{}).
		Joins("JOIN group_member ON group_info.id = group_member.group_id").
		Where("group_member.user_id = ? AND group_member.is_deleted = ? AND group_info.is_deleted = ?",
			userID, false, false).
		Order("group_info.created_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbGroups).Error
	if err != nil {
		return nil, err
	}

	groups := make([]*biz.Group, 0, len(dbGroups))
	for _, dbGroup := range dbGroups {
		groups = append(groups, &biz.Group{
			ID:        dbGroup.Id,
			Name:      dbGroup.Name,
			Notice:    dbGroup.Notice,
			MemberCnt: dbGroup.MemberCnt,
			OwnerID:   dbGroup.OwnerId,
			AddMode:   int32(dbGroup.AddMode),
			Avatar:    dbGroup.Avatar,
			Status:    int32(dbGroup.Status),
			CreatedAt: dbGroup.CreatedAt,
			UpdatedAt: dbGroup.UpdatedAt,
		})
	}

	return groups, nil
}

func (r *groupRepo) CountJoinedGroups(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&count).Error

	return count, err
}

// 批量检查用户是否在多个群中
func (r *groupRepo) BatchIsGroupMember(ctx context.Context, groupIDs []int64, userID int64) (map[int64]bool, error) {
	var members []struct {
		GroupID int64
	}

	err := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Select("group_id").
		Where("group_id IN ? AND user_id = ? AND is_deleted = ?", groupIDs, userID, false).
		Find(&members).Error

	if err != nil {
		return nil, err
	}

	result := make(map[int64]bool)
	for _, member := range members {
		result[member.GroupID] = true
	}

	return result, nil
}
