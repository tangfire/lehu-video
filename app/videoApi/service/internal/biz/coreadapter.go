package biz

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
)

type CoreAdapterRepo interface {
	GetUserInfo(ctx context.Context, userId, accountId int64) (*core.User, error)
	CountFollow4User(ctx context.Context, userId int64) ([]int64, error)
	CountBeFavoriteNumber4User(ctx context.Context, userId int64) (int64, error)
	CreateUser(ctx context.Context, mobile, email string, accountId int64) (int64, error)
}

type CoreAdapter struct {
	repo CoreAdapterRepo
}

func NewCoreAdapter(repo CoreAdapterRepo) *CoreAdapter {
	return &CoreAdapter{repo: repo}
}
