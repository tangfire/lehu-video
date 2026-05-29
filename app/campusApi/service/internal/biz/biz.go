package biz

import (
	"github.com/google/wire"
	"lehu-video/app/campusApi/service/internal/conf"
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
	NewMockCampusTimetableProvider,
	NewCampusIDGenerator,
	NewCampusRAGClient,
	NewCampusUsecase,
)
