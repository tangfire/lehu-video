package data

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
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

func (r *favoriteRepo) CreateFavorite(ctx context.Context, favorite *biz.Favorite) error {
	dbFavorite := model.Favorite{
		Id:           favorite.Id,
		UserId:       favorite.UserId,
		TargetType:   int64(favorite.TargetType),
		TargetId:     favorite.TargetId,
		FavoriteType: int64(favorite.FavoriteType),
		IsDeleted:    favorite.IsDeleted,
		CreatedAt:    favorite.CreatedAt,
		UpdatedAt:    favorite.UpdatedAt,
	}

	return r.data.db.WithContext(ctx).Create(&dbFavorite).Error
}

func (r *favoriteRepo) UpdateFavorite(ctx context.Context, favorite *biz.Favorite) error {
	dbFavorite := model.Favorite{
		Id:        favorite.Id,
		IsDeleted: favorite.IsDeleted,
		UpdatedAt: favorite.UpdatedAt,
	}

	return r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("id = ?", favorite.Id).
		Updates(&dbFavorite).Error
}

func (r *favoriteRepo) GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (*biz.Favorite, error) {
	var dbFavorite model.Favorite

	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", favoriteType).
		First(&dbFavorite).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.Favorite{
		Id:           dbFavorite.Id,
		UserId:       dbFavorite.UserId,
		TargetType:   int32(dbFavorite.TargetType),
		TargetId:     dbFavorite.TargetId,
		FavoriteType: int32(dbFavorite.FavoriteType),
		IsDeleted:    dbFavorite.IsDeleted,
		CreatedAt:    dbFavorite.CreatedAt,
		UpdatedAt:    dbFavorite.UpdatedAt,
	}, nil
}

func (r *favoriteRepo) SoftDeleteFavorite(ctx context.Context, favoriteId int64) error {
	return r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("id = ?", favoriteId).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}).Error
}

func (r *favoriteRepo) ListFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32, offset, limit int) ([]*biz.Favorite, error) {
	var dbFavorites []*model.Favorite

	query := r.data.db.WithContext(ctx).
		Where("is_deleted = ?", false)

	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}
	if targetId != -1 {
		query = query.Where("target_id = ?", targetId)
	}
	if targetType != -1 {
		query = query.Where("target_type = ?", targetType)
	}
	if favoriteType != -1 {
		query = query.Where("favorite_type = ?", favoriteType)
	}

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&dbFavorites).Error

	if err != nil {
		return nil, err
	}

	favorites := make([]*biz.Favorite, 0, len(dbFavorites))
	for _, dbFavorite := range dbFavorites {
		favorites = append(favorites, &biz.Favorite{
			Id:           dbFavorite.Id,
			UserId:       dbFavorite.UserId,
			TargetType:   int32(dbFavorite.TargetType),
			TargetId:     dbFavorite.TargetId,
			FavoriteType: int32(dbFavorite.FavoriteType),
			IsDeleted:    dbFavorite.IsDeleted,
			CreatedAt:    dbFavorite.CreatedAt,
			UpdatedAt:    dbFavorite.UpdatedAt,
		})
	}

	return favorites, nil
}

func (r *favoriteRepo) CountFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (int64, error) {
	var count int64

	query := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("is_deleted = ?", false)

	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}
	if targetId != -1 {
		query = query.Where("target_id = ?", targetId)
	}
	if targetType != -1 {
		query = query.Where("target_type = ?", targetType)
	}
	if favoriteType != -1 {
		query = query.Where("favorite_type = ?", favoriteType)
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *favoriteRepo) CountFavoritesByTargetIds(ctx context.Context, targetIds []int64, targetType, favoriteType int32) (map[int64]int64, error) {
	type Result struct {
		TargetId int64 `gorm:"column:target_id"`
		Count    int64 `gorm:"column:count"`
	}

	var results []Result

	err := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Select("target_id, COUNT(*) as count").
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", favoriteType).
		Where("is_deleted = ?", false).
		Group("target_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	resultMap := make(map[int64]int64)
	for _, res := range results {
		resultMap[res.TargetId] = res.Count
	}

	// 为没有点赞的目标设置0
	for _, targetId := range targetIds {
		if _, exists := resultMap[targetId]; !exists {
			resultMap[targetId] = 0
		}
	}

	return resultMap, nil
}

func (r *favoriteRepo) CountFavoritesByUserIds(ctx context.Context, userIds []int64, targetType, favoriteType int32) (map[int64]int64, error) {
	type Result struct {
		UserId int64 `gorm:"column:user_id"`
		Count  int64 `gorm:"column:count"`
	}

	var results []Result

	err := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Select("user_id, COUNT(*) as count").
		Where("user_id IN (?)", userIds).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", favoriteType).
		Where("is_deleted = ?", false).
		Group("user_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	resultMap := make(map[int64]int64)
	for _, res := range results {
		resultMap[res.UserId] = res.Count
	}

	// 为没有点赞的用户设置0
	for _, userId := range userIds {
		if _, exists := resultMap[userId]; !exists {
			resultMap[userId] = 0
		}
	}

	return resultMap, nil
}

func (r *favoriteRepo) GetFavoritesByUserAndTargets(ctx context.Context, userId int64, targetIds []int64, targetType, favoriteType int32) ([]*biz.Favorite, error) {
	var dbFavorites []*model.Favorite

	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", favoriteType).
		Where("is_deleted = ?", false).
		Find(&dbFavorites).Error

	if err != nil {
		return nil, err
	}

	favorites := make([]*biz.Favorite, 0, len(dbFavorites))
	for _, dbFavorite := range dbFavorites {
		favorites = append(favorites, &biz.Favorite{
			Id:           dbFavorite.Id,
			UserId:       dbFavorite.UserId,
			TargetType:   int32(dbFavorite.TargetType),
			TargetId:     dbFavorite.TargetId,
			FavoriteType: int32(dbFavorite.FavoriteType),
			IsDeleted:    dbFavorite.IsDeleted,
			CreatedAt:    dbFavorite.CreatedAt,
			UpdatedAt:    dbFavorite.UpdatedAt,
		})
	}

	return favorites, nil
}
