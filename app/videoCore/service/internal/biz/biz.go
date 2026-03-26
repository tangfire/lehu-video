package biz

import (
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoUsecase,
	NewUserUsecase,
	NewFavoriteUsecase, // 已更新，包含 commentCounter
	NewCommentUsecase,
	NewFollowUsecase,
	NewCollectionUsecase,
	NewFeedUsecase,
	NewVideoConsumer,
	NewVideoProducer,
	NewRecentViewedManager,
	NewFavoriteConsumer, // 已更新，包含 commentCounter
)

type PageStats struct {
	Page     int32
	PageSize int32
}
