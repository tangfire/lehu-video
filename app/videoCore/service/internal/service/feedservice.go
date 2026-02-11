package service

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/pkg/utils"

	pb "lehu-video/api/videoCore/service/v1"
)

type FeedServiceService struct {
	pb.UnimplementedFeedServiceServer
	uc *biz.FeedUsecase
}

func NewFeedServiceService(uc *biz.FeedUsecase) *FeedServiceService {
	return &FeedServiceService{uc: uc}
}
func (s *FeedServiceService) GetFeed(ctx context.Context, req *pb.GetFeedReq) (*pb.GetFeedResp, error) {
	// 构建查询参数
	query := &biz.FeedQuery{
		UserID:     req.UserId,
		LatestTime: req.LatestTime,
		PageSize:   req.PageSize,
		FeedType:   int32(req.FeedType),
	}

	// 调用业务层
	result, err := s.uc.GetFeed(ctx, query)
	if err != nil {
		return &pb.GetFeedResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	// 转换为proto格式
	pbItems := make([]*pb.FeedItem, 0, len(result.Items))
	for _, item := range result.Items {
		pbItems = append(pbItems, &pb.FeedItem{
			VideoId:   item.VideoID,
			Score:     item.Score,
			Timestamp: item.Timestamp,
			AuthorId:  item.AuthorID,
		})
	}

	return &pb.GetFeedResp{
		Meta:     utils.GetSuccessMeta(),
		Items:    pbItems,
		NextTime: result.NextTime,
	}, nil
}
func (s *FeedServiceService) GetHotVideos(ctx context.Context, req *pb.GetHotVideosReq) (*pb.GetHotVideosResp, error) {
	videoIDs, err := s.uc.GetHotVideos(ctx, int(req.Limit))
	if err != nil {
		return &pb.GetHotVideosResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.GetHotVideosResp{
		Meta:     utils.GetSuccessMeta(),
		VideoIds: videoIDs,
	}, nil
}
func (s *FeedServiceService) PushToUserTimeline(ctx context.Context, req *pb.PushToUserTimelineReq) (*pb.PushToUserTimelineResp, error) {
	// 转换为业务层结构
	items := make([]*biz.TimelineItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &biz.TimelineItem{
			VideoID:   item.VideoId,
			AuthorID:  item.AuthorId,
			Timestamp: item.Timestamp,
		})
	}

	err := s.uc.PushToUserTimeline(ctx, req.UserId, items)
	if err != nil {
		return &pb.PushToUserTimelineResp{
			Meta: utils.GetMetaWithError(err),
		}, nil
	}

	return &pb.PushToUserTimelineResp{
		Meta: utils.GetSuccessMeta(),
	}, nil
}

// VideoPublished - 视频发布事件处理（内部RPC）
func (s *FeedServiceService) VideoPublished(ctx context.Context, videoID, authorID string) error {
	return s.uc.VideoPublishedHandler(ctx, videoID, authorID)
}
