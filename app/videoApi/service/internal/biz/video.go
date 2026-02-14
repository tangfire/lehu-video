package biz

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cast"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
)

// Video 视频
type Video struct {
	ID             string
	Author         *VideoAuthor
	PlayURL        string
	CoverURL       string
	FavoriteCount  int64
	CommentCount   int64
	IsFavorite     bool
	Title          string
	IsCollected    bool
	CollectedCount int64
}

// VideoAuthor 视频作者
type VideoAuthor struct {
	ID          string
	Name        string
	Avatar      string
	IsFollowing bool
}

// ============ Input/Output 结构体 ============

// PageInput 分页输入
type PageInput struct {
	Page     int64
	PageSize int64
}

// PreSignUploadInput 预签名上传输入
type PreSignUploadInput struct {
	Hash     string
	FileType string
	Size     int64
	Filename string
}

// PreSignUploadOutput 预签名上传输出
type PreSignUploadOutput struct {
	URL    string
	FileID string
}

// ReportFinishUploadInput 报告上传完成输入
type ReportFinishUploadInput struct {
	FileID string
}

// ReportFinishUploadOutput 报告上传完成输出
type ReportFinishUploadOutput struct {
	URL string
}

// ReportVideoFinishUploadInput 报告视频上传完成输入
type ReportVideoFinishUploadInput struct {
	FileID      string
	Title       string
	CoverURL    string
	Description string
	VideoURL    string
	UserID      string
}

// ReportVideoFinishUploadOutput 报告视频上传完成输出
type ReportVideoFinishUploadOutput struct {
	VideoID string
}

// GetVideoInput 获取视频输入
type GetVideoInput struct {
	VideoID string
	UserID  string // 当前用户ID，用于判断是否点赞、关注等
}

// GetVideoOutput 获取视频输出
type GetVideoOutput struct {
	Video *Video
}

// FeedVideoInput 视频流输入
type FeedVideoInput struct {
	LatestTime int64
	UserID     string
	FeedNum    int64
}

// FeedVideoOutput 视频流输出
type FeedVideoOutput struct {
	Videos   []*Video
	NextTime int64
}

// ListPublishedVideoInput 获取已发布视频列表输入
type ListPublishedVideoInput struct {
	UserID   string // 要查询的用户ID
	Page     int64
	PageSize int64
}

// ListPublishedVideoOutput 获取已发布视频列表输出
type ListPublishedVideoOutput struct {
	Videos []*Video
	Total  int64
}

// ============ VideoUsecase 视频用例 ============

// VideoUsecase 视频业务用例
type VideoUsecase struct {
	base      BaseAdapter
	core      CoreAdapter
	assembler *VideoAssembler
	log       *log.Helper
}

// NewVideoUsecase 创建视频用例
func NewVideoUsecase(
	base BaseAdapter,
	core CoreAdapter,
	assembler *VideoAssembler,
	logger log.Logger,
) *VideoUsecase {
	return &VideoUsecase{
		base:      base,
		core:      core,
		assembler: assembler,
		log:       log.NewHelper(logger),
	}
}

// PreSignUpload 预签名上传
func (uc *VideoUsecase) PreSignUpload(ctx context.Context, input *PreSignUploadInput) (*PreSignUploadOutput, error) {
	// 参数验证
	if input.Hash == "" || input.Filename == "" || input.Size <= 0 {
		return nil, ErrInvalidParams
	}

	// 调用存储服务获取预签名URL
	fileID, url, err := uc.base.PreSign4Upload(ctx, input.Hash, input.FileType, input.Filename, input.Size, 3600)
	if err != nil {
		return nil, err
	}

	return &PreSignUploadOutput{
		URL:    url,
		FileID: fileID,
	}, nil
}

