package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoCore/service/internal/pkg/idgen"
)

type Collection struct {
	Id          int64
	UserId      int64
	Title       string
	Description string
}

type CreateCollectionCommand struct {
	UserId      int64
	Name        string
	Description string
}

type CreateCollectionResult struct {
	CollectionId int64
}

type UpdateCollectionCommand struct {
	CollectionId int64
	UserId       int64
	Name         string
	Description  string
}

type UpdateCollectionResult struct{}

type RemoveCollectionCommand struct {
	CollectionId int64
	UserId       int64
}

type RemoveCollectionResult struct{}

type GetCollectionByIdQuery struct {
	CollectionId int64
}

type GetCollectionByIdResult struct {
	Collection *Collection
}

type ListCollectionQuery struct {
	UserId    int64
	PageStats PageStats
}

type ListCollectionResult struct {
	Collections []*Collection
	Total       int64
}

type AddVideoToCollectionCommand struct {
	UserId       int64
	CollectionId int64
	VideoId      int64
}

type AddVideoToCollectionResult struct{}

type RemoveVideoFromCollectionCommand struct {
	UserId       int64
	CollectionId int64
	VideoId      int64
}

type RemoveVideoFromCollectionResult struct{}

type ListVideo4CollectionQuery struct {
	CollectionId int64
	PageStats    PageStats
}

type ListVideo4CollectionResult struct {
	VideoIds []int64
	Total    int64
}

type IsCollectedQuery struct {
	UserId   int64
	VideoIds []int64
}

type IsCollectedResult struct {
	CollectedVideoIds []int64
}

type CountCollect4VideoQuery struct {
	VideoIds []int64
}

type CountResult struct {
	Id    int64
	Count int64
}

type CountCollect4VideoResult struct {
	Counts []*CountResult
}

type CollectionVideoRelation struct {
	Id           int64
	CollectionId int64
	UserId       int64
	VideoId      int64
}

type CollectionRepo interface {
	CreateCollection(ctx context.Context, collection *Collection) error
	GetCollectionById(ctx context.Context, id int64) (*Collection, error)
	GetCollectionByUserIdAndId(ctx context.Context, userId, id int64) (*Collection, error)
	DeleteCollection(ctx context.Context, id int64) error
	ListCollectionsByUserId(ctx context.Context, userId int64, offset, limit int) ([]*Collection, error)
	CountCollectionsByUserId(ctx context.Context, userId int64) (int64, error)
	UpdateCollection(ctx context.Context, collection *Collection) error

	CreateCollectionVideo(ctx context.Context, relation *CollectionVideoRelation) error
	GetCollectionVideo(ctx context.Context, userId, collectionId, videoId int64) (*CollectionVideoRelation, error)
	DeleteCollectionVideo(ctx context.Context, relationId int64) error
	ListVideoIdsByCollectionId(ctx context.Context, collectionId int64, offset, limit int) (int64, []int64, error)
	CountCollectionsByVideoId(ctx context.Context, videoId int64) (int64, error)
	BatchCountCollectionsByVideoId(ctx context.Context, videoIds []int64) (map[int64]int64, error)
	ListCollectedVideoIds(ctx context.Context, userId int64, videoIds []int64) ([]int64, error)

	// 事务支持
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type CollectionUsecase struct {
	repo         CollectionRepo
	videoRepo    VideoRepo // 新增
	userCounter  UserCounterRepo
	videoCounter VideoCounterRepo // 新增
	idGen        idgen.Generator
	log          *log.Helper
}

func NewCollectionUsecase(repo CollectionRepo, videoRepo VideoRepo, userCounter UserCounterRepo, videoCounter VideoCounterRepo, idGen idgen.Generator, logger log.Logger) *CollectionUsecase {
	return &CollectionUsecase{
		repo:         repo,
		videoRepo:    videoRepo,
		userCounter:  userCounter,
		videoCounter: videoCounter,
		idGen:        idGen,
		log:          log.NewHelper(logger),
	}
}

func (uc *CollectionUsecase) CreateCollection(ctx context.Context, cmd *CreateCollectionCommand) (*CreateCollectionResult, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("收藏夹名称不能为空")
	}
	if cmd.Name == "默认收藏夹" {
		return nil, fmt.Errorf("命名不能为默认收藏夹")
	}

	collection := &Collection{
		Id:          uc.idGen.NextID(),
		UserId:      cmd.UserId,
		Title:       cmd.Name,
		Description: cmd.Description,
	}

	_, err := uc.ensureUserHasDefaultCollection(ctx, cmd.UserId)
	if err != nil {
		return nil, fmt.Errorf("创建默认收藏夹失败: %v", err)
	}

	err = uc.repo.CreateCollection(ctx, collection)
	if err != nil {
		uc.log.Errorf("创建收藏夹失败: %v", err)
		return nil, err
	}

	return &CreateCollectionResult{
		CollectionId: collection.Id,
	}, nil
}

