package data

import (
	"context"
	"errors"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoChat/service/internal/pkg/utils/respcheck"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/data/model"
)

type friendRepo struct {
	data *Data
	log  *log.Helper
}

func NewFriendRepo(data *Data, logger log.Logger) biz.FriendRepo {
	return &friendRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *friendRepo) CreateFriendRelation(ctx context.Context, relation *biz.FriendRelation) error {
	dbRelation := model.FriendRelation{
		ID:        relation.ID,
		UserID:    relation.UserID,
		FriendID:  relation.FriendID,
		Status:    int8(relation.Status),
		Remark:    relation.Remark,
		GroupName: relation.GroupName,
		CreatedAt: relation.CreatedAt,
		UpdatedAt: relation.UpdatedAt,
		IsDeleted: false,
	}

	return r.data.db.WithContext(ctx).Create(&dbRelation).Error
}

func (r *friendRepo) GetFriendRelation(ctx context.Context, userID, friendID int64) (*biz.FriendRelation, error) {
	var dbRelation model.FriendRelation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ? AND is_deleted = ?", userID, friendID, false).
		First(&dbRelation).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.FriendRelation{
		ID:        dbRelation.ID,
		UserID:    dbRelation.UserID,
		FriendID:  dbRelation.FriendID,
		Status:    int32(dbRelation.Status),
		Remark:    dbRelation.Remark,
		GroupName: dbRelation.GroupName,
		CreatedAt: dbRelation.CreatedAt,
		UpdatedAt: dbRelation.UpdatedAt,
	}, nil
}

func (r *friendRepo) CheckFriendRelation(ctx context.Context, userID, targetID int64) (bool, int32, error) {
	relation, err := r.GetFriendRelation(ctx, userID, targetID)
	if err != nil {
		return false, 0, err
	}

	if relation == nil {
		return false, 0, nil
	}

	return relation.Status == 1, relation.Status, nil
}

func (r *friendRepo) UpdateFriendRelation(ctx context.Context, relation *biz.FriendRelation) error {
	return r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("id = ?", relation.ID).
		Updates(map[string]interface{}{
			"status":     relation.Status,
			"remark":     relation.Remark,
			"group_name": relation.GroupName,
			"updated_at": time.Now(),
			"is_deleted": false,
		}).Error
}

