package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewUserUsecase,
	NewFileUsecase,
	NewVideoUsecase,
	NewVideoAssembler,
	NewCommentUsecase,
	NewFavoirteUsecase,
	NewFollowUsecase,
	NewCollectionUsecase,
	NewGroupUsecase,
	NewMessageUsecase,
)

type PageStats struct {
	Page     int32
	PageSize int32
}
