package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	pb "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

type UploadVideoInput struct {
	Hash     string
	FileType string
	Size     int64
	Filename string
}

type UploadVideoOutput struct {
	Url    string
	FileId int64
}

type UploadCoverInput struct {
	Hash     string
	FileType string
	Size     int64
	Filename string
}

type UploadCoverOutput struct {
	Url    string
	FileId int64
}

type ReportFinishUploadInput struct {
	FileId int64
}

type ReportFinishUploadOutput struct {
	Url string
}

type ReportVideoFinishUploadInput struct {
	FileId      int64
	Title       string
	CoverUrl    string
	Description string
	VideoUrl    string
}

type ReportVideoFinishUploadOutput struct {
	VideoId int64
}

type GetVideoByIdInput struct {
	VideoId int64
}

type VideoAuthor struct {
	ID          int64
	Name        string
	Avatar      string
	IsFollowing bool
}

type Video struct {
	ID             int64
	Author         *VideoAuthor
	PlayUrl        string
	CoverUrl       string
	FavoriteCount  int64
	CommentCount   int64
	IsFavorite     bool
	Title          string
	IsCollected    bool
	CollectedCount int64
}

type VideoRepo interface {
}

type VideoUsecase struct {
	base BaseAdapter
	core CoreAdapter
	repo VideoRepo
	log  *log.Helper
}

func NewVideoUsecase(base BaseAdapter, core CoreAdapter, repo VideoRepo, logger log.Logger) *VideoUsecase {
	return &VideoUsecase{
		base: base,
		core: core,
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *VideoUsecase) UploadVideo(ctx context.Context, in *UploadVideoInput) (out *UploadVideoOutput, err error) {
	fileId, url, err := uc.base.PreSign4Upload(ctx, in.Hash, in.FileType, in.Filename, in.Size, 3600)
	if err != nil {
		return
	}
	return &UploadVideoOutput{
		Url:    url,
		FileId: fileId,
	}, nil
}

func (uc *VideoUsecase) UploadCover(ctx context.Context, in *UploadCoverInput) (out *UploadCoverOutput, err error) {
	fileId, url, err := uc.base.PreSign4Upload(ctx, in.Hash, in.FileType, in.Filename, in.Size, 3600)
	if err != nil {
		return
	}
	return &UploadCoverOutput{
		Url:    url,
		FileId: fileId,
	}, nil
}

func (uc *VideoUsecase) ReportFinishUpload(ctx context.Context, in *ReportFinishUploadInput) (out *ReportFinishUploadOutput, err error) {
	url, err := uc.base.ReportUploaded(ctx, in.FileId)
	if err != nil {
		return nil, err
	}
	return &ReportFinishUploadOutput{
		Url: url,
	}, nil
}

func (uc *VideoUsecase) ReportVideoFinishUpload(ctx context.Context, in *ReportVideoFinishUploadInput) (out *ReportVideoFinishUploadOutput, err error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, err
	}
	// todo 跟DouTok不一样，待定
	_, err = uc.base.ReportUploaded(ctx, in.FileId)
	if err != nil {
		return nil, err
	}

	videoId, err := uc.core.SaveVideoInfo(ctx, in.Title, in.VideoUrl, in.CoverUrl, in.Description, userId)
	if err != nil {
		return nil, err
	}
	return &ReportVideoFinishUploadOutput{
		VideoId: videoId,
	}, nil
}

func (uc *VideoUsecase) GetVideoById(ctx context.Context, in *GetVideoByIdInput) (out *Video, err error) {
	video, err := uc.core.GetVideoById(ctx, in.VideoId)
	if err != nil {
		return nil, err
	}
	return video, nil
}

func (uc *VideoUsecase) FeedShortVideo(ctx context.Context, req *pb.FeedShortVideoReq) (*pb.FeedShortVideoResp, error) {
	return &pb.FeedShortVideoResp{}, nil
}

func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, req *pb.ListPublishedVideoReq) (*pb.ListPublishedVideoResp, error) {
	return &pb.ListPublishedVideoResp{}, nil
}
