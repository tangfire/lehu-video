package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

func (a *CoreAdapterImpl) GetFeed(ctx context.Context, userID string, latestTime int64, pageSize int32, feedType int32) ([]*biz.FeedItem, int64, error) {
	req := &core.GetFeedReq{
		UserId:     userID,
		LatestTime: latestTime,
		PageSize:   pageSize,
		FeedType:   core.FeedType(feedType),
	}
	resp, err := a.feed.GetFeed(ctx, req) // 注意需要注入 feed gRPC 客户端
	if err != nil {
		return nil, 0, err
	}
	if err := respcheck.ValidateResponseMeta(resp.Meta); err != nil {
		return nil, 0, err
	}

	items := make([]*biz.FeedItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &biz.FeedItem{
			VideoID:   item.VideoId,
			AuthorID:  item.AuthorId,
			Timestamp: item.Timestamp,
			Score:     item.Score,
		})
	}
	return items, resp.NextTime, nil
}
