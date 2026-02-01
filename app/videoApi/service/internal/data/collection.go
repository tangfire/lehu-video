package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *CoreAdapterImpl) IsCollected(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error) {
	resp, err := r.collection.IsCollected(ctx, &core.IsCollectedReq{
		UserId:      userId,
		VideoIdList: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool)
	if len(resp.VideoIdList) == 0 {
		return result, nil
	}

	for _, item := range resp.VideoIdList {
		result[item] = true
	}

	return result, nil
}

func (r *CoreAdapterImpl) CountCollected4Video(ctx context.Context, videoIdList []string) (map[string]int64, error) {
	resp, err := r.collection.CountCollect4Video(ctx, &core.CountCollect4VideoReq{
		VideoIdList: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for _, item := range resp.CountResult {
		result[item.Id] = item.Count
	}

	return result, nil
}

func (r *CoreAdapterImpl) GetCollectionById(ctx context.Context, collectionId string) (*biz.Collection, error) {
	resp, err := r.collection.GetCollectionById(ctx, &core.GetCollectionByIdReq{
		Id: collectionId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	collection := resp.Collection
	retCollection := &biz.Collection{
		Id:          collection.Id,
		UserId:      collection.UserId,
		Name:        collection.Name,
		Description: collection.Description,
	}
	return retCollection, nil
}
func (r *CoreAdapterImpl) AddVideo2Collection(ctx context.Context, userId string, collectionId string, videoId string) error {
	resp, err := r.collection.AddVideo2Collection(ctx, &core.AddVideo2CollectionReq{
		CollectionId: collectionId,
		VideoId:      videoId,
		UserId:       userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
func (r *CoreAdapterImpl) AddCollection(ctx context.Context, collection *biz.Collection) error {
	resp, err := r.collection.CreateCollection(ctx, &core.CreateCollectionReq{
		Name:        collection.Name,
		Description: collection.Description,
		UserId:      collection.UserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
func (r *CoreAdapterImpl) ListCollection(ctx context.Context, userId string, pageStats *biz.PageStats) (int64, []*biz.Collection, error) {
	resp, err := r.collection.ListCollection(ctx, &core.ListCollectionReq{
		UserId: userId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	collections := resp.Collections
	var retCollections []*biz.Collection
	for _, collection := range collections {
		retCollections = append(retCollections, &biz.Collection{
			Id:          collection.Id,
			UserId:      collection.UserId,
			Name:        collection.Name,
			Description: collection.Description,
		})
	}
	return int64(resp.PageStats.Total), retCollections, nil
}

func (r *CoreAdapterImpl) ListVideo4Collection(ctx context.Context, collectionId string, pageStats *biz.PageStats) (int64, []string, error) {
	resp, err := r.collection.ListVideo4Collection(ctx, &core.ListVideo4CollectionReq{
		CollectionId: collectionId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	return int64(resp.PageStats.Total), resp.VideoIdList, nil
}

func (r *CoreAdapterImpl) RemoveCollection(ctx context.Context, userId, collectionId string) error {
	resp, err := r.collection.RemoveCollection(ctx, &core.RemoveCollectionReq{
		Id:     collectionId,
		UserId: userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) RemoveVideo4Collection(ctx context.Context, userId string, collectionId string, videoId string) error {
	resp, err := r.collection.RemoveVideoFromCollection(ctx, &core.RemoveVideoFromCollectionReq{
		CollectionId: collectionId,
		VideoId:      videoId,
		UserId:       userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) UpdateCollection(ctx context.Context, collection *biz.Collection) error {
	resp, err := r.collection.UpdateCollection(ctx, &core.UpdateCollectionReq{
		Id:          collection.Id,
		Name:        collection.Name,
		Description: collection.Description,
		UserId:      collection.UserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
