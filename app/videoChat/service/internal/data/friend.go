package data

import (
	"context"
	"errors"
	"github.com/spf13/cast"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoChat/service/internal/biz"
	"lehu-video/app/videoChat/service/internal/data/model"
	"lehu-video/app/videoChat/service/internal/pkg/utils/respcheck"
)

type friendRepo struct {
	data *Data
	log  *log.Helper
	user core.UserServiceClient // videoCore用户服务客户端
}

func NewFriendRepo(data *Data, userClient core.UserServiceClient, logger log.Logger) biz.FriendRepo {
	return &friendRepo{
		data: data,
		user: userClient,
		log:  log.NewHelper(logger),
	}
}

func (r *friendRepo) GetUserInfo(ctx context.Context, userID int64) (*biz.UserInfo, error) {
	resp, err := r.user.GetUserBaseInfo(ctx, &core.GetUserBaseInfoReq{
		UserId: cast.ToString(userID),
	})
	if err != nil {
		r.log.Errorf("RPC调用GetUserBaseInfo失败: %v", err)
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 获取在线状态
	var onlineStatus int32
	var lastOnlineTime time.Time

	status, err := r.GetUserOnlineStatus(ctx, userID)
	if err == nil && status != nil {
		onlineStatus = status.OnlineStatus
		lastOnlineTime = status.LastOnlineTime
	} else {
		lastOnlineTime, _ = time.Parse("2006-01-02 15:04:05", resp.User.CreatedAt)
	}

	return &biz.UserInfo{
		ID:             cast.ToInt64(resp.User.Id),
		Name:           resp.User.Name,
		Nickname:       resp.User.Nickname,
		Avatar:         resp.User.Avatar,
		Signature:      resp.User.Signature,
		Gender:         resp.User.Gender,
		OnlineStatus:   onlineStatus,
		LastOnlineTime: lastOnlineTime,
	}, nil
}

// BatchGetUserInfo 获取用户基本信息，并合并在线状态
func (r *friendRepo) BatchGetUserInfo(ctx context.Context, userIDs []int64) (map[int64]*biz.UserInfo, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*biz.UserInfo), nil
	}

	// 调用 RPC 获取用户基本信息
	resp, err := r.user.BatchGetUserBaseInfo(ctx, &core.BatchGetUserBaseInfoReq{
		UserIds: cast.ToStringSlice(userIDs),
	})
	if err != nil {
		r.log.Errorf("RPC调用BatchGetUserBaseInfo失败: %v", err)
		return nil, err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 获取在线状态（忽略错误，保留原逻辑）
	onlineStatus, _ := r.BatchGetUserOnlineStatus(ctx, userIDs)

	// 组装最终结果
	result := make(map[int64]*biz.UserInfo)
	for _, user := range resp.Users {
		lastOnlineTime, _ := time.Parse("2006-01-02 15:04:05", user.CreatedAt)

		userInfo := &biz.UserInfo{
			ID:             cast.ToInt64(user.Id),
			Name:           user.Name,
			Nickname:       user.Nickname,
			Avatar:         user.Avatar,
			Signature:      user.Signature,
			Gender:         user.Gender,
			OnlineStatus:   0,
			LastOnlineTime: lastOnlineTime,
		}

		// 合并在线状态
		if onlineUser, ok := onlineStatus[cast.ToInt64(user.Id)]; ok {
			userInfo.OnlineStatus = onlineUser.OnlineStatus
			userInfo.LastOnlineTime = onlineUser.LastOnlineTime
		}

		result[cast.ToInt64(user.Id)] = userInfo
	}

	return result, nil
}

// ==================== 好友关系 ====================
func (r *friendRepo) CreateFriendRelation(ctx context.Context, relation *biz.FriendRelation) error {
	dbRelation := &model.FriendRelation{
		ID:          relation.ID,
		UserID:      relation.UserID,
		FriendID:    relation.FriendID,
		Status:      relation.Status,
		Remark:      relation.Remark,
		GroupName:   relation.GroupName,
		IsFollowing: relation.IsFollowing,
		IsFollower:  relation.IsFollower,
		CreatedAt:   relation.CreatedAt,
		UpdatedAt:   relation.UpdatedAt,
	}
	return r.data.db.WithContext(ctx).Create(dbRelation).Error
}

// 在 friend.go 文件的 friendRepo 结构体方法中添加

func (r *friendRepo) GetFriendRelation(ctx context.Context, userID, friendID int64) (*biz.FriendRelation, error) {
	var db model.FriendRelation
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		First(&db).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizFriendRelation(&db), nil
}

func (r *friendRepo) UpdateFriendRelation(ctx context.Context, relation *biz.FriendRelation) error {
	return r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("id = ?", relation.ID).
		Updates(map[string]interface{}{
			"status":       relation.Status,
			"remark":       relation.Remark,
			"group_name":   relation.GroupName,
			"is_following": relation.IsFollowing,
			"is_follower":  relation.IsFollower,
			"updated_at":   time.Now(),
		}).Error
}

func (r *friendRepo) DeleteFriendRelation(ctx context.Context, userID, friendID int64) error {
	return r.data.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Delete(&model.FriendRelation{}).Error
}

