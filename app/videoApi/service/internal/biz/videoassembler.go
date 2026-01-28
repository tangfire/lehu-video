package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

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
