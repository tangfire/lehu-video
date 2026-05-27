package biz

import (
	"github.com/google/wire"
	"lehu-video/app/videoApi/service/internal/conf"
)

func NewAuthSecret(auth *conf.Auth) string {
	if auth == nil || auth.ApiKey == "" {
		return "fireshine"
	}
	return auth.ApiKey
}

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewAuthSecret,
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
	NewMockCampusTimetableProvider,
	NewCampusIDGenerator,
	NewCampusUsecase,
)
