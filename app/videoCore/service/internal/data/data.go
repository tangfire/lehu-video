package data

import (
	"github.com/coocood/freecache"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/conf"
	"lehu-video/app/videoCore/service/internal/pkg/idgen"
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
	NewFreeCache,
	NewUserCounterRepo,
	NewIdGenerator,
	NewVideoCounterRepo,
)

type Data struct {
	db              *gorm.DB
	log             *log.Helper
	rdb             *redis.Client
	userSyncJob     *UserCounterSyncJob
	videoSyncJob    *VideoCounterSyncJob
	reconcileStopCh chan struct{} // 用于停止对账任务
}

func NewData(db *gorm.DB, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(logger)
	d := &Data{
		db:              db,
		rdb:             rdb,
		log:             logHelper,
		reconcileStopCh: make(chan struct{}),
	}

	// 创建并启动用户计数器同步任务
	counterRepo := NewUserCounterRepo(rdb, logger)
	userSyncJob := NewUserCounterSyncJob(db, counterRepo, logger)
	userSyncJob.Start()
	d.userSyncJob = userSyncJob

	// 创建并启动视频计数器同步任务
	videoCounterRepo := NewVideoCounterRepo(rdb, logger)
	videoSyncJob := NewVideoCounterSyncJob(db, videoCounterRepo, logger)
	videoSyncJob.Start()
	d.videoSyncJob = videoSyncJob

	// 创建并启动视频统计对账任务（每日凌晨3点执行）
	videoStatsReconcileJob := NewVideoStatsReconcileJob(db, logger)
	videoStatsReconcileJob.StartCron(d.reconcileStopCh)

	// 创建并启动用户获赞数对账任务（每日凌晨4点执行）
	userBeLikedReconcileJob := NewUserBeLikedReconcileJob(db, logger)
	userBeLikedReconcileJob.StartCron(d.reconcileStopCh)

	cleanup := func() {
		// 停止对账任务（关闭通道）
		close(d.reconcileStopCh)

		if d.userSyncJob != nil {
			d.userSyncJob.Stop()
		}
		if d.videoSyncJob != nil {
			d.videoSyncJob.Stop()
		}
		if err := d.rdb.Close(); err != nil {
			logHelper.Errorf("Redis close error: %v", err)
		}
	}
	return d, cleanup, nil
}

// NewIdGenerator 从配置创建 ID 生成器
func NewIdGenerator(c *conf.Idgen) idgen.Generator {
	return idgen.NewGenerator(c.WorkerId)
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

func NewFreeCache(c *conf.Data) *freecache.Cache {
	localCache := freecache.NewCache(int(c.LocalCache.Size))
	return localCache
}
