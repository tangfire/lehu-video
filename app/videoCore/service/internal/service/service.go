package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewVideoServiceService,
	NewFollowServiceService,
	NewFavoriteServiceService,
	NewCommentServiceService,
	NewUserServiceService,
	NewCollectionServiceService,
	NewFeedServiceService,
	NewLikeConsumer,
)
