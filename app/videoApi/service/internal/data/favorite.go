package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *CoreAdapterImpl) CountBeFavoriteNumber4User(ctx context.Context, userId string) (int64, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		Ids:           []string{userId},
		AggregateType: core.FavoriteAggregateType_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.Items[0].Count, nil
}

func (r *CoreAdapterImpl) CountFavorite4Video(ctx context.Context, videoIdList []string) (map[string]int64, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		AggregateType: core.FavoriteAggregateType_BY_VIDEO,
		Ids:           videoIdList,
		FavoriteType:  core.FavoriteType_FAVORITE,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for _, item := range resp.Items {
		result[item.BizId] = item.Count
	}
	return result, nil
}

func (r *CoreAdapterImpl) AddFavorite(ctx context.Context, id, userId string, target *biz.FavoriteTarget, _type *biz.FavoriteType) error {

	resp, err := r.favorite.AddFavorite(ctx, &core.AddFavoriteReq{
		Target: core.FavoriteTarget(*target), // 类型转换
		Type:   core.FavoriteType(*_type),    // 类型转换
		Id:     id,
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

func (r *CoreAdapterImpl) RemoveFavorite(ctx context.Context, id, userId string, target *biz.FavoriteTarget, _type *biz.FavoriteType) error {
	resp, err := r.favorite.RemoveFavorite(ctx, &core.RemoveFavoriteReq{
		Target: core.FavoriteTarget(*target), // 类型转换
		Type:   core.FavoriteType(*_type),    // 类型转换
		Id:     id,
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
