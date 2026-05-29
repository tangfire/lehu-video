package data

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/base/service/internal/conf"
	"lehu-video/app/base/service/internal/pkg/idgen"

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
	NewObjectStorageRepo,
	NewBizFileRepo,
	NewFileShardingConfig,
	NewFileRepoHelper,
	NewFileRepo,
	NewIdGenerator,
)

// Data .
type Data struct {
	// TODO wrapped database client
	rds *redis.Client
	db  *gorm.DB
	log *log.Helper
}

func (d *Data) Begin() *gorm.DB {
	return d.db.Begin()
}

// NewIdGenerator 从配置创建 ID 生成器
func NewIdGenerator(c *conf.Idgen) idgen.Generator {
	return idgen.NewGenerator(c.WorkerId)
}

// NewData .
func NewData(db *gorm.DB, rds *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if rds != nil {
			if err := rds.Close(); err != nil {
				log.NewHelper(logger).Warnf("close redis error: %v", err)
			}
		}
	}
	return &Data{
		db:  db,
		rds: rds,
		log: log.NewHelper(logger),
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
