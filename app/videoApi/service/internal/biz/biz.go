package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewUserUsecase,
	NewFileUsecase,
	NewVideoUsecase,
	NewVideoAssembler,
	NewCommentUsecase,
	NewFavoriteUsecase,
	NewFollowUsecase,
	NewCollectionUsecase,
	NewGroupUsecase,
	NewMessageUsecase,
	NewFriendUsecase,
)
