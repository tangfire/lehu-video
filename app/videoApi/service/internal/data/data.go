package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/videoApi/service/internal/conf"
	"lehu-video/app/videoApi/service/internal/pkg/kafka"
	"lehu-video/app/videoApi/service/internal/pkg/websocket"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewRedis,
	NewDB,
	NewDiscovery,
	NewRegistrar,
	NewAccountServiceClient,
	NewAuthServiceClient,
	NewUserServiceClient,
	NewVideoServiceClient,
	NewCollectionServiceClient,
	NewCommentServiceClient,
	NewFavoriteServiceClient,
	NewFollowServiceClient,
	NewBaseAdapter,
	NewCoreAdapter,
	NewFileServiceClient,
	NewGroupServiceClient,
	NewChatAdapter,
	NewMessageServiceClient,
	NewFriendServiceClient,
	NewFeedServiceClient,
	kafka.NewKafkaConsumer,
	kafka.NewKafkaProducer,
	websocket.NewManager,
)

// Data .
type Data struct {
	// TODO wrapped database client
	rds  *redis.Client
	db   *gorm.DB
	log  *log.Helper
	base *baseAdapterImpl
	core *CoreAdapterImpl
}

// NewData .
func NewData(db *gorm.DB, rds *redis.Client, base *baseAdapterImpl, core *CoreAdapterImpl, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if rds != nil {
			if err := rds.Close(); err != nil {
				log.NewHelper(logger).Warnf("close redis error: %v", err)
			}
		}
	}
	return &Data{
		rds:  rds,
		db:   db,
		log:  log.NewHelper(logger),
		base: base,
		core: core,
	}, cleanup, nil
}

func NewRedis(c *conf.Data) (*redis.Client, error) {
	rds := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       int(c.Redis.Db),
	})
	if err := rds.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return rds, nil
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

func NewRegistrar(conf *conf.Registry) (registry.Registrar, error) {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("new consul registrar: %w", err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r, nil
}
