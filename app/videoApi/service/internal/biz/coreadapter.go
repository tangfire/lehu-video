package biz

import (
	"context"
)

// CoreAdapter 核心服务适配器接口
type CoreAdapter interface {
	CreateUser(ctx context.Context, mobile, email string, accountId int64) (int64, error)
	GetUserInfo(ctx context.Context, userId, accountId int64) (*UserInfo, error)
	UpdateUserInfo(ctx context.Context, userId int64, name, avatar, backgroundImage, signature string) error
}
