package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
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

func (r *favoriteRepo) GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (bool, int64, error) {
	favorite := model.Favorite{}
	err := r.data.db.WithContext(ctx).Table(model.Favorite{}.TableName()).
		Where("user_id = ?", userId).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", favoriteType).
		Where("is_deleted = ?", false).First(&favorite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, favorite.Id, nil
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
