package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type Author struct {
	Id          int64
	Name        string
	Avatar      string
	IsFollowing int64
}

type Video struct {
	Id           int64
	Title        string
	Description  string
	VideoUrl     string
	CoverUrl     string
	LikeCount    int64
	CommentCount int64
	Author       *Author
	UploadTime   time.Time
}

// ✅ 使用Command/Query/Result模式
type PublishVideoCommand struct {
	UserId      int64
	Title       string
	Description string
	PlayUrl     string
	CoverUrl    string
}

type PublishVideoResult struct {
	VideoId int64
}

type GetVideoByIdQuery struct {
	VideoId int64
}

type GetVideoByIdResult struct {
	Video *Video
}

type ListPublishedVideoQuery struct {
	UserId     int64
	LatestTime int64 // Unix时间戳
	PageStats  PageStats
}

type ListPublishedVideoResult struct {
	Videos []*Video
	Total  int64
}

// Feed流查询
type FeedShortVideoQuery struct {
	LatestTime int64 // Unix时间戳
	PageStats  PageStats
}

type FeedShortVideoResult struct {
	Videos []*Video
}

type GetVideoByIdListQuery struct {
	VideoIdList []int64
}

type GetVideoByIdListResult struct {
	Videos []*Video
}

type VideoRepo interface {
	PublishVideo(ctx context.Context, video *Video) (int64, error)
	GetVideoById(ctx context.Context, id int64) (bool, *Video, error)
	GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats PageStats) (int64, []*Video, error)
	GetVideoByIdList(ctx context.Context, idList []int64) ([]*Video, error)
	GetFeedVideos(ctx context.Context, latestTime time.Time, pageStats PageStats) ([]*Video, error)
}

type VideoUsecase struct {
	repo VideoRepo
	log  *log.Helper
}

func NewVideoUsecase(repo VideoRepo, logger log.Logger) *VideoUsecase {
	return &VideoUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *VideoUsecase) PublishVideo(ctx context.Context, cmd *PublishVideoCommand) (*PublishVideoResult, error) {
	video := &Video{
		Title:       cmd.Title,
		Description: cmd.Description,
		VideoUrl:    cmd.PlayUrl,
		CoverUrl:    cmd.CoverUrl,
		Author:      &Author{Id: cmd.UserId},
		UploadTime:  time.Now(),
	}

	videoId, err := uc.repo.PublishVideo(ctx, video)
	if err != nil {
		return nil, err
	}

	return &PublishVideoResult{
		VideoId: videoId,
	}, nil
}

func (uc *VideoUsecase) GetVideoById(ctx context.Context, query *GetVideoByIdQuery) (*GetVideoByIdResult, error) {
	exist, video, err := uc.repo.GetVideoById(ctx, query.VideoId)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, nil // 或者返回特定错误
	}

	return &GetVideoByIdResult{
		Video: video,
	}, nil
}

func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, query *ListPublishedVideoQuery) (*ListPublishedVideoResult, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}

	total, videos, err := uc.repo.GetVideoListByUid(ctx, query.UserId, latestTime, query.PageStats)
	if err != nil {
		return nil, err
	}

	return &ListPublishedVideoResult{
		Videos: videos,
		Total:  total,
	}, nil
}

func (uc *VideoUsecase) FeedShortVideo(ctx context.Context, query *FeedShortVideoQuery) (*FeedShortVideoResult, error) {
	latestTime := time.Now()
	if query.LatestTime > 0 {
		latestTime = time.Unix(query.LatestTime, 0)
	}

	videos, err := uc.repo.GetFeedVideos(ctx, latestTime, query.PageStats)
	if err != nil {
		return nil, err
	}

	return &FeedShortVideoResult{
		Videos: videos,
	}, nil
}

func (uc *VideoUsecase) GetVideoByIdList(ctx context.Context, query *GetVideoByIdListQuery) (*GetVideoByIdListResult, error) {
	videos, err := uc.repo.GetVideoByIdList(ctx, query.VideoIdList)
	if err != nil {
		return nil, err
	}

	return &GetVideoByIdListResult{
		Videos: videos,
	}, nil
}