func (r *friendRepo) ListFriends(ctx context.Context, userID int64, offset, limit int, groupName *string) ([]*biz.FriendRelation, int64, error) {
	db := r.data.db.WithContext(ctx).Where("user_id = ? AND status = 1", userID)
	if groupName != nil && *groupName != "" {
		db = db.Where("group_name = ?", *groupName)
	}

	var total int64
	if err := db.Model(&model.FriendRelation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var dbRelations []model.FriendRelation
	err := db.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&dbRelations).Error
	if err != nil {
		return nil, 0, err
	}

	relations := make([]*biz.FriendRelation, 0, len(dbRelations))
	for _, relation := range dbRelations {
		relations = append(relations, r.toBizFriendRelation(&relation))
	}
	return relations, total, nil
}

func (r *friendRepo) CheckFriendRelation(ctx context.Context, userID, friendID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("user_id = ? AND friend_id = ? AND status = 1", userID, friendID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ==================== 好友申请 ====================

func (r *friendRepo) CreateFriendApply(ctx context.Context, apply *biz.FriendApply) error {
	dbApply := &model.FriendApply{
		ID:          apply.ID,
		ApplicantID: apply.ApplicantID,
		ReceiverID:  apply.ReceiverID,
		ApplyReason: apply.ApplyReason,
		Status:      apply.Status,
		HandledAt:   apply.HandledAt,
		CreatedAt:   apply.CreatedAt,
		UpdatedAt:   apply.UpdatedAt,
	}
	return r.data.db.WithContext(ctx).Create(dbApply).Error
}

func (r *friendRepo) GetFriendApply(ctx context.Context, applyID int64) (*biz.FriendApply, error) {
	var db model.FriendApply
	err := r.data.db.WithContext(ctx).Where("id = ?", applyID).First(&db).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &biz.FriendApply{
		ID:          db.ID,
		ApplicantID: db.ApplicantID,
		ReceiverID:  db.ReceiverID,
		ApplyReason: db.ApplyReason,
		Status:      db.Status,
		HandledAt:   db.HandledAt,
		CreatedAt:   db.CreatedAt,
		UpdatedAt:   db.UpdatedAt,
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

func (r *friendRepo) ListFriendApplies(ctx context.Context, userID int64, status *int32, offset, limit int) ([]*biz.FriendApply, int64, error) {
	db := r.data.db.WithContext(ctx).Where("receiver_id = ?", userID)
	if status != nil {
		db = db.Where("status = ?", *status)
	}

	var total int64
	if err := db.Model(&model.FriendApply{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var dbApplies []model.FriendApply
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&dbApplies).Error
	if err != nil {
		return nil, 0, err
	}

	applies := make([]*biz.FriendApply, 0, len(dbApplies))
	for _, a := range dbApplies {
		applies = append(applies, &biz.FriendApply{
			ID:          a.ID,
			ApplicantID: a.ApplicantID,
			ReceiverID:  a.ReceiverID,
			ApplyReason: a.ApplyReason,
			Status:      a.Status,
			HandledAt:   a.HandledAt,
			CreatedAt:   a.CreatedAt,
			UpdatedAt:   a.UpdatedAt,
		})
	}
	return applies, total, nil
}

func (r *friendRepo) CheckPendingApply(ctx context.Context, applicantID, receiverID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Model(&model.FriendApply{}).
		Where("applicant_id = ? AND receiver_id = ? AND status = 0", applicantID, receiverID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ==================== 在线状态 ====================

func (r *friendRepo) UpdateUserOnlineStatus(ctx context.Context, userID int64, status int32, deviceType string) error {
	now := time.Now()
	// upsert
	return r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Assign(&model.UserOnlineStatus{
			UserID:         userID,
			OnlineStatus:   status,
			DeviceType:     deviceType,
			LastOnlineTime: now,
			UpdatedAt:      now,
		}).
		FirstOrCreate(&model.UserOnlineStatus{}).Error
}

func (r *friendRepo) GetUserOnlineStatus(ctx context.Context, userID int64) (*biz.UserOnlineStatus, error) {
	var db model.UserOnlineStatus
	err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).First(&db).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &biz.UserOnlineStatus{
		ID:             db.ID,
		UserID:         db.UserID,
		OnlineStatus:   db.OnlineStatus,
		DeviceType:     db.DeviceType,
		LastOnlineTime: db.LastOnlineTime,
		CreatedAt:      db.CreatedAt,
		UpdatedAt:      db.UpdatedAt,
	}, nil
}

func (r *friendRepo) BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]*biz.UserOnlineStatus, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*biz.UserOnlineStatus), nil
	}
	var dbStatuses []model.UserOnlineStatus
	err := r.data.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&dbStatuses).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*biz.UserOnlineStatus)
	for _, s := range dbStatuses {
		result[s.UserID] = &biz.UserOnlineStatus{
			ID:             s.ID,
			UserID:         s.UserID,
			OnlineStatus:   s.OnlineStatus,
			DeviceType:     s.DeviceType,
			LastOnlineTime: s.LastOnlineTime,
			CreatedAt:      s.CreatedAt,
			UpdatedAt:      s.UpdatedAt,
		}
	}
	return result, nil
}

// 辅助函数
func (r *friendRepo) toBizFriendRelation(db *model.FriendRelation) *biz.FriendRelation {
	return &biz.FriendRelation{
		ID:          db.ID,
		UserID:      db.UserID,
		FriendID:    db.FriendID,
		Status:      db.Status,
		Remark:      db.Remark,
		GroupName:   db.GroupName,
		IsFollowing: db.IsFollowing,
		IsFollower:  db.IsFollower,
		CreatedAt:   db.CreatedAt,
		UpdatedAt:   db.UpdatedAt,
	}
}
