package data

import (
	"context"
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	consulAPI "github.com/hashicorp/consul/api"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoChat/service/internal/conf"
	"lehu-video/app/videoChat/service/internal/pkg/idgen"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewGroupRepo,
	NewDB,
	NewMessageRepo,
	NewFriendRepo,
	NewUserServiceClient,
	NewDiscovery,
	idgen.NewSnowflakeIDGenerator,
	idgen.NewIDGenerator,
)

// Data .
type Data struct {
	// TODO wrapped database client
	db   *gorm.DB
	log  *log.Helper
	user core.UserServiceClient
}

// NewData .
func NewData(db *gorm.DB, user core.UserServiceClient, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{db: db, user: user, log: log.NewHelper(logger)}, cleanup, nil
}

func NewUserServiceClient(r registry.Discovery) core.UserServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewUserServiceClient(conn)
}

func NewDB(c *conf.Data, logger log.Logger) *gorm.DB {
	log := log.NewHelper(logger)
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		log.Errorf("open db err:%v", err)
	}
	return db
}

func NewDiscovery(conf *conf.Registry) registry.Discovery {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}
