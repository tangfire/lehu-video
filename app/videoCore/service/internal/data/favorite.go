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

// db 优先返回事务中的数据库连接
func (r *favoriteRepo) db(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("db").(*gorm.DB); ok {
		return tx
	}
	return r.data.db.WithContext(ctx)
}

// toBizFavorite 将 model 转换为 biz
func (r *favoriteRepo) toBizFavorite(m *model.Favorite) *biz.Favorite {
	return &biz.Favorite{
		Id:           m.Id,
		UserId:       m.UserId,
		TargetType:   int32(m.TargetType),
		TargetId:     m.TargetId,
		FavoriteType: int32(m.FavoriteType),
		DeleteAt:     m.DeleteAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// toDBFavorite 将 biz 转换为 model
func (r *favoriteRepo) toDBFavorite(b *biz.Favorite) *model.Favorite {
	return &model.Favorite{
		Id:           b.Id,
		UserId:       b.UserId,
		TargetType:   int64(b.TargetType),
		TargetId:     b.TargetId,
		FavoriteType: int64(b.FavoriteType),
		DeleteAt:     b.DeleteAt,
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
}

func (r *favoriteRepo) CreateFavorite(ctx context.Context, favorite *biz.Favorite) error {
	return r.db(ctx).Create(r.toDBFavorite(favorite)).Error
}

func (r *favoriteRepo) UpdateFavorite(ctx context.Context, favorite *biz.Favorite) error {
	return r.db(ctx).Model(&model.Favorite{}).Where("id = ?", favorite.Id).
		Updates(map[string]interface{}{
			"favorite_type": favorite.FavoriteType,
			"delete_at":     favorite.DeleteAt,
			"updated_at":    favorite.UpdatedAt,
		}).Error
}

func (r *favoriteRepo) GetFavorite(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (*biz.Favorite, error) {
	var m model.Favorite
	err := r.db(ctx).
		Where("user_id = ?", userId).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("favorite_type = ?", favoriteType).
		Where("delete_at = ?", 0).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizFavorite(&m), nil
}

func (r *favoriteRepo) GetFavoriteByUserTarget(ctx context.Context, userId, targetId int64, targetType int32) (*biz.Favorite, error) {
	var m model.Favorite
	err := r.db(ctx).
		Where("user_id = ?", userId).
		Where("target_id = ?", targetId).
		Where("target_type = ?", targetType).
		Where("delete_at = ?", 0).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBizFavorite(&m), nil
}

func (r *favoriteRepo) SoftDeleteFavorite(ctx context.Context, favoriteId int64) error {
	return r.db(ctx).Model(&model.Favorite{}).
		Where("id = ?", favoriteId).
		Update("delete_at", time.Now().Unix()).Error
}

func (r *favoriteRepo) HardDeleteFavorite(ctx context.Context, favoriteId int64) error {
	return r.db(ctx).Where("id = ?", favoriteId).Delete(&model.Favorite{}).Error
}

func (r *favoriteRepo) ListFavorites(ctx context.Context, query *biz.ListFavoriteQuery) ([]*biz.Favorite, int64, error) {
	var ms []*model.Favorite
	var total int64

	db := r.db(ctx).Model(&model.Favorite{})

	if !query.IncludeDeleted {
		db = db.Where("delete_at = ?", 0)
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
		Find(&ms).Error
	if err != nil {
		return nil, 0, err
	}

	favorites := make([]*biz.Favorite, 0, len(ms))
	for _, m := range ms {
		favorites = append(favorites, r.toBizFavorite(m))
	}
	return favorites, total, nil
}

func (r *favoriteRepo) CountFavorites(ctx context.Context, userId, targetId int64, targetType, favoriteType int32) (int64, error) {
	var count int64
	db := r.db(ctx).Model(&model.Favorite{}).Where("delete_at = ?", 0)
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
	err := r.db(ctx).
		Model(&model.Favorite{}).
		Select("target_id, favorite_type, COUNT(*) as count").
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("delete_at = ?", 0).
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
	err := r.db(ctx).
		Model(&model.Favorite{}).
		Select("user_id, favorite_type, COUNT(*) as count").
		Where("user_id IN (?)", userIds).
		Where("target_type = ?", targetType).
		Where("delete_at = ?", 0).
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
	var ms []*model.Favorite
	err := r.db(ctx).
		Where("user_id = ?", userId).
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("delete_at = ?", 0).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	favorites := make([]*biz.Favorite, 0, len(ms))
	for _, m := range ms {
		favorites = append(favorites, r.toBizFavorite(m))
	}
	return favorites, nil
}

func (r *favoriteRepo) BatchGetFavorites(ctx context.Context, userIds, targetIds []int64, targetType int32) ([]*biz.Favorite, error) {
	if len(userIds) == 0 || len(targetIds) == 0 {
		return []*biz.Favorite{}, nil
	}
	var ms []*model.Favorite
	err := r.db(ctx).
		Where("user_id IN (?)", userIds).
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("delete_at = ?", 0).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	favorites := make([]*biz.Favorite, 0, len(ms))
	for _, m := range ms {
		favorites = append(favorites, r.toBizFavorite(m))
	}
	return favorites, nil
}

func (r *favoriteRepo) GetFavoriteStats(ctx context.Context, targetId int64, targetType int32) (*biz.FavoriteStats, error) {
	statsMap, err := r.BatchGetFavoriteStats(ctx, []int64{targetId}, targetType)
	if err != nil {
		return nil, err
	}
	return statsMap[targetId], nil
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
	err := r.db(ctx).
		Model(&model.Favorite{}).
		Select("target_id, favorite_type, COUNT(*) as count").
		Where("target_id IN (?)", targetIds).
		Where("target_type = ?", targetType).
		Where("delete_at = ?", 0).
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
