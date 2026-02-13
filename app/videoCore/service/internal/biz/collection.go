package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type Collection struct {
	Id          int64
	UserId      int64
	Title       string
	Description string
}

func (c *Collection) SetId() {
	c.Id = int64(uuid.New().ID())
}

// ✅ 保持Command/Query/Result模式
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
	UserId       int64 // 添加用户ID用于权限验证
	Name         string
	Description  string
}

type UpdateCollectionResult struct{}

type RemoveCollectionCommand struct {
	CollectionId int64
	UserId       int64 // 添加用户ID用于权限验证
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

// 领域对象
type CollectionVideoRelation struct {
	Id           int64
	CollectionId int64
	UserId       int64
	VideoId      int64
}

// 简化的仓储接口 - 只做数据访问
type CollectionRepo interface {
	// 收藏夹CRUD
	CreateCollection(ctx context.Context, collection *Collection) error
	GetCollectionById(ctx context.Context, id int64) (*Collection, error)
	GetCollectionByUserIdAndId(ctx context.Context, userId, id int64) (*Collection, error)
	DeleteCollection(ctx context.Context, id int64) error
	ListCollectionsByUserId(ctx context.Context, userId int64, offset, limit int) ([]*Collection, error)
	CountCollectionsByUserId(ctx context.Context, userId int64) (int64, error)
	UpdateCollection(ctx context.Context, collection *Collection) error

	// 收藏夹与视频关系
	CreateCollectionVideo(ctx context.Context, relation *CollectionVideoRelation) error
	GetCollectionVideo(ctx context.Context, userId, collectionId, videoId int64) (*CollectionVideoRelation, error)
	DeleteCollectionVideo(ctx context.Context, relationId int64) error
	ListVideoIdsByCollectionId(ctx context.Context, collectionId int64, offset, limit int) ([]int64, error)
	CountVideosByCollectionId(ctx context.Context, collectionId int64) (int64, error)
	CountCollectionsByVideoId(ctx context.Context, videoId int64) (int64, error)
	ListCollectedVideoIds(ctx context.Context, userId int64, videoIds []int64) ([]int64, error)
}

type CollectionUsecase struct {
	repo        CollectionRepo
	counterRepo CounterRepo // 新增
	log         *log.Helper
}

func NewCollectionUsecase(repo CollectionRepo, counterRepo CounterRepo, logger log.Logger) *CollectionUsecase {
	return &CollectionUsecase{
		repo:        repo,
		counterRepo: counterRepo,
		log:         log.NewHelper(logger),
	}
}

func (uc *CollectionUsecase) CreateCollection(ctx context.Context, cmd *CreateCollectionCommand) (*CreateCollectionResult, error) {
	// 业务验证
	if cmd.Name == "" {
		return nil, fmt.Errorf("收藏夹名称不能为空")
	}

	if cmd.Name == "默认收藏夹" {
		return nil, fmt.Errorf("命名不能为默认收藏夹")
	}

	// 业务逻辑：创建收藏夹
	collection := &Collection{
		UserId:      cmd.UserId,
		Title:       cmd.Name,
		Description: cmd.Description,
	}
	collection.SetId()

	// 业务逻辑：确保用户有默认收藏夹
	_, err := uc.ensureUserHasDefaultCollection(ctx, cmd.UserId)
	if err != nil {
		return nil, fmt.Errorf("创建默认收藏夹失败: %v", err)
	}

	// 如果是第一个自定义收藏夹，可以添加特殊逻辑
	// ...

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

	if collection == nil {
		return &GetCollectionByIdResult{Collection: nil}, nil
	}

	return &GetCollectionByIdResult{Collection: collection}, nil
}

func (uc *CollectionUsecase) RemoveCollection(ctx context.Context, cmd *RemoveCollectionCommand) (*RemoveCollectionResult, error) {
	// todo 其实这里可以放在api服务进行校验是不是自己的收藏夹
	// 业务逻辑：权限验证 - 只能删除自己的收藏夹
	collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, cmd.CollectionId)
	if err != nil {
		return nil, fmt.Errorf("查询收藏夹失败: %v", err)
	}

	if collection == nil {
		return nil, fmt.Errorf("收藏夹不存在或无权操作")
	}

	// 业务逻辑：不能删除默认收藏夹
	if collection.Title == "默认收藏夹" {
		return nil, fmt.Errorf("不能删除默认收藏夹")
	}

	// 业务逻辑：检查收藏夹是否为空（可选）
	// ...

	err = uc.repo.DeleteCollection(ctx, cmd.CollectionId)
	if err != nil {
		uc.log.Errorf("删除收藏夹失败: %v", err)
		return nil, err
	}

	return &RemoveCollectionResult{}, nil
}

func (uc *CollectionUsecase) ListCollection(ctx context.Context, query *ListCollectionQuery) (*ListCollectionResult, error) {
	// 业务逻辑：分页参数验证
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

	// 业务逻辑：确保用户有默认收藏夹
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
	// 业务验证
	if cmd.Name == "" {
		return nil, fmt.Errorf("收藏夹名称不能为空")
	}

	// 业务逻辑：权限验证
	collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, cmd.CollectionId)
	if err != nil {
		return nil, fmt.Errorf("查询收藏夹失败: %v", err)
	}

	if collection == nil {
		return nil, fmt.Errorf("收藏夹不存在或无权操作")
	}

	// 业务逻辑：不能修改默认收藏夹的名称（可选）
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

