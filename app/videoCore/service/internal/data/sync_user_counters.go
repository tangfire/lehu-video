package data

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type UserCounterSyncJob struct {
	db          *gorm.DB
	counterRepo biz.CounterRepo
	log         *log.Helper
	interval    time.Duration
	stopCh      chan struct{}
}

func NewUserCounterSyncJob(db *gorm.DB, counterRepo biz.CounterRepo, logger log.Logger) *UserCounterSyncJob {
	return &UserCounterSyncJob{
		db:          db,
		counterRepo: counterRepo,
		log:         log.NewHelper(logger),
		interval:    5 * time.Minute, // 默认每5分钟同步一次
		stopCh:      make(chan struct{}),
	}
}

func (j *UserCounterSyncJob) Start() {
	go j.run()
}

func (j *UserCounterSyncJob) Stop() {
	close(j.stopCh)
}

func (j *UserCounterSyncJob) run() {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	j.sync() // 立即执行一次
	for {
		select {
		case <-ticker.C:
			j.sync()
		case <-j.stopCh:
			return
		}
	}
}

func (j *UserCounterSyncJob) sync() {
	ctx := context.Background()
	userIDs, err := j.counterRepo.GetDirtyUserIDs(ctx)
	if err != nil {
		j.log.Errorf("获取脏用户ID列表失败: %v", err)
		return
	}
	if len(userIDs) == 0 {
		return
	}
	j.log.Infof("开始同步用户计数器，用户数量: %d", len(userIDs))

	// 批量获取 Redis 中的计数值
	fields := []string{"follow_count", "follower_count", "total_favorited", "work_count", "favorite_count"}
	countersMap, err := j.counterRepo.BatchGetUserCounters(ctx, userIDs, fields)
	if err != nil {
		j.log.Errorf("批量获取用户计数器失败: %v", err)
		return
	}

	// 批量更新 MySQL
	err = j.batchUpdateMySQL(ctx, countersMap)
	if err != nil {
		j.log.Errorf("批量更新 MySQL 失败: %v", err)
		return
	}

	// 清除脏标记
	for _, uid := range userIDs {
		if err := j.counterRepo.ClearDirtyFlag(ctx, uid); err != nil {
			j.log.Warnf("清除用户 %d 脏标记失败: %v", uid, err)
		}
	}
	j.log.Infof("用户计数器同步完成，更新 %d 个用户", len(countersMap))
}

func (j *UserCounterSyncJob) batchUpdateMySQL(ctx context.Context, countersMap map[int64]map[string]int64) error {
	if len(countersMap) == 0 {
		return nil
	}
	// 使用事务批量更新
	tx := j.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for uid, counters := range countersMap {
		updates := make(map[string]interface{})
		for field, val := range counters {
			updates[field] = val
		}
		if len(updates) == 0 {
			continue
		}
		updates["updated_at"] = time.Now()
		if err := tx.Table("user").Where("id = ?", uid).Updates(updates).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
