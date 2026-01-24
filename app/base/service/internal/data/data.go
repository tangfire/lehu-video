package data

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewRedis,
	NewAuthRepo,
	NewDB,
	NewAccountRepo,
	NewMinioRepo,
	NewMinioClient,
	NewMinioCore,
	NewFileRepo,
	NewFileRepoHelper,
	NewFileShardingConfig,
)

// Data .
type Data struct {
	// TODO wrapped database client
	rds *redis.Client
	db  *gorm.DB
	log *log.Helper
}

// NewData .
func NewData(db *gorm.DB, rds *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{
		db:  db,
		rds: rds,
		log: log.NewHelper(logger),
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
