package biz

import (
	"context"
)

// CoreAdapter 核心服务适配器接口
type CoreAdapter interface {
	CreateUser(ctx context.Context, mobile, email, accountId string) (string, error)
	GetUserInfo(ctx context.Context, userId, accountId string) (*UserInfo, error)
	GetUserInfoByIdList(ctx context.Context, userIdList []string) ([]*UserInfo, error)
	UpdateUserInfo(ctx context.Context, userId, name, avatar, backgroundImage, signature string) error
	SaveVideoInfo(ctx context.Context, title, videoUrl, coverUrl, desc, userId string) (string, error)
	GetVideoById(ctx context.Context, videoId string) (*Video, error)
	GetVideoByIdList(ctx context.Context, videoIdList []string) ([]*Video, error)
	ListPublishedVideo(ctx context.Context, userId string, pageStats *PageStats) (int64, []*Video, error)
	IsUserFavoriteVideo(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error)
	IsFollowing(ctx context.Context, userId string, targetUserIdList []string) (map[string]bool, error)
	IsCollected(ctx context.Context, userId string, videoIdList []string) (map[string]bool, error)
	CountComments4Video(ctx context.Context, videoIdList []string) (map[string]int64, error)
	CountFavorite4Video(ctx context.Context, videoIdList []string) (map[string]int64, error)
	CountCollected4Video(ctx context.Context, videoIdList []string) (map[string]int64, error)
	Feed(ctx context.Context, userId string, num int64, latestTime int64) ([]*Video, error)
	CreateComment(ctx context.Context, userId string, content string, videoId string, parentId string, replyUserId string) (*Comment, error)
	GetCommentById(ctx context.Context, commentId string) (*Comment, error)
	RemoveComment(ctx context.Context, commentId, userId string) error
	ListChildComment(ctx context.Context, commentId string, pageStats *PageStats) (int64, []*Comment, error)
	ListComment4Video(ctx context.Context, videoId string, pageStats *PageStats) (int64, []*Comment, error)
	AddFavorite(ctx context.Context, id, userId string, target *FavoriteTarget, _type *FavoriteType) error
	RemoveFavorite(ctx context.Context, id, userId string, target *FavoriteTarget, _type *FavoriteType) error
	ListUserFavoriteVideo(ctx context.Context, userId string, pageStats *PageStats) (int64, []string, error)
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
