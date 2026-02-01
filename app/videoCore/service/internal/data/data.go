package data

import (
	"github.com/redis/go-redis/v9"
	"lehu-video/app/videoCore/service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewVideoRepo,
	NewUserRepo,
	NewFollowRepo,
	NewFavoriteRepo,
	NewCommentRepo,
	NewCollectionRepo,
	NewRedis,
)

// Data .
type Data struct {
	// TODO wrapped database client
	db  *gorm.DB
	log *log.Helper
}

// NewData .
func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{db: db, log: log.NewHelper(logger)}, cleanup, nil
}

func NewDB(c *conf.Data, logger log.Logger) *gorm.DB {
	log := log.NewHelper(logger)
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		log.Errorf("open db err:%v", err)
	}
	return db
}

func NewRedis(c *conf.Data) *redis.Client {
	rds := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       0,
	})
	return rds
}
