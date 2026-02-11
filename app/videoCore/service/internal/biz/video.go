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

// VideoStats 视频统计信息
type VideoStats struct {
	VideoID      string
	LikeCount    int64
	CommentCount int64
	ShareCount   int64
	ViewCount    int64
	HotScore     float64
}

type VideoRepo interface {
	PublishVideo(ctx context.Context, video *Video) (int64, error)
	GetVideoById(ctx context.Context, id int64) (bool, *Video, error)
	GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats PageStats) (int64, []*Video, error)
	GetVideoByIdList(ctx context.Context, idList []int64) ([]*Video, error)
	GetFeedVideos(ctx context.Context, latestTime time.Time, pageStats PageStats) ([]*Video, error)

	GetHotVideos(ctx context.Context, limit int) ([]*Video, error)
	GetVideosByAuthors(ctx context.Context, authorIDs []string, latestTime int64, limit int) ([]*Video, error)
	GetAuthorInfo(ctx context.Context, authorID string) (*Author, error)
	GetVideoListByTime(ctx context.Context, latestTime time.Time, limit int) ([]*Video, error)
	GetVideoStats(ctx context.Context, videoID string) (*VideoStats, error)
}

type VideoUsecase struct {
	repo        VideoRepo
	counterRepo CounterRepo // 新增
	log         *log.Helper
}

func NewVideoUsecase(repo VideoRepo, counterRepo CounterRepo, logger log.Logger) *VideoUsecase {
	return &VideoUsecase{
		repo:        repo,
		counterRepo: counterRepo,
		log:         log.NewHelper(logger),
	}
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

	// ---------- 新增：增加用户 work_count ----------
	if _, err := uc.counterRepo.IncrUserCounter(ctx, cmd.UserId, "work_count", 1); err != nil {
		uc.log.Warnf("增加用户 work_count 失败: userId=%d, err=%v", cmd.UserId, err)
	}
	// ------------------------------------------------

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