func (r *friendRepo) DeleteFriendRelation(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

func (r *friendRepo) ListFriends(ctx context.Context, userID int64, offset, limit int, groupName *string) ([]*biz.FriendRelation, error) {
	var dbRelations []*model.FriendRelation

	query := r.data.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND is_deleted = ?", userID, 1, false)

	if groupName != nil {
		query = query.Where("group_name = ?", *groupName)
	}

	query = query.Order("updated_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbRelations).Error
	if err != nil {
		return nil, err
	}

	relations := make([]*biz.FriendRelation, 0, len(dbRelations))
	for _, dbRelation := range dbRelations {
		relations = append(relations, &biz.FriendRelation{
			ID:        dbRelation.ID,
			UserID:    dbRelation.UserID,
			FriendID:  dbRelation.FriendID,
			Status:    int32(dbRelation.Status),
			Remark:    dbRelation.Remark,
			GroupName: dbRelation.GroupName,
			CreatedAt: dbRelation.CreatedAt,
			UpdatedAt: dbRelation.UpdatedAt,
		})
	}

	return relations, nil
}

func (r *friendRepo) CountFriends(ctx context.Context, userID int64, groupName *string) (int64, error) {
	var count int64

	query := r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("user_id = ? AND status = ? AND is_deleted = ?", userID, 1, false)

	if groupName != nil {
		query = query.Where("group_name = ?", *groupName)
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *friendRepo) CreateFriendApply(ctx context.Context, apply *biz.FriendApply) error {
	dbApply := model.FriendApply{
		ID:          apply.ID,
		ApplicantID: apply.ApplicantID,
		ReceiverID:  apply.ReceiverID,
		ApplyReason: apply.ApplyReason,
		Status:      int8(apply.Status),
		HandledAt:   apply.HandledAt,
		CreatedAt:   apply.CreatedAt,
		UpdatedAt:   apply.UpdatedAt,
		IsDeleted:   false,
	}

	return r.data.db.WithContext(ctx).Create(&dbApply).Error
}

func (r *friendRepo) GetFriendApply(ctx context.Context, id int64) (*biz.FriendApply, error) {
	var dbApply model.FriendApply
	err := r.data.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbApply).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.FriendApply{
		ID:          dbApply.ID,
		ApplicantID: dbApply.ApplicantID,
		ReceiverID:  dbApply.ReceiverID,
		ApplyReason: dbApply.ApplyReason,
		Status:      int32(dbApply.Status),
		HandledAt:   dbApply.HandledAt,
		CreatedAt:   dbApply.CreatedAt,
		UpdatedAt:   dbApply.UpdatedAt,
	}, nil
}

func (r *friendRepo) UpdateFriendApply(ctx context.Context, apply *biz.FriendApply) error {
	return r.data.db.WithContext(ctx).
		Model(&model.FriendApply{}).
		Where("id = ?", apply.ID).
		Updates(map[string]interface{}{
			"status":     apply.Status,
			"handled_at": apply.HandledAt,
			"updated_at": time.Now(),
		}).Error
}

func (r *friendRepo) ListFriendApplies(ctx context.Context, userID int64, status *int32, offset, limit int) ([]*biz.FriendApply, error) {
	var dbApplies []*model.FriendApply

	query := r.data.db.WithContext(ctx).
		Where("receiver_id = ? AND is_deleted = ?", userID, false)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	query = query.Order("created_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbApplies).Error
	if err != nil {
		return nil, err
	}

	applies := make([]*biz.FriendApply, 0, len(dbApplies))
	for _, dbApply := range dbApplies {
		applies = append(applies, &biz.FriendApply{
			ID:          dbApply.ID,
			ApplicantID: dbApply.ApplicantID,
			ReceiverID:  dbApply.ReceiverID,
			ApplyReason: dbApply.ApplyReason,
			Status:      int32(dbApply.Status),
			HandledAt:   dbApply.HandledAt,
			CreatedAt:   dbApply.CreatedAt,
			UpdatedAt:   dbApply.UpdatedAt,
		})
	}

	return applies, nil
}

func (r *friendRepo) CountFriendApplies(ctx context.Context, userID int64, status *int32) (int64, error) {
	var count int64

	query := r.data.db.WithContext(ctx).
		Model(&model.FriendApply{}).
		Where("receiver_id = ? AND is_deleted = ?", userID, false)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *friendRepo) UpdateUserOnlineStatus(ctx context.Context, status *biz.UserOnlineStatus) error {
	dbStatus := model.UserOnlineStatus{
		ID:             status.ID,
		UserID:         status.UserID,
		Status:         int8(status.Status),
		DeviceType:     status.DeviceType,
		LastOnlineTime: status.LastOnlineTime,
		UpdatedAt:      time.Now(),
	}

	// 使用 upsert
	return r.data.db.WithContext(ctx).
		Where("user_id = ?", status.UserID).
		Assign(map[string]interface{}{
			"status":           status.Status,
			"device_type":      status.DeviceType,
			"last_online_time": status.LastOnlineTime,
			"updated_at":       time.Now(),
		}).
		FirstOrCreate(&dbStatus).Error
}

func (r *friendRepo) GetUserOnlineStatus(ctx context.Context, userID int64) (*biz.UserOnlineStatus, error) {
	var dbStatus model.UserOnlineStatus
	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&dbStatus).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.UserOnlineStatus{
		ID:             dbStatus.ID,
		UserID:         dbStatus.UserID,
		Status:         int32(dbStatus.Status),
		DeviceType:     dbStatus.DeviceType,
		LastOnlineTime: dbStatus.LastOnlineTime,
		CreatedAt:      dbStatus.CreatedAt,
		UpdatedAt:      dbStatus.UpdatedAt,
	}, nil
}

func (r *friendRepo) BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]*biz.UserOnlineStatus, error) {
	var dbStatuses []*model.UserOnlineStatus

	err := r.data.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&dbStatuses).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]*biz.UserOnlineStatus)
	for _, dbStatus := range dbStatuses {
		result[dbStatus.UserID] = &biz.UserOnlineStatus{
			ID:             dbStatus.ID,
			UserID:         dbStatus.UserID,
			Status:         int32(dbStatus.Status),
			DeviceType:     dbStatus.DeviceType,
			LastOnlineTime: dbStatus.LastOnlineTime,
			CreatedAt:      dbStatus.CreatedAt,
			UpdatedAt:      dbStatus.UpdatedAt,
		}
	}

	return result, nil
}

