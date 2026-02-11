package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoUsecase,
	NewUserUsecase,
	NewFavoriteUsecase,
	NewCommentUsecase,
	NewFollowUsecase,
	NewCollectionUsecase,
	NewFeedUsecase,
)

type PageStats struct {
	Page     int32
	PageSize int32
}
