package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

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
	FileID int64
}

// ReportFinishUploadInput 报告上传完成输入
type ReportFinishUploadInput struct {
	FileID int64
}

// ReportFinishUploadOutput 报告上传完成输出
type ReportFinishUploadOutput struct {
	URL string
}

// ReportVideoFinishUploadInput 报告视频上传完成输入
type ReportVideoFinishUploadInput struct {
	FileID      int64
	Title       string
	CoverURL    string
	Description string
	VideoURL    string
	UserID      int64
}

// ReportVideoFinishUploadOutput 报告视频上传完成输出
type ReportVideoFinishUploadOutput struct {
	VideoID int64
}

// GetVideoInput 获取视频输入
type GetVideoInput struct {
	VideoID int64
	UserID  int64 // 当前用户ID，用于判断是否点赞、关注等
}

// GetVideoOutput 获取视频输出
type GetVideoOutput struct {
	Video *Video
}

// FeedVideoInput 视频流输入
type FeedVideoInput struct {
	LatestTime int64
	UserID     int64
	FeedNum    int64
}

// FeedVideoOutput 视频流输出
type FeedVideoOutput struct {
	Videos   []*Video
	NextTime int64
}

// ListPublishedVideoInput 获取已发布视频列表输入
type ListPublishedVideoInput struct {
	UserID   int64 // 要查询的用户ID
	Page     int64
	PageSize int64
}

// ListPublishedVideoOutput 获取已发布视频列表输出
type ListPublishedVideoOutput struct {
	Videos []*Video
	Total  int64
}

// ============ 业务模型 ============

// VideoAuthor 视频作者
type VideoAuthor struct {
	ID          int64
	Name        string
	Avatar      string
	IsFollowing bool
}

