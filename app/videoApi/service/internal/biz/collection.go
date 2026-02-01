package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type AddVideo2CollectionInput struct {
	CollectionId string
	VideoId      string
}

type CreateCollectionInput struct {
	Name        string
	Description string
}

type ListCollectionInput struct {
	PageStats *PageStats
}

type ListCollectionOutput struct {
	Collections []*Collection
	Total       int64
}

type ListVideo4CollectionsInput struct {
	CollectionId string
	PageStats    *PageStats
}

type ListVideo4CollectionsOutput struct {
	Videos []*Video
	Total  int64
}

type RemoveVideo4CollectionInput struct {
	CollectionId string
	VideoId      string
}

type UpdateCollectionInput struct {
	Id          string
	Name        string
	Description string
}

type CollectionUsecase struct {
	core      CoreAdapter
	assembler *VideoAssembler
	log       *log.Helper
}

func NewCollectionUsecase(core CoreAdapter, assembler *VideoAssembler, logger log.Logger) *CollectionUsecase {
	return &CollectionUsecase{
		core:      core,
		assembler: assembler,
		log:       log.NewHelper(logger),
	}
}

func (uc *CollectionUsecase) checkCollectionBelongUser(ctx context.Context, collectionId string) error {
	if collectionId == "0" || collectionId == "" {
		log.Context(ctx).Warnf("collectionId is empty")
		return nil
	}

	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}

	data, err := uc.core.GetCollectionById(ctx, collectionId)
	if err != nil {
		log.Context(ctx).Errorf("failed to get collection info: %v", err)
		return errors.New("信息不存在")
	}

	if data.UserId != userId {
		return errors.New("此收藏夹不属于当前用户")
	}

	return nil
}

func (uc *CollectionUsecase) AddVideo2Collection(ctx context.Context, input *AddVideo2CollectionInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return errors.New("获取用户信息失败")
	}
	err = uc.checkCollectionBelongUser(ctx, input.CollectionId)
	if err != nil {
		return err
	}
	err = uc.core.AddVideo2Collection(ctx, userId, input.CollectionId, input.VideoId)
	if err != nil {
		log.Context(ctx).Errorf("failed to add video info: %v", err)
		return errors.New("添加失败")
	}
	return nil
}

func (uc *CollectionUsecase) CreateCollection(ctx context.Context, input *CreateCollectionInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return err
	}
	err = uc.core.AddCollection(ctx, &Collection{
		Id:          "0",
		UserId:      userId,
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		log.Context(ctx).Errorf("failed to add collection info: %v", err)
		return errors.New("创建失败")
	}
	return nil
}

func (uc *CollectionUsecase) ListCollection(ctx context.Context, input *ListCollectionInput) (output *ListCollectionOutput, err error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		log.Context(ctx).Errorf("failed to get user id: %v", err)
		return nil, errors.New("获取用户失败")
	}
	total, collections, err := uc.core.ListCollection(ctx, userId, input.PageStats)
	if err != nil {
		log.Context(ctx).Errorf("failed to list collections: %v", err)
		return nil, errors.New("获取失败")
	}
	return &ListCollectionOutput{
		Collections: collections,
		Total:       total,
	}, nil
}

func (uc *CollectionUsecase) ListVideo4Collection(ctx context.Context, input *ListVideo4CollectionsInput) (output *ListVideo4CollectionsOutput, err error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		log.Context(ctx).Errorf("failed to get user id: %v", err)
		return nil, errors.New("获取用户信息失败")
	}

	err = uc.checkCollectionBelongUser(ctx, input.CollectionId)
	if err != nil {
		log.Context(ctx).Errorf("failed to get collection info: %v", err)
		return nil, err
	}

	total, videoIds, err := uc.core.ListVideo4Collection(ctx, input.CollectionId, input.PageStats)
	if err != nil {
		log.Context(ctx).Errorf("failed to list video info: %v", err)
		return nil, errors.New("获取失败")
	}
	videos, err := uc.core.GetVideoByIdList(ctx, videoIds)
	if err != nil {
		log.Context(ctx).Errorf("failed to get video info: %v", err)
		return nil, errors.New("获取信息失败")
	}

	result, err := uc.assembler.AssembleVideos(ctx, videos, userId)
	if err != nil {
		log.Context(ctx).Warnf("something wrong in assembling videos: %v", err)
	}
	return &ListVideo4CollectionsOutput{
		Videos: result,
		Total:  total,
	}, nil

}

func (uc *CollectionUsecase) RemoveCollection(ctx context.Context, collectionId string) error {
	err := uc.checkCollectionBelongUser(ctx, collectionId)
	if err != nil {
		return err
	}

	userId, err := claims.GetUserId(ctx)
	if err != nil {
		log.Context(ctx).Errorf("failed to get user id: %v", err)
		return errors.New("获取用户信息失败")
	}

	err = uc.core.RemoveCollection(ctx, userId, collectionId)
	if err != nil {
		log.Context(ctx).Errorf("failed to remove collection info: %v", err)
		return errors.New("删除失败")
	}
	return nil
}

func (uc *CollectionUsecase) RemoveVideo4Collection(ctx context.Context, input *RemoveVideo4CollectionInput) error {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		log.Context(ctx).Errorf("failed to get user id: %v", err)
		return errors.New("获取用户信息失败")
	}
	err = uc.checkCollectionBelongUser(ctx, input.CollectionId)
	if err != nil {
		log.Context(ctx).Errorf("failed to get collection info: %v", err)
		return err
	}
	err = uc.core.RemoveVideo4Collection(ctx, userId, input.CollectionId, input.VideoId)
	if err != nil {
		log.Context(ctx).Errorf("failed to remove video info: %v", err)
		return errors.New("删除失败")
	}
	return nil
}

func (uc *CollectionUsecase) UpdateCollection(ctx context.Context, input *UpdateCollectionInput) error {
	err := uc.checkCollectionBelongUser(ctx, input.Id)
	if err != nil {
		return err
	}

	userId, err := claims.GetUserId(ctx)
	if err != nil {
		log.Context(ctx).Errorf("failed to get user id: %v", err)
		return errors.New("获取用户信息失败")
	}
	err = uc.core.UpdateCollection(ctx, &Collection{
		Id:          input.Id,
		UserId:      userId,
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		log.Context(ctx).Errorf("failed to update collection info: %v", err)
		return err
	}
	return nil

}
