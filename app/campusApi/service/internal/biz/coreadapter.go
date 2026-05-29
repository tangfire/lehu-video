package biz

import (
	"context"
)

// CoreAdapter 核心服务适配器接口
type CoreAdapter interface {
	CreateUser(ctx context.Context, mobile, email, accountId string) (string, error)
	GetUserBaseInfo(ctx context.Context, userID, accountID string) (*UserBaseInfo, error)
	BatchGetUserBaseInfo(ctx context.Context, userIDs []string) ([]*UserBaseInfo, error)
	UpdateUserInfo(ctx context.Context, userID, name, nickName, avatar, backgroundImage, signature string, gender int32) error
	SearchUsers(ctx context.Context, keyword string, page, pageSize int32) (int64, []*UserBaseInfo, error)
}