func (uc *CollectionUsecase) GetCollectionById(ctx context.Context, query *GetCollectionByIdQuery) (*GetCollectionByIdResult, error) {
	collection, err := uc.repo.GetCollectionById(ctx, query.CollectionId)
	if err != nil {
		uc.log.Errorf("查询收藏夹失败: %v", err)
		return nil, err
	}
	return &GetCollectionByIdResult{Collection: collection}, nil
}

func (uc *CollectionUsecase) RemoveCollection(ctx context.Context, cmd *RemoveCollectionCommand) (*RemoveCollectionResult, error) {
	collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, cmd.CollectionId)
	if err != nil {
		return nil, fmt.Errorf("查询收藏夹失败: %v", err)
	}
	if collection == nil {
		return nil, fmt.Errorf("收藏夹不存在或无权操作")
	}
	if collection.Title == "默认收藏夹" {
		return nil, fmt.Errorf("不能删除默认收藏夹")
	}

	err = uc.repo.DeleteCollection(ctx, cmd.CollectionId)
	if err != nil {
		uc.log.Errorf("删除收藏夹失败: %v", err)
		return nil, err
	}
	return &RemoveCollectionResult{}, nil
}

func (uc *CollectionUsecase) ListCollection(ctx context.Context, query *ListCollectionQuery) (*ListCollectionResult, error) {
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}
	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize

	_, err := uc.ensureUserHasDefaultCollection(ctx, query.UserId)
	if err != nil {
		return nil, fmt.Errorf("创建默认收藏夹失败: %v", err)
	}

	collections, err := uc.repo.ListCollectionsByUserId(ctx, query.UserId, int(offset), int(query.PageStats.PageSize))
	if err != nil {
		uc.log.Errorf("查询收藏夹列表失败: %v", err)
		return nil, err
	}
	total, err := uc.repo.CountCollectionsByUserId(ctx, query.UserId)
	if err != nil {
		uc.log.Errorf("统计收藏夹数量失败: %v", err)
		return nil, err
	}
	return &ListCollectionResult{
		Collections: collections,
		Total:       total,
	}, nil
}

func (uc *CollectionUsecase) UpdateCollection(ctx context.Context, cmd *UpdateCollectionCommand) (*UpdateCollectionResult, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("收藏夹名称不能为空")
	}
	collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, cmd.CollectionId)
	if err != nil {
		return nil, fmt.Errorf("查询收藏夹失败: %v", err)
	}
	if collection == nil {
		return nil, fmt.Errorf("收藏夹不存在或无权操作")
	}
	if collection.Title == "默认收藏夹" && cmd.Name != "默认收藏夹" {
		return nil, fmt.Errorf("不能修改默认收藏夹的名称")
	}
	collection.Title = cmd.Name
	collection.Description = cmd.Description
	err = uc.repo.UpdateCollection(ctx, collection)
	if err != nil {
		uc.log.Errorf("更新收藏夹失败: %v", err)
		return nil, err
	}
	return &UpdateCollectionResult{}, nil
}

