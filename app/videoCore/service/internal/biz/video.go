package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoCore/service/v1"
	"time"
)

type Author struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	IsFollowing int64  `json:"is_following"`
}

type Video struct {
	Id           int64   `json:"id" gorm:"column:id"`
	Title        string  `json:"title" gorm:"column:title"`
	Description  string  `json:"description" gorm:"column:description"`
	VideoUrl     string  `json:"video_url" gorm:"column:video_url"`
	CoverUrl     string  `json:"cover_url" gorm:"column:cover_url"`
	LikeCount    int64   `json:"like_count" gorm:"column:like_count"`
	CommentCount int64   `json:"comment_count" gorm:"column:comment_count"`
	Author       *Author `json:"author" gorm:"-"`
	UploadTime   time.Time
}

type PageStats struct {
	Page     int32
	PageSize int32
}

type VideoRepo interface {
	PublishVideo(ctx context.Context, video Video) (int64, error)
	GetVideoById(ctx context.Context, id int64) (bool, *Video, error)
	GetVideoListByUid(ctx context.Context, uid int64, latestTime time.Time, pageStats PageStats) ([]*Video, error)
}

type VideoUsecase struct {
	repo VideoRepo
	log  *log.Helper
}

func NewVideoUsecase(repo VideoRepo, logger log.Logger) *VideoUsecase {
	return &VideoUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *VideoUsecase) PublishVideo(ctx context.Context, req *pb.PublishVideoReq) (*pb.PublishVideoResp, error) {
	videoId, err := uc.repo.PublishVideo(ctx, Video{
		Title:       req.Title,
		Description: req.Description,
		VideoUrl:    req.PlayUrl,
		CoverUrl:    req.CoverUrl,
		Author:      &Author{Id: req.UserId},
	})
	if err != nil {
		return nil, err
	}
	return &pb.PublishVideoResp{
		Meta: &pb.Metadata{
			Code:    0,
			Message: "success",
		},
		VideoId: videoId,
	}, nil
}

func (uc *VideoUsecase) GetVideoById(ctx context.Context, req *pb.GetVideoByIdReq) (*pb.GetVideoByIdResp, error) {
	exist, video, err := uc.repo.GetVideoById(ctx, req.VideoId)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return &pb.GetVideoByIdResp{
		Meta: &pb.Metadata{
			Code:    0,
			Message: "success",
		},
		Video: &pb.Video{
			Id:    video.Id,
			Title: video.Title,
			Author: &pb.Author{
				Id:     video.Author.Id,
				Name:   video.Author.Name,
				Avatar: video.Author.Avatar,
			},
			PlayUrl:       video.VideoUrl,
			CoverUrl:      video.CoverUrl,
			FavoriteCount: video.LikeCount,
			CommentCount:  video.CommentCount,
			UploadTime:    video.UploadTime.Format(time.DateTime),
			Description:   video.Description,
		},
	}, nil
}

func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	latestTime := time.Now()
	if req.LatestTime > 0 {
		latestTime = time.Unix(req.LatestTime, 0)
	}
	list, err := uc.repo.GetVideoListByUid(ctx, req.UserId, latestTime, PageStats{
		Page:     req.PageStats.Page,
		PageSize: req.PageStats.Size,
	})
	if err != nil {
		return nil, err
	}
	var videoList []*pb.Video
	for _, video := range list {
		videoList = append(videoList, &pb.Video{
			Id:    video.Id,
			Title: video.Title,
			Author: &pb.Author{
				Id:     video.Author.Id,
				Name:   video.Author.Name,
				Avatar: video.Author.Avatar,
			},
			PlayUrl:       video.VideoUrl,
			CoverUrl:      video.CoverUrl,
			FavoriteCount: video.LikeCount,
			CommentCount:  video.CommentCount,
			UploadTime:    video.UploadTime.Format(time.DateTime),
			Description:   video.Description,
		})
	}
	return &pb.ListPublishedVideoResp{Meta: &pb.Metadata{Code: 0, Message: "success"}, Videos: videoList}, nil
}
