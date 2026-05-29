package data

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	core "lehu-video/api/campusUser/service/v1"
	"lehu-video/app/campusApi/service/internal/biz"
)

type CoreAdapterImpl struct {
	user core.UserServiceClient
	log  *log.Helper
}

func NewCampusCoreAdapter(user core.UserServiceClient, logger log.Logger) biz.CoreAdapter {
	return &CoreAdapterImpl{
		user: user,
		log:  log.NewHelper(logger),
	}
}

func NewUserServiceClient(r registry.Discovery) (core.UserServiceClient, error) {
	conn, err := dialService(r, "discovery:///campus-estation.user.service")
	if err != nil {
		return nil, fmt.Errorf("dial user service: %w", err)
	}
	return core.NewUserServiceClient(conn), nil
}
