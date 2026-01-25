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

type PageStats struct {
	Page     int32
	PageSize int32
}

// ✅ biz层自己的请求/响应结构体
type PublishVideoReq struct {
	UserId      int64
	Title       string
	Description string
	PlayUrl     string
	CoverUrl    string
}

type PublishVideoResp struct {
	VideoId int64
}

type GetVideoByIdReq struct {
	VideoId int64
}

type GetVideoByIdResp struct {
	Video *Video
}

type ListPublishedVideoReq struct {
	UserId     int64
	LatestTime int64 // Unix时间戳
	PageStats  PageStats
}

type ListPublishedVideoResp struct {
	Videos []*Video
}

// 新增的请求/响应
type FeedShortVideoReq struct {
	LatestTime int64 // Unix时间戳
	PageStats  PageStats
}

type FeedShortVideoResp struct {
	Videos []*Video
}

type GetVideoByIdListReq struct {
	VideoIdList []int64
}

type GetVideoByIdListResp struct {
	Videos []*Video
}

type VideoRepo interface {
	PublishVideo(ctx context.Context, video *Video) (int64, error)
	GetVideoById(ctx context.Context, id int64) (bool, *Video, error)
	GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats PageStats) ([]*Video, error)
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

func (uc *VideoUsecase) PublishVideo(ctx context.Context, req *PublishVideoReq) (*PublishVideoResp, error) {
	video := &Video{
		Title:       req.Title,
		Description: req.Description,
		VideoUrl:    req.PlayUrl,
		CoverUrl:    req.CoverUrl,
		Author:      &Author{Id: req.UserId},
		UploadTime:  time.Now(),
	}

	videoId, err := uc.repo.PublishVideo(ctx, video)
	if err != nil {
		return nil, err
	}

	return &PublishVideoResp{
		VideoId: videoId,
	}, nil
}

func (uc *VideoUsecase) GetVideoById(ctx context.Context, req *GetVideoByIdReq) (*GetVideoByIdResp, error) {
	exist, video, err := uc.repo.GetVideoById(ctx, req.VideoId)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, nil // 或者返回特定错误
	}

	return &GetVideoByIdResp{
		Video: video,
	}, nil
}

func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, req *ListPublishedVideoReq) (*ListPublishedVideoResp, error) {
	latestTime := time.Now()
	if req.LatestTime > 0 {
		latestTime = time.Unix(req.LatestTime, 0)
	}

	videos, err := uc.repo.GetVideoListByUid(ctx, req.UserId, latestTime, req.PageStats)
	if err != nil {
		return nil, err
	}

	return &ListPublishedVideoResp{
		Videos: videos,
	}, nil
}

func (uc *VideoUsecase) FeedShortVideo(ctx context.Context, req *FeedShortVideoReq) (*FeedShortVideoResp, error) {
	latestTime := time.Now()
	if req.LatestTime > 0 {
		latestTime = time.Unix(req.LatestTime, 0)
	}

	videos, err := uc.repo.GetFeedVideos(ctx, latestTime, req.PageStats)
	if err != nil {
		return nil, err
	}

	return &FeedShortVideoResp{
		Videos: videos,
	}, nil
}

func (uc *VideoUsecase) GetVideoByIdList(ctx context.Context, req *GetVideoByIdListReq) (*GetVideoByIdListResp, error) {
	videos, err := uc.repo.GetVideoByIdList(ctx, req.VideoIdList)
	if err != nil {
		return nil, err
	}

	return &GetVideoByIdListResp{
		Videos: videos,
	}, nil
}
