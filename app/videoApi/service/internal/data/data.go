package data

import (
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/videoApi/service/internal/conf"
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
	}
	return &Data{
		rds:  rds,
		db:   db,
		log:  log.NewHelper(logger),
		base: base,
		core: core,
	}, cleanup, nil
}

func NewRedis(c *conf.Data) *redis.Client {
	rds := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       int(c.Redis.Db),
	})
	return rds
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

func NewRegistrar(conf *conf.Registry) registry.Registrar {
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
