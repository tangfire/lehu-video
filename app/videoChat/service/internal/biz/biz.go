package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewGroupUsecase,
	NewMessageUsecase,
	NewFriendUsecase,
)

// 分页统计
type PageStats struct {
	Page     int
	PageSize int
	Sort     string
}
