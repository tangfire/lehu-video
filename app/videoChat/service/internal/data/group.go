package data

import (
	"context"
	"encoding/json"
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

func (r *groupRepo) CreateGroup(ctx context.Context, group *biz.Group) error {
	// 序列化成员列表
	membersJSON, err := json.Marshal(group.Members)
	if err != nil {
		return err
	}

	dbGroup := model.GroupInfo{
		Id:        group.ID,
		Name:      group.Name,
		Notice:    group.Notice,
		Members:   membersJSON,
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

	// 反序列化成员列表
	var members []int64
	if len(dbGroup.Members) > 0 {
		err = json.Unmarshal(dbGroup.Members, &members)
		if err != nil {
			return nil, err
		}
	}

	return &biz.Group{
		ID:        dbGroup.Id,
		Name:      dbGroup.Name,
		Notice:    dbGroup.Notice,
		Members:   members,
		MemberCnt: dbGroup.MemberCnt,
		OwnerID:   dbGroup.OwnerId,
		AddMode:   int32(dbGroup.AddMode),
		Avatar:    dbGroup.Avatar,
		Status:    int32(dbGroup.Status),
		CreatedAt: dbGroup.CreatedAt,
		UpdatedAt: dbGroup.UpdatedAt,
	}, nil
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

	// 反序列化成员列表
	var members []int64
	if len(dbGroup.Members) > 0 {
		err = json.Unmarshal(dbGroup.Members, &members)
		if err != nil {
			return nil, err
		}
	}

	return &biz.Group{
		ID:        dbGroup.Id,
		Name:      dbGroup.Name,
		Notice:    dbGroup.Notice,
		Members:   members,
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
	// 序列化成员列表
	membersJSON, err := json.Marshal(group.Members)
	if err != nil {
		return err
	}

	return r.data.db.WithContext(ctx).
		Model(&model.GroupInfo{}).
		Where("id = ?", group.ID).
		Updates(map[string]interface{}{
			"name":       group.Name,
			"notice":     group.Notice,
			"members":    membersJSON,
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
		// 反序列化成员列表
		var members []int64
		if len(dbGroup.Members) > 0 {
			err = json.Unmarshal(dbGroup.Members, &members)
			if err != nil {
				return nil, err
			}
		}

		groups = append(groups, &biz.Group{
			ID:        dbGroup.Id,
			Name:      dbGroup.Name,
			Notice:    dbGroup.Notice,
			Members:   members,
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

func (r *groupRepo) ListJoinedGroups(ctx context.Context, userID int64, offset, limit int) ([]*biz.Group, error) {
	var dbGroups []*model.GroupInfo

	// 先查询用户加入的群聊ID
	var groupIDs []int64
	subQuery := r.data.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Select("group_id").
		Where("user_id = ? AND is_deleted = ?", userID, false)

	if limit > 0 {
		subQuery = subQuery.Offset(offset).Limit(limit)
	}

	err := subQuery.Pluck("group_id", &groupIDs).Error
	if err != nil {
		return nil, err
	}

	if len(groupIDs) == 0 {
		return []*biz.Group{}, nil
	}

	// 查询群聊信息
	err = r.data.db.WithContext(ctx).
		Where("id IN ? AND is_deleted = ?", groupIDs, false).
		Order("created_at DESC").
		Find(&dbGroups).Error
	if err != nil {
		return nil, err
	}

	groups := make([]*biz.Group, 0, len(dbGroups))
	for _, dbGroup := range dbGroups {
		// 反序列化成员列表
		var members []int64
		if len(dbGroup.Members) > 0 {
			err = json.Unmarshal(dbGroup.Members, &members)
			if err != nil {
				return nil, err
			}
		}

		groups = append(groups, &biz.Group{
			ID:        dbGroup.Id,
			Name:      dbGroup.Name,
			Notice:    dbGroup.Notice,
			Members:   members,
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