// ReportFinishUpload 报告上传完成
func (uc *VideoUsecase) ReportFinishUpload(ctx context.Context, input *ReportFinishUploadInput) (*ReportFinishUploadOutput, error) {
	if input.FileID == "" || cast.ToInt64(input.FileID) <= 0 {
		return nil, ErrInvalidParams
	}

	url, err := uc.base.ReportUploaded(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	return &ReportFinishUploadOutput{
		URL: url,
	}, nil
}

// ReportVideoFinishUpload 报告视频上传完成
func (uc *VideoUsecase) ReportVideoFinishUpload(ctx context.Context, input *ReportVideoFinishUploadInput) (*ReportVideoFinishUploadOutput, error) {
	// 参数验证
	if input.FileID == "" || cast.ToInt64(input.FileID) <= 0 || input.Title == "" || input.VideoURL == "" || input.CoverURL == "" {
		return nil, ErrInvalidParams
	}
	if input.UserID == "" || cast.ToInt64(input.UserID) <= 0 {
		return nil, ErrUnauthorized
	}

	// 1. 报告文件上传完成
	_, err := uc.base.ReportUploaded(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	// 2. 创建视频记录
	videoID, err := uc.core.SaveVideoInfo(ctx, input.Title, input.VideoURL, input.CoverURL, input.Description, input.UserID)
	if err != nil {
		return nil, err
	}

	return &ReportVideoFinishUploadOutput{
		VideoID: videoID,
	}, nil
}

// GetVideo 获取视频
func (uc *VideoUsecase) GetVideo(ctx context.Context, input *GetVideoInput) (*GetVideoOutput, error) {
	if input.VideoID == "" || cast.ToInt64(input.VideoID) <= 0 {
		return nil, ErrInvalidParams
	}

	userId, err := claims.GetUserId(ctx)
	if err != nil {
		return nil, ErrUnauthorized
	}
	// 获取视频基本信息
	video, err := uc.core.GetVideoById(ctx, userId, input.VideoID)
	if err != nil {
		return nil, err
	}
	if video == nil {
		return nil, ErrVideoNotFound
	}

	// 组装完整的视频信息
	video, err = uc.assembler.AssembleVideo(ctx, video, input.UserID)
	if err != nil {
		return nil, err
	}

	return &GetVideoOutput{
		Video: video,
	}, nil
}

// FeedVideo 视频流
// api/videoApi/service/internal/biz/video.go
func (uc *VideoUsecase) FeedVideo(ctx context.Context, input *FeedVideoInput) (*FeedVideoOutput, error) {
	// 1. 调用 core 获取 FeedItem 列表
	feedItems, nextTime, err := uc.core.GetFeed(ctx, input.UserID, input.LatestTime, int32(input.FeedNum), 1 /* 默认推荐流，可按需调整 */)
	if err != nil {
		return nil, err
	}
	if len(feedItems) == 0 {
		return &FeedVideoOutput{Videos: []*Video{}, NextTime: nextTime}, nil
	}

	// 2. 提取 videoID 列表
	videoIDs := make([]string, 0, len(feedItems))
	for _, item := range feedItems {
		videoIDs = append(videoIDs, item.VideoID)
	}

	// 3. 批量获取视频详情（基础信息）
	videos, err := uc.core.GetVideoByIdList(ctx, videoIDs)
	if err != nil {
		return nil, err
	}

	// 4. 组装完整视频信息（互动状态、计数等）
	videos, err = uc.assembler.AssembleVideos(ctx, videos, input.UserID)
	if err != nil {
		return nil, err
	}

	// 5. 按 FeedItem 的顺序重新排序（因为 GetVideoByIdList 返回顺序可能与输入不一致）
	videoMap := make(map[string]*Video, len(videos))
	for _, v := range videos {
		videoMap[v.ID] = v
	}
	orderedVideos := make([]*Video, 0, len(feedItems))
	for _, item := range feedItems {
		if v, ok := videoMap[item.VideoID]; ok {
			orderedVideos = append(orderedVideos, v)
		}
	}

	return &FeedVideoOutput{
		Videos:   orderedVideos,
		NextTime: nextTime,
	}, nil
}

// ListPublishedVideo 获取已发布视频列表
func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, input *ListPublishedVideoInput) (*ListPublishedVideoOutput, error) {
	if cast.ToInt64(input.UserID) <= 0 {
		return nil, ErrInvalidParams
	}

	// 设置默认分页
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}

	// 获取用户发布的视频
	total, videos, err := uc.core.ListPublishedVideo(ctx, input.UserID, &PageStats{
		Page:     int(input.Page),
		PageSize: int(input.PageSize),
	})
	if err != nil {
		return nil, err
	}

	// 组装完整的视频信息
	videos, err = uc.assembler.AssembleVideos(ctx, videos, input.UserID)
	if err != nil {
		return nil, err
	}

	return &ListPublishedVideoOutput{
		Videos: videos,
		Total:  total,
	}, nil
}

// ============ 辅助结构体 ============

// VideoCountInfo 视频计数信息
type VideoCountInfo struct {
	CommentCounts  map[string]int64
	FavoriteCounts map[string]int64
	CollectCounts  map[string]int64
}

// UserInteractionInfo 用户互动信息
type UserInteractionInfo struct {
	IsFavoriteMap  map[string]bool
	IsCollectMap   map[string]bool
	IsFollowingMap map[string]bool
}

// ============ 错误定义 ============

var (
	ErrInvalidParams = errors.New("invalid parameters")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrVideoNotFound = errors.New("video not found")
)
