package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
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
	NewIdGenerator,
	NewConversationRepo,
	NewRedisClient,
)

// Data .
type Data struct {
	// TODO wrapped database client
	db    *gorm.DB
	redis *redis.Client
	log   *log.Helper
	user  core.UserServiceClient
}

// NewData .
func NewData(db *gorm.DB, user core.UserServiceClient, redis *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if redis != nil {
			if err := redis.Close(); err != nil {
				log.NewHelper(logger).Warnf("close redis error: %v", err)
			}
		}
	}
	return &Data{db: db, user: user, redis: redis, log: log.NewHelper(logger)}, cleanup, nil
}

// NewIdGenerator 从配置创建 ID 生成器
func NewIdGenerator(c *conf.Idgen) idgen.Generator {
	return idgen.NewGenerator(c.WorkerId)
}

func NewUserServiceClient(r registry.Discovery) (core.UserServiceClient, error) {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
			tracing.Client(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("dial core user service: %w", err)
	}
	return core.NewUserServiceClient(conn), nil
}

func NewDB(c *conf.Data, logger log.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get mysql db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

func NewDiscovery(conf *conf.Registry) (registry.Discovery, error) {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("new consul discovery: %w", err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r, nil
}

func NewRedisClient(conf *conf.Data) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Addr,
		Password: conf.Redis.Password,
		DB:       int(conf.Redis.Db),
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return client, nil
}
