package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
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

type FeedShortVideoInput struct {
	LatestTime int64
	UserId     int64
	FeedNum    int64
}

type FeedShortVideoOutput struct {
	Videos   []*Video
	NextTime int64
}

type ListPublishedVideoInput struct {
	UserId    int64
	PageStats *PageStats
}
type ListPublishedVideoOutput struct {
	VideoList []*Video
	Total     int64
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

func (uc *VideoUsecase) AssembleVideo(ctx context.Context, userId int64, data []*Video) ([]*Video, error) {
	videoList, _ := uc.AssembleVideoList(ctx, userId, data)
	uc.AssembleAuthorInfo(ctx, videoList)
	uc.AssembleUserIsFollowing(ctx, videoList, userId)
	uc.AssembleVideoCountInfo(ctx, videoList)
	return videoList, nil
}

func (uc *VideoUsecase) AssembleAuthorInfo(ctx context.Context, data []*Video) {
	var userIdList []int64
	for _, video := range data {
		userIdList = append(userIdList, video.Author.ID)
	}

	userList, err := uc.core.GetUserInfoByIdList(ctx, userIdList)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("failed to get user info: %v", err)
	}

	userMap := make(map[int64]*UserInfo)
	for _, user := range userList {
		userMap[user.Id] = user
	}

	for _, video := range data {
		if video.Author == nil {
			continue
		}

		author, ok := userMap[video.Author.ID]
		if !ok {
			continue
		}
		video.Author.Name = author.Name
		video.Author.Avatar = author.Avatar
	}

}

func (uc *VideoUsecase) AssembleVideoList(ctx context.Context, userId int64, data []*Video) ([]*Video, error) {
	var result []*Video
	var videoIdList []int64
	for _, video := range data {
		videoIdList = append(videoIdList, video.ID)
	}
	isFavoriteMap, err := uc.core.IsUserFavoriteVideo(ctx, userId, videoIdList)
	if err != nil {
		log.Context(ctx).Warnf("failed to check favorite video: %v", err)
	}

	for _, video := range data {
		isFavorite, ok := isFavoriteMap[video.ID]

		result = append(result, &Video{
			ID:            video.ID,
			Title:         video.Title,
			PlayUrl:       video.PlayUrl,
			CoverUrl:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    isFavorite && ok,
			Author: &VideoAuthor{
				ID:          video.Author.ID,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing,
			},
		})
	}
	return result, nil
}

func (uc *VideoUsecase) AssembleUserIsFollowing(ctx context.Context, list []*Video, userId int64) {
	var targetUserId []int64
	var targetVideoId []int64
	for _, video := range list {
		targetUserId = append(targetUserId, video.Author.ID)
		targetVideoId = append(targetVideoId, video.ID)
	}

	isFollowingMap, err := uc.core.IsFollowing(ctx, userId, targetUserId)
	if err != nil {
		log.Context(ctx).Errorf("failed to check is following: %v", err)
	}

	isCollectedMap, err := uc.core.IsCollected(ctx, userId, targetVideoId)
	if err != nil {
		log.Context(ctx).Errorf("failed to check is collected: %v", err)
	}

	isFavoriteMap, err := uc.core.IsUserFavoriteVideo(ctx, userId, targetVideoId)
	if err != nil {
		log.Context(ctx).Errorf("failed to check is favorite: %v", err)
	}

	for _, video := range list {
		author := video.Author
		author.IsFollowing = isFollowingMap[author.ID]
		video.IsCollected = isCollectedMap[video.ID]
		video.IsFavorite = isFavoriteMap[video.ID]
	}

}

func (uc *VideoUsecase) AssembleVideoCountInfo(ctx context.Context, list []*Video) {
	var videoIdList []int64
	for _, video := range list {
		videoIdList = append(videoIdList, video.ID)
	}

	commentCountMap, err := uc.core.CountComments4Video(ctx, videoIdList)
	if err != nil {
		log.Context(ctx).Errorf("failed to count comments: %v", err)
	}

	favoriteCountMap, err := uc.core.CountFavorite4Video(ctx, videoIdList)
	if err != nil {
		log.Context(ctx).Errorf("failed to count favorite: %v", err)
	}

	collectedCountMap, err := uc.core.CountCollected4Video(ctx, videoIdList)
	if err != nil {
		log.Context(ctx).Errorf("failed to count collected: %v", err)
	}

	for _, video := range list {
		video.CommentCount = commentCountMap[video.ID]
		video.FavoriteCount = favoriteCountMap[video.ID]
		video.CollectedCount = collectedCountMap[video.ID]
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

func (uc *VideoUsecase) FeedShortVideo(ctx context.Context, in *FeedShortVideoInput) (*FeedShortVideoOutput, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, err
	}
	videos, err := uc.core.Feed(ctx, in.UserId, in.FeedNum, in.LatestTime)
	if err != nil {
		return nil, err
	}
	uc.AssembleUserIsFollowing(ctx, videos, userId)
	uc.AssembleVideoCountInfo(ctx, videos)

	return &FeedShortVideoOutput{
		Videos:   videos,
		NextTime: 0,
	}, nil
}

func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, in *ListPublishedVideoInput) (*ListPublishedVideoOutput, error) {
	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, err
	}
	total, videos, err := uc.core.ListPublishedVideo(ctx, userId, in.PageStats)
	if err != nil {
		return nil, err
	}
	videoList, err := uc.AssembleVideoList(ctx, userId, videos)
	if err != nil {
		return nil, err
	}
	uc.AssembleUserIsFollowing(ctx, videoList, userId)
	uc.AssembleVideoCountInfo(ctx, videoList)
	return &ListPublishedVideoOutput{
		VideoList: videoList,
		Total:     total,
	}, nil
}
