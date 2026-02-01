package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (r *CoreAdapterImpl) SaveVideoInfo(ctx context.Context, title, videoUrl, coverUrl, desc, userId string) (string, error) {
	resp, err := r.video.PublishVideo(ctx, &core.PublishVideoReq{
		Title:       title,
		CoverUrl:    coverUrl,
		PlayUrl:     videoUrl,
		Description: desc,
		UserId:      userId,
	})
	if err != nil {
		return "0", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", err
	}
	return resp.VideoId, nil
}

func (r *CoreAdapterImpl) GetVideoById(ctx context.Context, videoId string) (*biz.Video, error) {
	resp, err := r.video.GetVideoById(ctx, &core.GetVideoByIdReq{
		VideoId: videoId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	video := resp.Video
	author := video.Author
	retVideo := &biz.Video{
		ID: video.Id,
		Author: &biz.VideoAuthor{
			ID:          author.Id,
			Name:        author.Name,
			Avatar:      author.Avatar,
			IsFollowing: author.IsFollowing != 0,
		},
		PlayURL:       video.PlayUrl,
		CoverURL:      video.CoverUrl,
		FavoriteCount: video.FavoriteCount,
		CommentCount:  video.CommentCount,
		IsFavorite:    video.IsFavorite != 0,
		Title:         video.Title,
	}
	return retVideo, nil
}

func (r *CoreAdapterImpl) ListPublishedVideo(ctx context.Context, userId string, pageStats *biz.PageStats) (int64, []*biz.Video, error) {
	resp, err := r.video.ListPublishedVideo(ctx, &core.ListPublishedVideoReq{
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
	var retVideos []*biz.Video
	for _, video := range resp.Videos {
		retVideos = append(retVideos, &biz.Video{
			ID: video.Id,
			Author: &biz.VideoAuthor{
				ID:          video.Author.Id,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing != 0,
			},
			PlayURL:       video.PlayUrl,
			CoverURL:      video.CoverUrl,
			FavoriteCount: video.IsFavorite,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite != 0,
			Title:         video.Title,
		})
	}
	return int64(resp.PageStats.Total), retVideos, nil
}

func (r *CoreAdapterImpl) IsUserFavoriteVideo(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error) {

	var items []*core.IsFavoriteReqItem
	for _, id := range videoIdList {
		items = append(items, &core.IsFavoriteReqItem{
			BizId:  id,
			UserId: userId,
		})
	}

	resp, err := r.favorite.IsFavorite(ctx, &core.IsFavoriteReq{
		Items: items,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool)
	if len(resp.Items) == 0 {
		return result, nil
	}

	for _, item := range resp.Items {
		result[item.BizId] = item.IsFavorite
	}
	return result, nil
}

func (r *CoreAdapterImpl) Feed(ctx context.Context, userId string, num int64, latestTime int64) ([]*biz.Video, error) {
	req := &core.FeedShortVideoReq{
		LatestTime: latestTime,
		UserId:     userId,
		FeedNum:    num,
	}

	resp, err := r.video.FeedShortVideo(ctx, req)
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	var videos []*biz.Video
	for _, video := range resp.Videos {
		videos = append(videos, &biz.Video{
			ID: video.Id,
			Author: &biz.VideoAuthor{
				ID:          video.Author.Id,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing != 0,
			},
			PlayURL:       video.PlayUrl,
			CoverURL:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite != 0,
			Title:         video.Title,
		})
	}
	return videos, nil
}

func (r *CoreAdapterImpl) ListUserFavoriteVideo(ctx context.Context, userId string, pageStats *biz.PageStats) (int64, []string, error) {
	resp, err := r.favorite.ListFavorite(ctx, &core.ListFavoriteReq{
		Id:            userId,
		AggregateType: core.FavoriteAggregateType_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE,
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
	return int64(resp.PageStats.Total), resp.Ids, nil
}

func (r *CoreAdapterImpl) GetVideoByIdList(ctx context.Context, videoIdList []string) ([]*biz.Video, error) {
	resp, err := r.video.GetVideoByIdList(ctx, &core.GetVideoByIdListReq{
		VideoIdList: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	videos := resp.Videos
	var retVideos []*biz.Video
	for _, video := range videos {
		retVideos = append(retVideos, &biz.Video{
			ID: video.Id,
			Author: &biz.VideoAuthor{
				ID:          video.Author.Id,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing != 0,
			},
			PlayURL:       video.PlayUrl,
			CoverURL:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite != 0,
			Title:         video.Title,
		})
	}
	return retVideos, nil
}
