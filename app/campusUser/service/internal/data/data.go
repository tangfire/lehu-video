package data

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/campusUser/service/internal/conf"
	"lehu-video/app/campusUser/service/internal/pkg/idgen"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewUserRepo,
	NewIdGenerator,
)

type Data struct {
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

func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(logger)
	d := &Data{
		db:  db,
		log: logHelper,
	}
	cleanup := func() {}
	return d, cleanup, nil
}

// NewIdGenerator 从配置创建 ID 生成器
func NewIdGenerator(c *conf.Idgen) idgen.Generator {
	return idgen.NewGenerator(c.WorkerId)
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