// AddVideoToCollection 添加视频到收藏夹
func (uc *CollectionUsecase) AddVideoToCollection(ctx context.Context, cmd *AddVideoToCollectionCommand) (*AddVideoToCollectionResult, error) {
	collectionId := cmd.CollectionId
	if collectionId == 0 {
		defaultCollection, err := uc.ensureUserHasDefaultCollection(ctx, cmd.UserId)
		if err != nil {
			return nil, fmt.Errorf("获取默认收藏夹失败: %v", err)
		}
		collectionId = defaultCollection.Id
	}

	// 数据库操作（事务内）
	err := uc.repo.WithTransaction(ctx, func(ctx context.Context) error {
		// 重新查询收藏夹（在事务内）
		collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, collectionId)
		if err != nil {
			return fmt.Errorf("查询收藏夹失败: %v", err)
		}
		if collection == nil {
			return fmt.Errorf("收藏夹不存在或无权操作")
		}

		existing, err := uc.repo.GetCollectionVideo(ctx, cmd.UserId, collectionId, cmd.VideoId)
		if err != nil {
			return fmt.Errorf("检查收藏关系失败: %v", err)
		}
		if existing != nil {
			// 已收藏，幂等返回
			return nil
		}

		// 创建收藏关系
		relation := &CollectionVideoRelation{
			Id:           uc.idGen.NextID(),
			CollectionId: collectionId,
			UserId:       cmd.UserId,
			VideoId:      cmd.VideoId,
		}
		if err := uc.repo.CreateCollectionVideo(ctx, relation); err != nil {
			return fmt.Errorf("创建收藏关系失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 事务成功后，更新计数
	// 更新视频收藏计数（使用 videoCounter）
	if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.VideoId, "collection_count", 1); err != nil {
		uc.log.Warnf("更新视频收藏计数失败: videoId=%d, err=%v", cmd.VideoId, err)
	}
	// 更新用户的收藏计数（Redis计数器）
	if _, err := uc.userCounter.IncrUserCounter(ctx, cmd.UserId, "collection_count", 1); err != nil {
		uc.log.Warnf("增加用户 collection_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}

	return &AddVideoToCollectionResult{}, nil
}

// RemoveVideoFromCollection 从收藏夹移除视频
func (uc *CollectionUsecase) RemoveVideoFromCollection(ctx context.Context, cmd *RemoveVideoFromCollectionCommand) (*RemoveVideoFromCollectionResult, error) {
	collectionId := cmd.CollectionId
	if collectionId == 0 {
		defaultCollection, err := uc.ensureUserHasDefaultCollection(ctx, cmd.UserId)
		if err != nil {
			return nil, fmt.Errorf("获取默认收藏夹失败: %v", err)
		}
		collectionId = defaultCollection.Id
	}

	// 数据库操作（事务内）
	err := uc.repo.WithTransaction(ctx, func(ctx context.Context) error {
		collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, collectionId)
		if err != nil {
			return fmt.Errorf("查询收藏夹失败: %v", err)
		}
		if collection == nil {
			return fmt.Errorf("收藏夹不存在或无权操作")
		}

		relation, err := uc.repo.GetCollectionVideo(ctx, cmd.UserId, collectionId, cmd.VideoId)
		if err != nil {
			return fmt.Errorf("检查收藏关系失败: %v", err)
		}
		if relation == nil {
			return nil // 幂等
		}

		if err := uc.repo.DeleteCollectionVideo(ctx, relation.Id); err != nil {
			return fmt.Errorf("删除收藏关系失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 事务成功后，更新计数
	// 更新视频收藏计数
	if err := uc.videoCounter.IncrVideoCounter(ctx, cmd.VideoId, "collection_count", -1); err != nil {
		uc.log.Warnf("更新视频收藏计数失败: videoId=%d, err=%v", cmd.VideoId, err)
	}
	// 更新用户收藏计数
	if _, err := uc.userCounter.IncrUserCounter(ctx, cmd.UserId, "collection_count", -1); err != nil {
		uc.log.Warnf("减少用户 collection_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}

	return &RemoveVideoFromCollectionResult{}, nil
}

func (uc *CollectionUsecase) ListVideo4Collection(ctx context.Context, query *ListVideo4CollectionQuery) (*ListVideo4CollectionResult, error) {
	if query.PageStats.Page < 1 {
		query.PageStats.Page = 1
	}
	if query.PageStats.PageSize <= 0 {
		query.PageStats.PageSize = 20
	}
	if query.PageStats.PageSize > 100 {
		query.PageStats.PageSize = 100
	}
	offset := (query.PageStats.Page - 1) * query.PageStats.PageSize
	total, videoIds, err := uc.repo.ListVideoIdsByCollectionId(ctx, query.CollectionId, int(offset), int(query.PageStats.PageSize))
	if err != nil {
		uc.log.Errorf("查询收藏夹视频列表失败: %v", err)
		return nil, err
	}
	return &ListVideo4CollectionResult{
		VideoIds: videoIds,
		Total:    total,
	}, nil
}

func (uc *CollectionUsecase) IsCollected(ctx context.Context, query *IsCollectedQuery) (*IsCollectedResult, error) {
	if len(query.VideoIds) == 0 {
		return &IsCollectedResult{CollectedVideoIds: []int64{}}, nil
	}
	collectedVideoIds, err := uc.repo.ListCollectedVideoIds(ctx, query.UserId, query.VideoIds)
	if err != nil {
		uc.log.Errorf("检查视频是否被收藏失败: %v", err)
		return nil, err
	}
	return &IsCollectedResult{
		CollectedVideoIds: collectedVideoIds,
	}, nil
}

func (uc *CollectionUsecase) CountCollectedNumber4Video(ctx context.Context, query *CountCollect4VideoQuery) (*CountCollect4VideoResult, error) {
	if len(query.VideoIds) == 0 {
		return &CountCollect4VideoResult{Counts: []*CountResult{}}, nil
	}
	countMap, err := uc.repo.BatchCountCollectionsByVideoId(ctx, query.VideoIds)
	if err != nil {
		uc.log.Errorf("批量统计视频收藏数失败: %v", err)
		return nil, err
	}
	counts := make([]*CountResult, 0, len(query.VideoIds))
	for _, videoId := range query.VideoIds {
		counts = append(counts, &CountResult{
			Id:    videoId,
			Count: countMap[videoId],
		})
	}
	return &CountCollect4VideoResult{Counts: counts}, nil
}

// ensureUserHasDefaultCollection 确保用户有默认收藏夹
func (uc *CollectionUsecase) ensureUserHasDefaultCollection(ctx context.Context, userId int64) (*Collection, error) {
	collections, err := uc.repo.ListCollectionsByUserId(ctx, userId, 0, 10)
	if err != nil {
		return nil, err
	}
	for _, collection := range collections {
		if collection.Title == "默认收藏夹" {
			return collection, nil
		}
	}
	defaultCollection := &Collection{
		Id:          uc.idGen.NextID(),
		UserId:      userId,
		Title:       "默认收藏夹",
		Description: "默认收藏夹",
	}
	err = uc.repo.CreateCollection(ctx, defaultCollection)
	if err != nil {
		return nil, err
	}
	return defaultCollection, nil
}
