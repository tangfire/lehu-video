package data

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
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
	return r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("id = ?", favorite.Id).
		Updates(map[string]interface{}{
			"is_deleted": favorite.IsDeleted,
			"updated_at": favorite.UpdatedAt,
		}).Error
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
	return r.toBizFavorite(&dbFavorite), nil
}

func (r *favoriteRepo) GetFavoriteByUserTarget(ctx context.Context, userId, targetId int64, targetType int32) (*biz.Favorite, error) {
	var dbFavorite model.Favorite
	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("is_deleted = ?", false).
		First(&dbFavorite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizFavorite(&dbFavorite), nil
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

func (r *favoriteRepo) HardDeleteFavorite(ctx context.Context, favoriteId int64) error {
	return r.data.db.WithContext(ctx).
		Where("id = ?", favoriteId).
		Delete(&model.Favorite{}).Error
}

func (r *favoriteRepo) ListFavorites(ctx context.Context, query *biz.ListFavoriteQuery) ([]*biz.Favorite, int64, error) {
	var dbFavorites []*model.Favorite
	var total int64

	db := r.data.db.WithContext(ctx).Model(&model.Favorite{})

	if !query.IncludeDeleted {
		db = db.Where("is_deleted = ?", false)
	}

	if query.AggregateType == 2 { // BY_USER
		db = db.Where("user_id = ?", query.Id)
	} else {
		db = db.Where("target_id = ?", query.Id)
	}

	if query.FavoriteType != -1 {
		db = db.Where("favorite_type = ?", query.FavoriteType)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.
		Offset(int((query.PageStats.Page - 1) * query.PageStats.PageSize)).
		Limit(int(query.PageStats.PageSize)).
		Order("created_at DESC").
		Find(&dbFavorites).Error
	if err != nil {
		return nil, 0, err
	}

	favorites := make([]*biz.Favorite, 0, len(dbFavorites))
	for _, f := range dbFavorites {
		favorites = append(favorites, r.toBizFavorite(f))
	}
	return favorites, total, nil
}

func (r *favoriteRepo) CountFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (int64, error) {
	var count int64
	db := r.data.db.WithContext(ctx).Model(&model.Favorite{}).Where("is_deleted = ?", false)
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
	err := db.Count(&count).Error
	return count, err
}

func (r *favoriteRepo) CountFavoritesByTargetIds(ctx context.Context, targetIds []int64, targetType int32) (map[int64]biz.FavoriteCount, error) {
	if len(targetIds) == 0 {
		return make(map[int64]biz.FavoriteCount), nil
	}

	type Result struct {
		TargetId     int64 `gorm:"column:target_id"`
		FavoriteType int64 `gorm:"column:favorite_type"`
		Count        int64 `gorm:"column:count"`
	}
	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Select("target_id, favorite_type, COUNT(*) as count").
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("is_deleted = ?", false).
		Group("target_id, favorite_type").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	resultMap := make(map[int64]biz.FavoriteCount)
	for _, tid := range targetIds {
		resultMap[tid] = biz.FavoriteCount{LikeCount: 0, DislikeCount: 0, TotalCount: 0}
	}
	for _, res := range results {
		cnt := resultMap[res.TargetId]
		if res.FavoriteType == 0 {
			cnt.LikeCount = res.Count
		} else {
			cnt.DislikeCount = res.Count
		}
		cnt.TotalCount = cnt.LikeCount + cnt.DislikeCount
		resultMap[res.TargetId] = cnt
	}
	return resultMap, nil
}

func (r *favoriteRepo) CountFavoritesByUserIds(ctx context.Context, userIds []int64, targetType int32) (map[int64]biz.FavoriteCount, error) {
	if len(userIds) == 0 {
		return make(map[int64]biz.FavoriteCount), nil
	}

	type Result struct {
		UserId       int64 `gorm:"column:user_id"`
		FavoriteType int64 `gorm:"column:favorite_type"`
		Count        int64 `gorm:"column:count"`
	}
	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Select("user_id, favorite_type, COUNT(*) as count").
		Where("user_id IN (?)", userIds).
		Where("target_type = ?", targetType).
		Where("is_deleted = ?", false).
		Group("user_id, favorite_type").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	resultMap := make(map[int64]biz.FavoriteCount)
	for _, uid := range userIds {
		resultMap[uid] = biz.FavoriteCount{LikeCount: 0, DislikeCount: 0, TotalCount: 0}
	}
	for _, res := range results {
		cnt := resultMap[res.UserId]
		if res.FavoriteType == 0 {
			cnt.LikeCount = res.Count
		} else {
			cnt.DislikeCount = res.Count
		}
		cnt.TotalCount = cnt.LikeCount + cnt.DislikeCount
		resultMap[res.UserId] = cnt
	}
	return resultMap, nil
}

func (r *favoriteRepo) GetFavoritesByUserAndTargets(ctx context.Context, userId int64, targetIds []int64, targetType int32) ([]*biz.Favorite, error) {
	if len(targetIds) == 0 {
		return []*biz.Favorite{}, nil
	}
	var dbFavorites []*model.Favorite
	err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("is_deleted = ?", false).
		Find(&dbFavorites).Error
	if err != nil {
		return nil, err
	}
	favorites := make([]*biz.Favorite, 0, len(dbFavorites))
	for _, f := range dbFavorites {
		favorites = append(favorites, r.toBizFavorite(f))
	}
	return favorites, nil
}

func (r *favoriteRepo) BatchGetFavorites(ctx context.Context, userIds, targetIds []int64, targetType int32) ([]*biz.Favorite, error) {
	if len(userIds) == 0 || len(targetIds) == 0 {
		return []*biz.Favorite{}, nil
	}
	var dbFavorites []*model.Favorite
	err := r.data.db.WithContext(ctx).
		Where("user_id IN (?)", userIds).
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("is_deleted = ?", false).
		Find(&dbFavorites).Error
	if err != nil {
		return nil, err
	}
	favorites := make([]*biz.Favorite, 0, len(dbFavorites))
	for _, f := range dbFavorites {
		favorites = append(favorites, r.toBizFavorite(f))
	}
	return favorites, nil
}

func (r *favoriteRepo) GetFavoriteStats(ctx context.Context, targetId int64, targetType int32) (*biz.FavoriteStats, error) {
	var likeCount, dislikeCount int64
	err := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", 0).
		Where("is_deleted = ?", false).
		Count(&likeCount).Error
	if err != nil {
		return nil, err
	}
	err = r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", 1).
		Where("is_deleted = ?", false).
		Count(&dislikeCount).Error
	if err != nil {
		return nil, err
	}
	totalCount := likeCount + dislikeCount
	hotScore := float64(likeCount) - float64(dislikeCount)*0.5
	return &biz.FavoriteStats{
		TargetId:     targetId,
		TargetType:   targetType,
		LikeCount:    likeCount,
		DislikeCount: dislikeCount,
		TotalCount:   totalCount,
		HotScore:     hotScore,
	}, nil
}

func (r *favoriteRepo) BatchGetFavoriteStats(ctx context.Context, targetIds []int64, targetType int32) (map[int64]*biz.FavoriteStats, error) {
	if len(targetIds) == 0 {
		return make(map[int64]*biz.FavoriteStats), nil
	}

	type Result struct {
		TargetId     int64 `gorm:"column:target_id"`
		FavoriteType int64 `gorm:"column:favorite_type"`
		Count        int64 `gorm:"column:count"`
	}
	var results []Result
	err := r.data.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Select("target_id, favorite_type, COUNT(*) as count").
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("is_deleted = ?", false).
		Group("target_id, favorite_type").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	statsMap := make(map[int64]*biz.FavoriteStats)
	for _, tid := range targetIds {
		statsMap[tid] = &biz.FavoriteStats{
			TargetId:     tid,
			TargetType:   targetType,
			LikeCount:    0,
			DislikeCount: 0,
			TotalCount:   0,
			HotScore:     0,
		}
	}
	for _, res := range results {
		stat := statsMap[res.TargetId]
		if res.FavoriteType == 0 {
			stat.LikeCount = res.Count
		} else {
			stat.DislikeCount = res.Count
		}
		stat.TotalCount = stat.LikeCount + stat.DislikeCount
		stat.HotScore = float64(stat.LikeCount) - float64(stat.DislikeCount)*0.5
	}
	return statsMap, nil
}

func (r *favoriteRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		newCtx := context.WithValue(ctx, "db", tx)
		return fn(newCtx)
	})
}

// 转换函数
func (r *favoriteRepo) toBizFavorite(dbFavorite *model.Favorite) *biz.Favorite {
	return &biz.Favorite{
		Id:           dbFavorite.Id,
		UserId:       dbFavorite.UserId,
		TargetType:   int32(dbFavorite.TargetType),
		TargetId:     dbFavorite.TargetId,
		FavoriteType: int32(dbFavorite.FavoriteType),
		IsDeleted:    dbFavorite.IsDeleted,
		CreatedAt:    dbFavorite.CreatedAt,
		UpdatedAt:    dbFavorite.UpdatedAt,
	}
}
