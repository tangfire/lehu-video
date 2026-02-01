package data

import (
	"context"
	"errors"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *CoreAdapterImpl) CountBeFavoriteNumber4User(ctx context.Context, userId string) (int64, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		Ids:           []string{userId},
		AggregateType: core.FavoriteAggregateType_FAVORITE_AGGREGATE_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE_TYPE_LIKE,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.Items[0].LikeCount, nil
}

func (r *CoreAdapterImpl) AddFavorite(ctx context.Context, id, userId string, target *biz.FavoriteTarget, _type *biz.FavoriteType) (*biz.AddFavoriteResult, error) {
	if target == nil || _type == nil {
		return nil, errors.New("参数不完整")
	}

	resp, err := r.favorite.AddFavorite(ctx, &core.AddFavoriteReq{
		Target: core.FavoriteTarget(*target),
		Type:   core.FavoriteType(*_type),
		BizId:  id,
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 获取最新的统计数据
	stats, err := r.GetFavoriteStats(ctx, id, target)
	if err != nil {
		// 即使获取统计失败，也返回添加成功
		r.log.Warnf("获取点赞统计失败: targetId=%s, err=%v", id, err)
		return &biz.AddFavoriteResult{
			AlreadyFavorited: resp.AlreadyFavorited,
			TotalCount:       0,
			TotalLikes:       0,
			TotalDislikes:    0,
			PreviousType:     -1,
		}, nil
	}

	return &biz.AddFavoriteResult{
		AlreadyFavorited: resp.AlreadyFavorited,
		TotalCount:       stats.TotalCount,
		TotalLikes:       stats.LikeCount,
		TotalDislikes:    stats.DislikeCount,
		PreviousType:     -1, // 新版本core服务没有返回这个字段
	}, nil
}

func (r *CoreAdapterImpl) RemoveFavorite(ctx context.Context, id, userId string, target *biz.FavoriteTarget, _type *biz.FavoriteType) (*biz.RemoveFavoriteResult, error) {
	if target == nil || _type == nil {
		return nil, errors.New("参数不完整")
	}

	resp, err := r.favorite.RemoveFavorite(ctx, &core.RemoveFavoriteReq{
		Target: core.FavoriteTarget(*target),
		Type:   core.FavoriteType(*_type),
		BizId:  id,
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 获取最新的统计数据
	stats, err := r.GetFavoriteStats(ctx, id, target)
	if err != nil {
		// 即使获取统计失败，也返回取消成功
		r.log.Warnf("获取点赞统计失败: targetId=%s, err=%v", id, err)
		return &biz.RemoveFavoriteResult{
			NotFavorited:  resp.NotFavorited,
			TotalCount:    0,
			TotalLikes:    0,
			TotalDislikes: 0,
		}, nil
	}

	return &biz.RemoveFavoriteResult{
		NotFavorited:  resp.NotFavorited,
		TotalCount:    stats.TotalCount,
		TotalLikes:    stats.LikeCount,
		TotalDislikes: stats.DislikeCount,
	}, nil
}

func (r *CoreAdapterImpl) ListUserFavoriteVideo(ctx context.Context, userId string, pageStats *biz.PageStats) (int64, []string, error) {
	if pageStats == nil {
		pageStats = &biz.PageStats{Page: 1, PageSize: 20}
	}

	resp, err := r.favorite.ListFavorite(ctx, &core.ListFavoriteReq{
		Id:            userId,
		AggregateType: core.FavoriteAggregateType_FAVORITE_AGGREGATE_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE_TYPE_LIKE,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return 0, nil, err
	}

	var total int64
	if resp.PageStats != nil {
		total = int64(resp.PageStats.Total)
	} else {
		// 如果没有分页信息，估算总数
		total = int64(len(resp.Ids))
	}

	return total, resp.Ids, nil
}

func (r *CoreAdapterImpl) CheckFavoriteStatus(ctx context.Context, userId, targetId string, target *biz.FavoriteTarget, _type *biz.FavoriteType) (*biz.CheckFavoriteResult, error) {
	// 如果userId为0，表示未登录用户，只返回统计信息
	if userId == "0" {
		stats, err := r.GetFavoriteStats(ctx, targetId, target)
		if err != nil {
			return nil, err
		}

		return &biz.CheckFavoriteResult{
			IsFavorite:    false,
			FavoriteType:  -1,
			TotalLikes:    stats.LikeCount,
			TotalDislikes: stats.DislikeCount,
			TotalCount:    stats.TotalCount,
		}, nil
	}

	resp, err := r.favorite.IsFavorite(ctx, &core.IsFavoriteReq{
		BizId:  targetId,
		UserId: userId,
		Target: core.FavoriteTarget(*target),
		Type:   core.FavoriteType(*_type),
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 获取统计数据
	stats, err := r.GetFavoriteStats(ctx, targetId, target)
	if err != nil {
		return nil, err
	}

	return &biz.CheckFavoriteResult{
		IsFavorite:    resp.IsFavorite,
		FavoriteType:  int32(*_type),
		TotalLikes:    stats.LikeCount,
		TotalDislikes: stats.DislikeCount,
		TotalCount:    stats.TotalCount,
	}, nil
}

func (r *CoreAdapterImpl) GetFavoriteStats(ctx context.Context, targetId string, target *biz.FavoriteTarget) (*biz.FavoriteStats, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		AggregateType: core.FavoriteAggregateType_FAVORITE_AGGREGATE_BY_VIDEO,
		Ids:           []string{targetId},
		FavoriteType:  core.FavoriteType_FAVORITE_TYPE_LIKE, // 这里只查询点赞，需要分别查询点赞和点踩
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 查询点踩数量
	dislikeResp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		AggregateType: core.FavoriteAggregateType_FAVORITE_AGGREGATE_BY_VIDEO,
		Ids:           []string{targetId},
		FavoriteType:  core.FavoriteType_FAVORITE_TYPE_DISLIKE,
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(dislikeResp.Meta); err != nil {
		return nil, err
	}

	// 解析结果
	var likeCount, dislikeCount int64
	if len(resp.Items) > 0 {
		likeCount = resp.Items[0].LikeCount
	}
	if len(dislikeResp.Items) > 0 {
		dislikeCount = dislikeResp.Items[0].DislikeCount
	}

	totalCount := likeCount + dislikeCount
	hotScore := float64(likeCount) - float64(dislikeCount)*0.5

	return &biz.FavoriteStats{
		LikeCount:    likeCount,
		DislikeCount: dislikeCount,
		TotalCount:   totalCount,
		HotScore:     hotScore,
	}, nil
}

func (r *CoreAdapterImpl) CountFavorite4Video(ctx context.Context, videoIdList []string) (map[string]biz.FavoriteCount, error) {
	if len(videoIdList) == 0 {
		return map[string]biz.FavoriteCount{}, nil
	}

	// 查询点赞数量
	likeResp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		AggregateType: core.FavoriteAggregateType_FAVORITE_AGGREGATE_BY_VIDEO,
		Ids:           videoIdList,
		FavoriteType:  core.FavoriteType_FAVORITE_TYPE_LIKE,
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(likeResp.Meta); err != nil {
		return nil, err
	}

	// 查询点踩数量
	dislikeResp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		AggregateType: core.FavoriteAggregateType_FAVORITE_AGGREGATE_BY_VIDEO,
		Ids:           videoIdList,
		FavoriteType:  core.FavoriteType_FAVORITE_TYPE_DISLIKE,
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(dislikeResp.Meta); err != nil {
		return nil, err
	}

	// 构建结果
	result := make(map[string]biz.FavoriteCount)

	// 初始化所有视频的计数
	for _, videoId := range videoIdList {
		result[videoId] = biz.FavoriteCount{
			LikeCount:    0,
			DislikeCount: 0,
			TotalCount:   0,
		}
	}

	// 填充点赞数量
	for _, item := range likeResp.Items {
		counts := result[item.BizId]
		counts.LikeCount = item.LikeCount
		counts.TotalCount = counts.LikeCount + counts.DislikeCount
		result[item.BizId] = counts
	}

	// 填充点踩数量
	for _, item := range dislikeResp.Items {
		counts := result[item.BizId]
		counts.DislikeCount = item.DislikeCount
		counts.TotalCount = counts.LikeCount + counts.DislikeCount
		result[item.BizId] = counts
	}

	return result, nil
}

func (r *CoreAdapterImpl) IsUserFavoriteVideo(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error) {
	if userId == "" || len(videoIdList) == 0 {
		return map[string]bool{}, nil
	}

	resp, err := r.favorite.BatchIsFavorite(ctx, &core.BatchIsFavoriteReq{
		BizIds: videoIdList,
		UserId: userId,
		Target: core.FavoriteTarget_FAVORITE_TARGET_VIDEO,
	})
	if err != nil {
		return nil, err
	}

	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, err
	}

	// 构建结果映射
	result := make(map[string]bool)
	for _, videoId := range videoIdList {
		result[videoId] = false
	}

	for _, item := range resp.Items {
		result[item.BizId] = item.IsLiked
	}

	return result, nil
}