func (uc *CollectionUsecase) AddVideoToCollection(ctx context.Context, cmd *AddVideoToCollectionCommand) (*AddVideoToCollectionResult, error) {
	collectionId := cmd.CollectionId
	if collectionId == 0 {
		defaultCollection, err := uc.ensureUserHasDefaultCollection(ctx, cmd.UserId)
		if err != nil {
			return nil, fmt.Errorf("获取默认收藏夹失败: %v", err)
		}
		collectionId = defaultCollection.Id
	}

	collection, err := uc.repo.GetCollectionByUserIdAndId(ctx, cmd.UserId, collectionId)
	if err != nil {
		return nil, fmt.Errorf("查询收藏夹失败: %v", err)
	}
	if collection == nil {
		return nil, fmt.Errorf("收藏夹不存在或无权操作")
	}

	existingRelation, err := uc.repo.GetCollectionVideo(ctx, cmd.UserId, collectionId, cmd.VideoId)
	if err != nil {
		return nil, fmt.Errorf("检查收藏关系失败: %v", err)
	}
	if existingRelation != nil {
		// 已经收藏，直接返回成功（幂等性）
		return &AddVideoToCollectionResult{}, nil
	}

	relation := &CollectionVideoRelation{
		CollectionId: collectionId,
		UserId:       cmd.UserId,
		VideoId:      cmd.VideoId,
	}
	relation.Id = int64(uuid.New().ID())

	err = uc.repo.CreateCollectionVideo(ctx, relation)
	if err != nil {
		uc.log.Errorf("添加视频到收藏夹失败: %v", err)
		return nil, err
	}

	// ---------- 新增：增加用户 favorite_count ----------
	if _, err := uc.counterRepo.IncrUserCounter(ctx, cmd.UserId, "favorite_count", 1); err != nil {
		uc.log.Warnf("增加用户 favorite_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}
	// ------------------------------------------------

	return &AddVideoToCollectionResult{}, nil
}

func (uc *CollectionUsecase) RemoveVideoFromCollection(ctx context.Context, cmd *RemoveVideoFromCollectionCommand) (*RemoveVideoFromCollectionResult, error) {
	collectionId := cmd.CollectionId
	if collectionId == 0 {
		defaultCollection, err := uc.ensureUserHasDefaultCollection(ctx, cmd.UserId)
		if err != nil {
			return nil, fmt.Errorf("获取默认收藏夹失败: %v", err)
		}
		collectionId = defaultCollection.Id
	}

	relation, err := uc.repo.GetCollectionVideo(ctx, cmd.UserId, collectionId, cmd.VideoId)
	if err != nil {
		return nil, fmt.Errorf("检查收藏关系失败: %v", err)
	}
	if relation == nil {
		return &RemoveVideoFromCollectionResult{}, nil
	}

	err = uc.repo.DeleteCollectionVideo(ctx, relation.Id)
	if err != nil {
		uc.log.Errorf("从收藏夹移除视频失败: %v", err)
		return nil, err
	}

	// ---------- 新增：减少用户 favorite_count ----------
	if _, err := uc.counterRepo.DecrUserCounter(ctx, cmd.UserId, "favorite_count", 1); err != nil {
		uc.log.Warnf("减少用户 favorite_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}
	// ------------------------------------------------

	return &RemoveVideoFromCollectionResult{}, nil
}

func (uc *CollectionUsecase) ListVideo4Collection(ctx context.Context, query *ListVideo4CollectionQuery) (*ListVideo4CollectionResult, error) {
	// 业务逻辑：分页参数验证
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

	// todo 这两步应该可以合成一步的，一般我们ListVideoIdsByCollectionId的时候，就要获取total了
	videoIds, err := uc.repo.ListVideoIdsByCollectionId(ctx, query.CollectionId, int(offset), int(query.PageStats.PageSize))
	if err != nil {
		uc.log.Errorf("查询收藏夹视频列表失败: %v", err)
		return nil, err
	}

	total, err := uc.repo.CountVideosByCollectionId(ctx, query.CollectionId)
	if err != nil {
		uc.log.Errorf("统计收藏夹视频数量失败: %v", err)
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

	var counts []*CountResult
	for _, videoId := range query.VideoIds {
		count, err := uc.repo.CountCollectionsByVideoId(ctx, videoId)
		if err != nil {
			uc.log.Errorf("统计视频收藏数失败: %v", err)
			return nil, err
		}

		counts = append(counts, &CountResult{
			Id:    videoId,
			Count: count,
		})
	}

	return &CountCollect4VideoResult{
		Counts: counts,
	}, nil
}

// 内部方法：确保用户有默认收藏夹
func (uc *CollectionUsecase) ensureUserHasDefaultCollection(ctx context.Context, userId int64) (*Collection, error) {
	// 查询用户的所有收藏夹
	collections, err := uc.repo.ListCollectionsByUserId(ctx, userId, 0, 10)
	if err != nil {
		return nil, err
	}

	// 检查是否有默认收藏夹
	for _, collection := range collections {
		if collection.Title == "默认收藏夹" {
			return collection, nil
		}
	}

	// 如果没有，创建默认收藏夹
	defaultCollection := &Collection{
		UserId:      userId,
		Title:       "默认收藏夹",
		Description: "默认收藏夹",
	}
	defaultCollection.SetId()

	err = uc.repo.CreateCollection(ctx, defaultCollection)
	if err != nil {
		return nil, err
	}

	return defaultCollection, nil
}
