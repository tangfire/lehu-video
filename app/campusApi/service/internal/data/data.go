package data

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/campusApi/service/internal/conf"
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
	NewBaseAdapter,
	NewFileServiceClient,
	NewCampusCoreAdapter,
	NewCampusRepo,
)

// Data .
type Data struct {
	// TODO wrapped database client
	rds *redis.Client
	db  *gorm.DB
	log *log.Helper
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
	if d == nil || d.rds == nil {
		return fmt.Errorf("redis is not initialized")
	}
	if err := d.rds.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
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
		rds: rds,
		db:  db,
		log: log.NewHelper(logger),
	}, cleanup, nil
}

func NewRedis(c *conf.Data) (*redis.Client, error) {
	db := int(c.Redis.Db)
	if value := strings.TrimSpace(os.Getenv("LEHU_REDIS_DB")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid LEHU_REDIS_DB: %w", err)
		}
		db = parsed
	}
	rds := redis.NewClient(&redis.Options{
		Addr:     firstEnv("LEHU_REDIS_ADDR", c.Redis.Addr),
		Password: firstEnv("LEHU_REDIS_PASSWORD", c.Redis.Password),
		DB:       db,
	})
	if err := rds.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return rds, nil
}

func NewDB(c *conf.Data, logger log.Logger) (*gorm.DB, error) {
	source := firstEnv("LEHU_MYSQL_DSN", c.Database.Source)
	db, err := gorm.Open(mysql.Open(source), &gorm.Config{})
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

func firstEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
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
