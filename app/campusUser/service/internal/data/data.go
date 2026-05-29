package data

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/campusUser/service/internal/conf"
	"lehu-video/app/campusUser/service/internal/pkg/idgen"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewRedis,
	NewUserRepo,
	NewUserCounterRepo,
	NewIdGenerator,
)

type Data struct {
	db              *gorm.DB
	log             *log.Helper
	rdb             *redis.Client
	userSyncJob     *UserCounterSyncJob
	reconcileStopCh chan struct{}
	stopOnce        sync.Once
}

func (d *Data) PingMySQL(ctx context.Context) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("mysql is not initialized")
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("get mysql db: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}
	return nil
}

func (d *Data) PingRedis(ctx context.Context) error {
	if d == nil || d.rdb == nil {
		return fmt.Errorf("redis is not initialized")
	}
	if err := d.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

func NewData(db *gorm.DB, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(logger)
	d := &Data{
		db:              db,
		rdb:             rdb,
		log:             logHelper,
		reconcileStopCh: make(chan struct{}),
	}

	counterRepo := NewUserCounterRepo(rdb, logger)
	userSyncJob := NewUserCounterSyncJob(db, counterRepo, logger)
	userSyncJob.Start()
	d.userSyncJob = userSyncJob

	cleanup := func() {
		d.stopOnce.Do(func() {
			close(d.reconcileStopCh)
			if d.userSyncJob != nil {
				d.userSyncJob.Stop()
			}
			if err := d.rdb.Close(); err != nil {
				logHelper.Errorf("Redis close error: %v", err)
			}
		})
	}
	return d, cleanup, nil
}

// NewIdGenerator 从配置创建 ID 生成器
func NewIdGenerator(c *conf.Idgen) idgen.Generator {
	return idgen.NewGenerator(c.WorkerId)
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

func NewRedis(c *conf.Data) (*redis.Client, error) {
	rds := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       0,
	})
	if err := rds.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return rds, nil
}
