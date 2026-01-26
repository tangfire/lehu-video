package biz

import (
	"context"
)

// CoreAdapter 核心服务适配器接口
type CoreAdapter interface {
	CreateUser(ctx context.Context, mobile, email string, accountId int64) (int64, error)
	GetUserInfo(ctx context.Context, userId, accountId int64) (*UserInfo, error)
	GetUserInfoByIdList(ctx context.Context, userIdList []int64) ([]*UserInfo, error)
	UpdateUserInfo(ctx context.Context, userId int64, name, avatar, backgroundImage, signature string) error
	SaveVideoInfo(ctx context.Context, title, videoUrl, coverUrl, desc string, userId int64) (int64, error)
	GetVideoById(ctx context.Context, videoId int64) (*Video, error)
	ListPublishedVideo(ctx context.Context, userId int64, pageStats *PageStats) (int64, []*Video, error)
	IsUserFavoriteVideo(ctx context.Context, userId int64, videoIdList []int64) (map[int64]bool, error)
	IsFollowing(ctx context.Context, userId int64, targetUserIdList []int64) (map[int64]bool, error)
	IsCollected(ctx context.Context, userId int64, videoIdList []int64) (map[int64]bool, error)
	CountComments4Video(ctx context.Context, videoIdList []int64) (map[int64]int64, error)
	CountFavorite4Video(ctx context.Context, videoIdList []int64) (map[int64]int64, error)
	CountCollected4Video(ctx context.Context, videoIdList []int64) (map[int64]int64, error)
	Feed(ctx context.Context, userId int64, num int64, latestTime int64) ([]*Video, error)
}
