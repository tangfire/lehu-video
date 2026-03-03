package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/data/model"
	"time"
)

type collectionRepo struct {
	data *Data
	log  *log.Helper
}

func NewCollectionRepo(data *Data, logger log.Logger) biz.CollectionRepo {
	return &collectionRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// db 返回当前上下文中的数据库连接，支持事务
func (r *collectionRepo) db(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("db").(*gorm.DB); ok {
		return tx
	}
	return r.data.db.WithContext(ctx)
}

// WithTransaction 实现事务支持
func (r *collectionRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务对象存入上下文，键为 "db"，以便被 db() 方法识别
		newCtx := context.WithValue(ctx, "db", tx)
		return fn(newCtx)
	})
}

// 以下方法全部使用 r.db(ctx) 代替 r.data.db.WithContext(ctx)

func (r *collectionRepo) CreateCollection(ctx context.Context, collection *biz.Collection) error {
	dbCollection := model.Collection{
		Id:          collection.Id,
		UserId:      collection.UserId,
		Title:       collection.Title,
		Description: collection.Description,
		IsDeleted:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return r.db(ctx).Create(&dbCollection).Error
}

func (r *collectionRepo) GetCollectionById(ctx context.Context, id int64) (*biz.Collection, error) {
	var dbCollection model.Collection
	err := r.db(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&dbCollection).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.Collection{
		Id:          dbCollection.Id,
		UserId:      dbCollection.UserId,
		Title:       dbCollection.Title,
		Description: dbCollection.Description,
	}, nil
}

func (r *collectionRepo) GetCollectionByUserIdAndId(ctx context.Context, userId, id int64) (*biz.Collection, error) {
	var dbCollection model.Collection
	err := r.db(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = ?", id, userId, false).
		First(&dbCollection).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.Collection{
		Id:          dbCollection.Id,
		UserId:      dbCollection.UserId,
		Title:       dbCollection.Title,
		Description: dbCollection.Description,
	}, nil
}

func (r *collectionRepo) DeleteCollection(ctx context.Context, id int64) error {
	return r.db(ctx).
		Model(&model.Collection{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

func (r *collectionRepo) ListCollectionsByUserId(ctx context.Context, userId int64, offset, limit int) ([]*biz.Collection, error) {
	var dbCollections []*model.Collection

	query := r.db(ctx).
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&dbCollections).Error
	if err != nil {
		return nil, err
	}

	collections := make([]*biz.Collection, 0, len(dbCollections))
	for _, dbCollection := range dbCollections {
		collections = append(collections, &biz.Collection{
			Id:          dbCollection.Id,
			UserId:      dbCollection.UserId,
			Title:       dbCollection.Title,
			Description: dbCollection.Description,
		})
	}

	return collections, nil
}

func (r *collectionRepo) CountCollectionsByUserId(ctx context.Context, userId int64) (int64, error) {
	var count int64
	err := r.db(ctx).
		Model(&model.Collection{}).
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Count(&count).Error

	return count, err
}

func (r *collectionRepo) UpdateCollection(ctx context.Context, collection *biz.Collection) error {
	return r.db(ctx).
		Model(&model.Collection{}).
		Where("id = ?", collection.Id).
		Updates(map[string]interface{}{
			"title":       collection.Title,
			"description": collection.Description,
			"updated_at":  time.Now(),
		}).Error
}

func (r *collectionRepo) CreateCollectionVideo(ctx context.Context, relation *biz.CollectionVideoRelation) error {
	dbRelation := model.CollectionVideo{
		Id:           relation.Id,
		CollectionId: relation.CollectionId,
		UserId:       relation.UserId,
		VideoId:      relation.VideoId,
		IsDeleted:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return r.db(ctx).Create(&dbRelation).Error
}

func (r *collectionRepo) GetCollectionVideo(ctx context.Context, userId, collectionId, videoId int64) (*biz.CollectionVideoRelation, error) {
	var dbRelation model.CollectionVideo
	err := r.db(ctx).
		Where("user_id = ? AND collection_id = ? AND video_id = ? AND is_deleted = ?",
			userId, collectionId, videoId, false).
		First(&dbRelation).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &biz.CollectionVideoRelation{
		Id:           dbRelation.Id,
		CollectionId: dbRelation.CollectionId,
		UserId:       dbRelation.UserId,
		VideoId:      dbRelation.VideoId,
	}, nil
}

func (r *collectionRepo) DeleteCollectionVideo(ctx context.Context, relationId int64) error {
	return r.db(ctx).
		Model(&model.CollectionVideo{}).
		Where("id = ?", relationId).
		Update("is_deleted", true).Error
}

func (r *collectionRepo) ListVideoIdsByCollectionId(ctx context.Context, collectionId int64, offset, limit int) (int64, []int64, error) {
	var videoIds []int64

	var count int64
	err := r.db(ctx).
		Model(&model.CollectionVideo{}).
		Where("collection_id = ? AND is_deleted = ?", collectionId, false).
		Count(&count).Error

	query := r.db(ctx).
		Model(&model.CollectionVideo{}).
		Select("video_id").
		Where("collection_id = ? AND is_deleted = ?", collectionId, false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err = query.Pluck("video_id", &videoIds).Error
	return count, videoIds, err
}

func (r *collectionRepo) CountCollectionsByVideoId(ctx context.Context, videoId int64) (int64, error) {
	var count int64
	err := r.db(ctx).
		Model(&model.CollectionVideo{}).
		Where("video_id = ? AND is_deleted = ?", videoId, false).
		Count(&count).Error

	return count, err
}

func (r *collectionRepo) BatchCountCollectionsByVideoId(ctx context.Context, videoIds []int64) (map[int64]int64, error) {
	if len(videoIds) == 0 {
		return make(map[int64]int64), nil
	}

	var results []struct {
		VideoID int64
		Count   int64
	}

	err := r.db(ctx).
		Model(&model.CollectionVideo{}).
		Select("video_id, COUNT(*) as count").
		Where("video_id IN (?) AND is_deleted = ?", videoIds, false).
		Group("video_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	countMap := make(map[int64]int64, len(results))
	for _, rc := range results {
		countMap[rc.VideoID] = rc.Count
	}
	return countMap, nil
}

func (r *collectionRepo) ListCollectedVideoIds(ctx context.Context, userId int64, videoIds []int64) ([]int64, error) {
	var collectedVideoIds []int64

	err := r.db(ctx).
		Model(&model.CollectionVideo{}).
		Select("video_id").
		Where("user_id = ? AND video_id IN ? AND is_deleted = ?", userId, videoIds, false).
		Pluck("video_id", &collectedVideoIds).Error

	return collectedVideoIds, err
}