// Video 视频
type Video struct {
	ID             int64
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

// FileInfo 文件信息
type FileInfo struct {
	ObjectName string `json:"object_name"`
	Hash       string `json:"hash"`
}

// ============ VideoAssembler 视频组装器 ============

// VideoAssembler 视频组装器，负责组装视频的完整信息
type VideoAssembler struct {
	core CoreAdapter
	log  *log.Helper
}

// NewVideoAssembler 创建视频组装器
func NewVideoAssembler(
	core CoreAdapter,
	logger log.Logger,
) *VideoAssembler {
	return &VideoAssembler{
		core: core,
		log:  log.NewHelper(logger),
	}
}

// AssembleVideo 组装单个视频信息
func (a *VideoAssembler) AssembleVideo(ctx context.Context, video *Video, currentUserID int64) (*Video, error) {
	if video == nil {
		return nil, nil
	}

	videos := []*Video{video}
	assembledVideos, err := a.AssembleVideos(ctx, videos, currentUserID)
	if err != nil {
		return nil, err
	}

	if len(assembledVideos) == 0 {
		return nil, nil
	}

	return assembledVideos[0], nil
}

// AssembleVideos 批量组装视频信息
func (a *VideoAssembler) AssembleVideos(ctx context.Context, videos []*Video, currentUserID int64) ([]*Video, error) {
	if len(videos) == 0 {
		return videos, nil
	}

	// 收集需要查询的ID
	videoIDs := make([]int64, 0, len(videos))
	authorIDs := make([]int64, 0, len(videos))

	for _, video := range videos {
		videoIDs = append(videoIDs, video.ID)
		if video.Author != nil {
			authorIDs = append(authorIDs, video.Author.ID)
		}
	}

	// 并行获取所有需要的信息
	userInfos, err := a.getUserInfos(ctx, authorIDs)
	if err != nil {
		return nil, err
	}

	counts, err := a.getVideoCounts(ctx, videoIDs)
	if err != nil {
		return nil, err
	}

	interactions, err := a.getUserInteractions(ctx, currentUserID, videoIDs, authorIDs)
	if err != nil {
		return nil, err
	}

	// 组装视频信息
	return a.doAssembleVideos(videos, userInfos, counts, interactions), nil
}

// getUserInfos 批量获取用户信息
func (a *VideoAssembler) getUserInfos(ctx context.Context, userIDs []int64) (map[int64]*UserInfo, error) {
	if len(userIDs) == 0 {
		return map[int64]*UserInfo{}, nil
	}

	userInfoList, err := a.core.GetUserInfoByIdList(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*UserInfo)
	for _, user := range userInfoList {
		userMap[user.Id] = user
	}

	return userMap, nil
}

// getVideoCounts 批量获取视频计数信息
func (a *VideoAssembler) getVideoCounts(ctx context.Context, videoIDs []int64) (*VideoCountInfo, error) {
	if len(videoIDs) == 0 {
		return &VideoCountInfo{}, nil
	}

	var commentCounts, favoriteCounts, collectCounts map[int64]int64
	var commentErr, favoriteErr, collectErr error

	// 并行查询各种计数
	commentCounts, commentErr = a.core.CountComments4Video(ctx, videoIDs)
	favoriteCounts, favoriteErr = a.core.CountFavorite4Video(ctx, videoIDs)
	collectCounts, collectErr = a.core.CountCollected4Video(ctx, videoIDs)

	// 记录错误但不中断流程
	if commentErr != nil {
		a.log.WithContext(ctx).Warnf("failed to count comments: %v", commentErr)
	}
	if favoriteErr != nil {
		a.log.WithContext(ctx).Warnf("failed to count favorites: %v", favoriteErr)
	}
	if collectErr != nil {
		a.log.WithContext(ctx).Warnf("failed to count collects: %v", collectErr)
	}

	return &VideoCountInfo{
		CommentCounts:  commentCounts,
		FavoriteCounts: favoriteCounts,
		CollectCounts:  collectCounts,
	}, nil
}

// getUserInteractions 批量获取用户互动信息
func (a *VideoAssembler) getUserInteractions(ctx context.Context, userID int64, videoIDs, authorIDs []int64) (*UserInteractionInfo, error) {
	// 如果是未登录用户，不查询互动信息
	if userID <= 0 {
		return &UserInteractionInfo{}, nil
	}

	var isFavoriteMap, isCollectMap, isFollowingMap map[int64]bool
	var favoriteErr, collectErr, followErr error

	// 并行查询用户互动状态
	if len(videoIDs) > 0 {
		isFavoriteMap, favoriteErr = a.core.IsUserFavoriteVideo(ctx, userID, videoIDs)
		isCollectMap, collectErr = a.core.IsCollected(ctx, userID, videoIDs)
	}

	if len(authorIDs) > 0 {
		isFollowingMap, followErr = a.core.IsFollowing(ctx, userID, authorIDs)
	}

	// 记录错误但不中断流程
	if favoriteErr != nil {
		a.log.WithContext(ctx).Warnf("failed to check favorites: %v", favoriteErr)
	}
	if collectErr != nil {
		a.log.WithContext(ctx).Warnf("failed to check collects: %v", collectErr)
	}
	if followErr != nil {
		a.log.WithContext(ctx).Warnf("failed to check follows: %v", followErr)
	}

	return &UserInteractionInfo{
		IsFavoriteMap:  isFavoriteMap,
		IsCollectMap:   isCollectMap,
		IsFollowingMap: isFollowingMap,
	}, nil
}

// doAssembleVideos 执行视频组装
func (a *VideoAssembler) doAssembleVideos(
	videos []*Video,
	userInfos map[int64]*UserInfo,
	counts *VideoCountInfo,
	interactions *UserInteractionInfo,
) []*Video {
	result := make([]*Video, 0, len(videos))

	for _, video := range videos {
		assembledVideo := a.assembleSingleVideo(video, userInfos, counts, interactions)
		result = append(result, assembledVideo)
	}

	return result
}

// assembleSingleVideo 组装单个视频
func (a *VideoAssembler) assembleSingleVideo(
	video *Video,
	userInfos map[int64]*UserInfo,
	counts *VideoCountInfo,
	interactions *UserInteractionInfo,
) *Video {
	// 组装作者信息
	if video.Author != nil {
		if userInfo, exists := userInfos[video.Author.ID]; exists {
			video.Author.Name = userInfo.Name
			video.Author.Avatar = userInfo.Avatar
		}
		if interactions.IsFollowingMap != nil {
			video.Author.IsFollowing = interactions.IsFollowingMap[video.Author.ID]
		}
	}

	// 组装计数信息
	if counts.CommentCounts != nil {
		video.CommentCount = counts.CommentCounts[video.ID]
	}
	if counts.FavoriteCounts != nil {
		video.FavoriteCount = counts.FavoriteCounts[video.ID]
	}
	if counts.CollectCounts != nil {
		video.CollectedCount = counts.CollectCounts[video.ID]
	}

	// 组装互动状态
	if interactions.IsFavoriteMap != nil {
		video.IsFavorite = interactions.IsFavoriteMap[video.ID]
	}
	if interactions.IsCollectMap != nil {
		video.IsCollected = interactions.IsCollectMap[video.ID]
	}

	return video
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
	if input.FileID <= 0 {
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
	if input.FileID <= 0 || input.Title == "" || input.VideoURL == "" || input.CoverURL == "" {
		return nil, ErrInvalidParams
	}
	if input.UserID <= 0 {
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
	if input.VideoID <= 0 {
		return nil, ErrInvalidParams
	}

	// 获取视频基本信息
	video, err := uc.core.GetVideoById(ctx, input.VideoID)
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
func (uc *VideoUsecase) FeedVideo(ctx context.Context, input *FeedVideoInput) (*FeedVideoOutput, error) {
	// 设置默认值
	if input.FeedNum <= 0 {
		input.FeedNum = 30
	}

	// 获取视频列表
	videos, err := uc.core.Feed(ctx, input.UserID, input.FeedNum, input.LatestTime)
	if err != nil {
		return nil, err
	}

	// 组装完整的视频信息
	videos, err = uc.assembler.AssembleVideos(ctx, videos, input.UserID)
	if err != nil {
		return nil, err
	}

	// 计算下次请求的时间（这里简化处理，实际可能需要根据视频创建时间计算）
	var nextTime int64
	if len(videos) > 0 {
		// 使用最后一个视频的时间作为下次请求的latest_time
		nextTime = time.Now().Unix()
	}

	return &FeedVideoOutput{
		Videos:   videos,
		NextTime: nextTime,
	}, nil
}

// ListPublishedVideo 获取已发布视频列表
func (uc *VideoUsecase) ListPublishedVideo(ctx context.Context, input *ListPublishedVideoInput) (*ListPublishedVideoOutput, error) {
	if input.UserID <= 0 {
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
		Page:     int32(input.Page),
		PageSize: int32(input.PageSize),
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
	CommentCounts  map[int64]int64
	FavoriteCounts map[int64]int64
	CollectCounts  map[int64]int64
}

// UserInteractionInfo 用户互动信息
type UserInteractionInfo struct {
	IsFavoriteMap  map[int64]bool
	IsCollectMap   map[int64]bool
	IsFollowingMap map[int64]bool
}

// ============ 错误定义 ============

var (
	ErrInvalidParams = errors.New("invalid parameters")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrVideoNotFound = errors.New("video not found")
)
