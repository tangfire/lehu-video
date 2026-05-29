package biz

import (
	"os"
	"strings"

	"github.com/google/wire"
	"lehu-video/app/campusApi/service/internal/conf"
)

func NewAuthSecret(auth *conf.Auth) string {
	if value := strings.TrimSpace(os.Getenv("LEHU_JWT_SECRET")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("LEHU_AUTH_API_KEY")); value != "" {
		return value
	}
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
