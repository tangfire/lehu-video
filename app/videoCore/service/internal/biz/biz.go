package biz

import (
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoUsecase,
	NewUserUsecase,
	NewFavoriteUsecase,
	NewCommentUsecase,
	NewFollowUsecase,
	NewCollectionUsecase,
	NewFeedUsecase,
	NewVideoConsumer,
	NewVideoProducer,
	NewRecentViewedManager,
	NewFavoriteConsumer,
)

type PageStats struct {
	Page     int32
	PageSize int32
}
