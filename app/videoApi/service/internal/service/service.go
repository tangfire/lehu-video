package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewUserServiceService,
	NewFileServiceService,
	NewVideoServiceService,
	NewCommentServiceService,
	NewFavoriteServiceService,
	NewFollowServiceService,
	NewCollectionServiceService,
	NewWebSocketService,
	NewGroupServiceService,
	NewMessageServiceService,
	NewFriendServiceService,
)
