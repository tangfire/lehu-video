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
)

type PageStats struct {
	Page     int32
	PageSize int32
}
