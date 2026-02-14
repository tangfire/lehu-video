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

func (r *CoreAdapterImpl) GetVideoById(ctx context.Context, userId, videoId string) (*biz.Video, error) {
	resp, err := r.video.GetVideoById(ctx, &core.GetVideoByIdReq{
		VideoId: videoId,
		UserId:  userId,
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

	// 修复：使用video.FavoriteCount而不是video.IsFavorite
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
		FavoriteCount: video.FavoriteCount, // 修复这里
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

// Feed deprecated
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