func (r *friendRepo) GetUserInfo(ctx context.Context, userID int64) (*biz.UserInfo, error) {
	resp, err := r.data.user.GetUserInfo(ctx, &core.GetUserInfoReq{
		UserId: userID,
	})
	if err != nil {
		r.log.Errorf("RPC调用GetUserInfo失败: %v", err)
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	// 解析时间
	lastOnlineTime := time.Now()
	if resp.User.LastOnlineTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", resp.User.LastOnlineTime); err == nil {
			lastOnlineTime = t
		}
	}

	return &biz.UserInfo{
		ID:             resp.User.Id,
		Username:       resp.User.Name,
		Nickname:       resp.User.Nickname,
		Avatar:         resp.User.Avatar,
		Signature:      resp.User.Signature,
		Gender:         resp.User.Gender,
		OnlineStatus:   resp.User.OnlineStatus,
		LastOnlineTime: lastOnlineTime,
	}, nil
}

func (r *friendRepo) BatchGetUserInfo(ctx context.Context, userIDs []int64) (map[int64]*biz.UserInfo, error) {
	resp, err := r.data.user.GetUserByIdList(ctx, &core.GetUserByIdListReq{
		UserIdList: userIDs,
	})
	if err != nil {
		r.log.Errorf("RPC调用GetUserByIdList失败: %v", err)
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]*biz.UserInfo)
	for _, user := range resp.UserList {
		// 解析时间
		lastOnlineTime := time.Now()
		if user.LastOnlineTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", user.LastOnlineTime); err == nil {
				lastOnlineTime = t
			}
		}

		result[user.Id] = &biz.UserInfo{
			ID:             user.Id,
			Username:       user.Name,
			Nickname:       user.Nickname,
			Avatar:         user.Avatar,
			Signature:      user.Signature,
			Gender:         user.Gender,
			OnlineStatus:   user.OnlineStatus,
			LastOnlineTime: lastOnlineTime,
		}
	}

	return result, nil
}

func (r *friendRepo) SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]*biz.UserInfo, int64, error) {
	// 计算页码
	page := offset/limit + 1
	if page < 1 {
		page = 1
	}

	resp, err := r.data.user.SearchUsers(ctx, &core.SearchUsersReq{
		Keyword:  keyword,
		Page:     int32(page),
		PageSize: int32(limit),
	})
	if err != nil {
		r.data.log.Errorf("RPC调用SearchUsers失败: %v", err)
		return nil, 0, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, 0, err
	}

	users := make([]*biz.UserInfo, 0, len(resp.Users))
	for _, user := range resp.Users {
		// 解析时间
		lastOnlineTime := time.Now()
		if user.LastOnlineTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", user.LastOnlineTime); err == nil {
				lastOnlineTime = t
			}
		}

		users = append(users, &biz.UserInfo{
			ID:             user.Id,
			Username:       user.Name,
			Nickname:       user.Nickname,
			Avatar:         user.Avatar,
			Signature:      user.Signature,
			Gender:         user.Gender,
			OnlineStatus:   user.OnlineStatus,
			LastOnlineTime: lastOnlineTime,
		})
	}

	return users, int64(resp.Total), nil
}
