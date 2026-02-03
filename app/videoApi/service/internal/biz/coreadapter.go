package biz

import (
	"context"
)

// CoreAdapter 核心服务适配器接口
type CoreAdapter interface {
	CreateUser(ctx context.Context, mobile, email, accountId string) (string, error)
	GetUserBaseInfo(ctx context.Context, userID, accountID string) (*UserBaseInfo, error)
	BatchGetUserBaseInfo(ctx context.Context, userIDs []string) ([]*UserBaseInfo, error)
	GetUserInfoByIdList(ctx context.Context, userIdList []string) ([]*UserInfo, error)
	UpdateUserInfo(ctx context.Context, userID, name, nickName, avatar, backgroundImage, signature string, gender int32) error
	SearchUsers(ctx context.Context, keyword string, page, pageSize int32) (int64, []*UserBaseInfo, error)
	SaveVideoInfo(ctx context.Context, title, videoUrl, coverUrl, desc, userId string) (string, error)
	GetVideoById(ctx context.Context, videoId string) (*Video, error)
	GetVideoByIdList(ctx context.Context, videoIdList []string) ([]*Video, error)
	ListPublishedVideo(ctx context.Context, userId string, pageStats *PageStats) (int64, []*Video, error)
	IsUserFavoriteVideo(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error)
	IsFollowing(ctx context.Context, userId string, targetUserIdList []string) (map[string]bool, error)
	IsCollected(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error)
	CountComments4Video(ctx context.Context, videoIdList []string) (map[string]int64, error)
	CountFavorite4Video(ctx context.Context, videoIdList []string) (map[string]FavoriteCount, error)
	CountCollected4Video(ctx context.Context, videoIdList []string) (map[string]int64, error)
	Feed(ctx context.Context, userId string, num int64, latestTime int64) ([]*Video, error)
	CreateComment(ctx context.Context, userId string, content string, videoId string, parentId string, replyUserId string) (*Comment, error)
	GetCommentById(ctx context.Context, commentId string) (*Comment, error)
	RemoveComment(ctx context.Context, commentId, userId string) error
	ListChildComment(ctx context.Context, commentId string, pageStats *PageStats) (int64, []*Comment, error)
	ListComment4Video(ctx context.Context, videoId string, pageStats *PageStats) (int64, []*Comment, error)

	// 点赞相关接口（新增）
	AddFavorite(ctx context.Context, id, userId string, target *FavoriteTarget, _type *FavoriteType) (*AddFavoriteResult, error)
	RemoveFavorite(ctx context.Context, id, userId string, target *FavoriteTarget, _type *FavoriteType) (*RemoveFavoriteResult, error)
	ListUserFavoriteVideo(ctx context.Context, userId string, pageStats *PageStats) (int64, []string, error)
	CheckFavoriteStatus(ctx context.Context, userId, targetId string, target *FavoriteTarget, _type *FavoriteType) (*CheckFavoriteResult, error)
	GetFavoriteStats(ctx context.Context, targetId string, target *FavoriteTarget) (*FavoriteStats, error)
	CountBeFavoriteNumber4User(ctx context.Context, userId string) (int64, error)

	AddFollow(ctx context.Context, userId, targetUserId string) error
	RemoveFollow(ctx context.Context, userId, targetUserId string) error
	ListFollow(ctx context.Context, userId string, _type *FollowType, pageStats *PageStats) (int64, []string, error)
	GetCollectionById(ctx context.Context, collectionId string) (*Collection, error)
	AddVideo2Collection(ctx context.Context, userId string, collectionId string, videoId string) error
	AddCollection(ctx context.Context, collection *Collection) error
	ListCollection(ctx context.Context, userId string, pageStats *PageStats) (int64, []*Collection, error)
	ListVideo4Collection(ctx context.Context, collectionId string, pageStats *PageStats) (int64, []string, error)
	RemoveCollection(ctx context.Context, userId, collectionId string) error
	RemoveVideo4Collection(ctx context.Context, userId string, collectionId string, videoId string) error
	UpdateCollection(ctx context.Context, collection *Collection) error
	CountFollow4User(ctx context.Context, userId string) ([]int64, error)
}

// 新增类型定义
type FavoriteCount struct {
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
}

type AddFavoriteResult struct {
	AlreadyFavorited bool
	TotalCount       int64
	TotalLikes       int64
	TotalDislikes    int64
	PreviousType     int32
}

type RemoveFavoriteResult struct {
	NotFavorited  bool
	TotalCount    int64
	TotalLikes    int64
	TotalDislikes int64
}

type CheckFavoriteResult struct {
	IsFavorite    bool
	FavoriteType  int32
	TotalLikes    int64
	TotalDislikes int64
	TotalCount    int64
}

type FavoriteStatus struct {
	IsLiked      bool
	IsDisliked   bool
	LikeCount    int64
	DislikeCount int64
}

type FavoriteStats struct {
	LikeCount    int64
	DislikeCount int64
	TotalCount   int64
	HotScore     float64
}

// 枚举类型
type FavoriteTarget int32
type FavoriteType int32

const (
	FavoriteTargetVideo   FavoriteTarget = 0
	FavoriteTargetComment FavoriteTarget = 1
)

const (
	FavoriteTypeLike    FavoriteType = 0
	FavoriteTypeDislike FavoriteType = 1
)
