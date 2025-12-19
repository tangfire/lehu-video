package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"lehu-video/app/videoCore/service/internal/pkg/utils"
	"time"
)

type favoriteRepo struct {
	data *Data
	log  *log.Helper
}

func NewFavoriteRepo(data *Data, logger log.Logger) biz.FavoriteRepo {
	return &favoriteRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *favoriteRepo) AddFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) error {
	favorite := model.Favorite{
		Id:           int64(uuid.New().ID()),
		UserId:       userId,
		TargetType:   int64(targetType),
		TargetId:     targetId,
		FavoriteType: int64(favoriteType),
		IsDeleted:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 使用 clause.OnConflict 的正确写法
	return r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "target_id"},
				{Name: "target_type"},
				{Name: "favorite_type"},
			},
			Where: clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: "is_deleted", Value: false},
			}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).
		Create(&favorite).Error
}

func (r *favoriteRepo) GetFavoriteList(ctx context.Context, userId, targetId int64, targetType, favoriteType int32, pageStats *pb.PageStatsReq) ([]int64, error) {
	db := r.data.db.WithContext(ctx).Table(model.Favorite{}.TableName())
	if userId != -1 {
		db = db.Where("user_id = ?", userId)
	}
	if targetId != -1 {
		db = db.Where("target_id = ?", targetId)
	}
	if targetType != -1 {
		db = db.Where("target_type = ?", targetType)
	}
	if favoriteType != -1 {
		db = db.Where("favorite_type = ?", favoriteType)
	}
	db = db.Where("is_deleted = ?", false).Order("id desc")
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, err
	}
	var favoriteList []model.Favorite
	err = db.Offset(int((pageStats.Page - 1) * pageStats.Size)).Limit(int(pageStats.Size)).Find(&favoriteList).Error
	if err != nil {
		return nil, err
	}
	targetIdList := make([]int64, 0, len(favoriteList))
	targetIdList = utils.Slice2Slice(favoriteList, func(favorite model.Favorite) int64 {
		return favorite.TargetId
	})
	return targetIdList, nil
}

func (r *favoriteRepo) DeleteFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) error {
	err := r.data.db.WithContext(ctx).Table(model.Favorite{}.TableName()).
		Where("user_id = ?", userId).
		Where("target_id = ?", targetId).
		UpdateColumns(map[string]interface{}{
			"is_deleted": true,
		}).Error
	if err != nil {
		return err
	}
	return nil
}
