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

func (r *friendRepo) BatchGetUserInfo(ctx context.Context, userIDs []int64) (map[int64]*biz.UserInfo, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*biz.UserInfo), nil
	}

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

	// 获取在线状态
	onlineStatus, _ := r.BatchGetUserOnlineStatus(ctx, userIDs)

	// 组装结果
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

		// 更新在线状态
		if onlineUser, ok := onlineStatus[cast.ToInt64(user.Id)]; ok {
			userInfo.OnlineStatus = onlineUser.OnlineStatus
			userInfo.LastOnlineTime = onlineUser.LastOnlineTime
		}

		result[cast.ToInt64(user.Id)] = userInfo
	}

	return result, nil
}

// 好友关系操作
func (r *friendRepo) CreateFriendRelation(ctx context.Context, relation *biz.FriendRelation) error {
	dbRelation := model.FriendRelation{
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

	return r.data.db.WithContext(ctx).Create(&dbRelation).Error
}

// 在 friend.go 文件的 friendRepo 结构体方法中添加

func (r *friendRepo) CheckFriendRelation(ctx context.Context, userID, friendID int64) (bool, error) {
	var count int64

	// 检查是否存在好友关系（状态为1表示正常好友）
	err := r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("user_id = ? AND friend_id = ? AND status = 1", userID, friendID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	// 同时检查反向关系（如果对方也把你加为好友）
	if count > 0 {
		return true, nil
	}

	// 为了确保是双向好友，也可以检查反向关系
	var reverseCount int64
	err = r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("user_id = ? AND friend_id = ? AND status = 1", friendID, userID).
		Count(&reverseCount).Error

	if err != nil {
		return false, err
	}

	// 可以选择不同的逻辑：
	// 1. 如果是单向好友关系：return count > 0, nil
	// 2. 如果是双向好友关系：return count > 0 && reverseCount > 0, nil

	// 这里假设是双向好友关系
	return count > 0 && reverseCount > 0, nil
}

func (r *friendRepo) GetFriendRelation(ctx context.Context, userID, friendID int64) (*biz.FriendRelation, error) {
	var dbRelation model.FriendRelation

	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		First(&dbRelation).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.FriendRelation{
		ID:          dbRelation.ID,
		UserID:      dbRelation.UserID,
		FriendID:    dbRelation.FriendID,
		Status:      dbRelation.Status,
		Remark:      dbRelation.Remark,
		GroupName:   dbRelation.GroupName,
		IsFollowing: dbRelation.IsFollowing,
		IsFollower:  dbRelation.IsFollower,
		CreatedAt:   dbRelation.CreatedAt,
		UpdatedAt:   dbRelation.UpdatedAt,
	}, nil
}

func (r *friendRepo) UpdateFriendRelation(ctx context.Context, relation *biz.FriendRelation) error {
	updates := map[string]interface{}{
		"status":       relation.Status,
		"remark":       relation.Remark,
		"group_name":   relation.GroupName,
		"is_following": relation.IsFollowing,
		"is_follower":  relation.IsFollower,
		"updated_at":   time.Now(),
	}

	return r.data.db.WithContext(ctx).
		Model(&model.FriendRelation{}).
		Where("id = ?", relation.ID).
		Updates(updates).Error
}

func (r *friendRepo) DeleteFriendRelation(ctx context.Context, userID, friendID int64) error {
	return r.data.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Delete(&model.FriendRelation{}).Error
}

func (r *friendRepo) ListFriends(ctx context.Context, userID int64, offset, limit int, groupName *string) ([]*biz.FriendRelation, int64, error) {
	// 查询条件
	db := r.data.db.WithContext(ctx).
		Where("user_id = ? AND status = 1", userID)

	if groupName != nil && *groupName != "" {
		db = db.Where("group_name = ?", *groupName)
	}

	// 获取总数
	var total int64
	if err := db.Model(&model.FriendRelation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var dbRelations []model.FriendRelation
	err := db.Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&dbRelations).Error

	if err != nil {
		return nil, 0, err
	}

	// 转换
	relations := make([]*biz.FriendRelation, 0, len(dbRelations))
	for _, dbRelation := range dbRelations {
		relations = append(relations, &biz.FriendRelation{
			ID:          dbRelation.ID,
			UserID:      dbRelation.UserID,
			FriendID:    dbRelation.FriendID,
			Status:      dbRelation.Status,
			Remark:      dbRelation.Remark,
			GroupName:   dbRelation.GroupName,
			IsFollowing: dbRelation.IsFollowing,
			IsFollower:  dbRelation.IsFollower,
			CreatedAt:   dbRelation.CreatedAt,
			UpdatedAt:   dbRelation.UpdatedAt,
		})
	}

	return relations, total, nil
}

// 好友申请操作
func (r *friendRepo) CreateFriendApply(ctx context.Context, apply *biz.FriendApply) error {
	dbApply := model.FriendApply{
		ID:          apply.ID,
		ApplicantID: apply.ApplicantID,
		ReceiverID:  apply.ReceiverID,
		ApplyReason: apply.ApplyReason,
		Status:      apply.Status,
		HandledAt:   apply.HandledAt,
		CreatedAt:   apply.CreatedAt,
		UpdatedAt:   apply.UpdatedAt,
	}

	return r.data.db.WithContext(ctx).Create(&dbApply).Error
}

func (r *friendRepo) GetFriendApply(ctx context.Context, applyID int64) (*biz.FriendApply, error) {
	var dbApply model.FriendApply

	err := r.data.db.WithContext(ctx).
		Where("id = ?", applyID).
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
		Status:      dbApply.Status,
		HandledAt:   dbApply.HandledAt,
		CreatedAt:   dbApply.CreatedAt,
		UpdatedAt:   dbApply.UpdatedAt,
	}, nil
}

func (r *friendRepo) UpdateFriendApply(ctx context.Context, apply *biz.FriendApply) error {
	updates := map[string]interface{}{
		"status":     apply.Status,
		"handled_at": apply.HandledAt,
		"updated_at": time.Now(),
	}

	return r.data.db.WithContext(ctx).
		Model(&model.FriendApply{}).
		Where("id = ?", apply.ID).
		Updates(updates).Error
}

func (r *friendRepo) ListFriendApplies(ctx context.Context, userID int64, status *int32, offset, limit int) ([]*biz.FriendApply, int64, error) {
	// 查询条件
	db := r.data.db.WithContext(ctx).
		Where("receiver_id = ?", userID)

	if status != nil {
		db = db.Where("status = ?", *status)
	}

	// 获取总数
	var total int64
	if err := db.Model(&model.FriendApply{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var dbApplies []model.FriendApply
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&dbApplies).Error

	if err != nil {
		return nil, 0, err
	}

	// 转换
	applies := make([]*biz.FriendApply, 0, len(dbApplies))
	for _, dbApply := range dbApplies {
		applies = append(applies, &biz.FriendApply{
			ID:          dbApply.ID,
			ApplicantID: dbApply.ApplicantID,
			ReceiverID:  dbApply.ReceiverID,
			ApplyReason: dbApply.ApplyReason,
			Status:      dbApply.Status,
			HandledAt:   dbApply.HandledAt,
			CreatedAt:   dbApply.CreatedAt,
			UpdatedAt:   dbApply.UpdatedAt,
		})
	}

	return applies, total, nil
}

func (r *friendRepo) CheckPendingApply(ctx context.Context, applicantID int64, receiverID int64) (bool, error) {
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

// 在线状态操作
func (r *friendRepo) UpdateUserOnlineStatus(ctx context.Context, userID int64, status int32, deviceType string) error {
	now := time.Now()

	dbStatus := model.UserOnlineStatus{
		UserID:         userID,
		OnlineStatus:   status,
		DeviceType:     deviceType,
		LastOnlineTime: now,
		UpdatedAt:      now,
	}

	// 使用upsert
	return r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Assign(dbStatus).
		FirstOrCreate(&model.UserOnlineStatus{}).Error
}

func (r *friendRepo) GetUserOnlineStatus(ctx context.Context, userID int64) (*biz.UserInfo, error) {
	var dbStatus model.UserOnlineStatus

	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&dbStatus).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 返回默认离线状态
		return &biz.UserInfo{
			ID:             userID,
			OnlineStatus:   0,
			LastOnlineTime: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	// 获取用户基础信息
	userInfo, err := r.GetUserInfo(ctx, userID)
	if err != nil {
		// 如果获取不到用户信息，只返回在线状态
		return &biz.UserInfo{
			ID:             userID,
			OnlineStatus:   dbStatus.OnlineStatus,
			LastOnlineTime: dbStatus.LastOnlineTime,
		}, nil
	}

	// 更新在线状态
	userInfo.OnlineStatus = dbStatus.OnlineStatus
	userInfo.LastOnlineTime = dbStatus.LastOnlineTime

	return userInfo, nil
}

func (r *friendRepo) BatchGetUserOnlineStatus(ctx context.Context, userIDs []int64) (map[int64]*biz.UserInfo, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*biz.UserInfo), nil
	}

	// 查询在线状态
	var dbStatuses []model.UserOnlineStatus
	err := r.data.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&dbStatuses).Error

	if err != nil {
		return nil, err
	}

	// 构建在线状态映射
	statusMap := make(map[int64]model.UserOnlineStatus)
	for _, status := range dbStatuses {
		statusMap[status.UserID] = status
	}

	// 获取用户信息
	userInfos, err := r.BatchGetUserInfo(ctx, userIDs)
	if err != nil {
		// 如果获取用户信息失败，只返回在线状态
		result := make(map[int64]*biz.UserInfo)
		for _, userID := range userIDs {
			if status, ok := statusMap[userID]; ok {
				result[userID] = &biz.UserInfo{
					ID:             userID,
					OnlineStatus:   status.OnlineStatus,
					LastOnlineTime: status.LastOnlineTime,
				}
			} else {
				result[userID] = &biz.UserInfo{
					ID:             userID,
					OnlineStatus:   0,
					LastOnlineTime: time.Now(),
				}
			}
		}
		return result, nil
	}

	// 合并在线状态
	for userID, userInfo := range userInfos {
		if status, ok := statusMap[userID]; ok {
			userInfo.OnlineStatus = status.OnlineStatus
			userInfo.LastOnlineTime = status.LastOnlineTime
		} else {
			userInfo.OnlineStatus = 0
		}
	}

	return userInfos, nil
}
