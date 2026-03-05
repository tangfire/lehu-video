// data/sync_video_counters.go
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
)

// VideoCounterSyncJob 视频计数器同步任务
type VideoCounterSyncJob struct {
	db        *gorm.DB
	videoRepo biz.VideoCounterRepo
	log       *log.Helper
	interval  time.Duration
	stopCh    chan struct{}
}

func NewVideoCounterSyncJob(db *gorm.DB, videoCounterRepo biz.VideoCounterRepo, logger log.Logger) *VideoCounterSyncJob {
	return &VideoCounterSyncJob{
		db:        db,
		videoRepo: videoCounterRepo,
		log:       log.NewHelper(logger),
		interval:  5 * time.Minute,
		stopCh:    make(chan struct{}),
	}
}

func (j *VideoCounterSyncJob) Start() {
	go j.run()
}

func (j *VideoCounterSyncJob) Stop() {
	close(j.stopCh)
}

func (j *VideoCounterSyncJob) run() {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	j.sync()
	for {
		select {
		case <-ticker.C:
			j.sync()
		case <-j.stopCh:
			return
		}
	}
}

func (j *VideoCounterSyncJob) sync() {
	ctx := context.Background()
	videoIDs, err := j.videoRepo.GetDirtyVideoIDs(ctx)
	if err != nil {
		j.log.Errorf("获取脏视频ID列表失败: %v", err)
		return
	}
	if len(videoIDs) == 0 {
		return
	}
	j.log.Infof("开始同步视频计数器，视频数量: %d", len(videoIDs))

	// 批量从Redis获取所有计数（增加 view_count）
	fields := []string{"like_count", "comment_count", "collection_count", "view_count"}
	countersMap, err := j.videoRepo.BatchGetVideoCounters(ctx, videoIDs, fields...)
	if err != nil {
		j.log.Errorf("批量获取视频计数器失败: %v", err)
		return
	}

	// 更新MySQL
	err = j.batchUpdateMySQL(ctx, countersMap)
	if err != nil {
		j.log.Errorf("批量更新MySQL失败: %v", err)
		return
	}

	// 清除脏标记
	for _, vid := range videoIDs {
		if err := j.videoRepo.ClearDirtyFlag(ctx, vid); err != nil {
			j.log.Warnf("清除视频 %d 脏标记失败: %v", vid, err)
		}
	}
	j.log.Infof("视频计数器同步完成，更新 %d 个视频", len(countersMap))
}

func (j *VideoCounterSyncJob) batchUpdateMySQL(ctx context.Context, countersMap map[int64]map[string]int64) error {
	if len(countersMap) == 0 {
		return nil
	}
	tx := j.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for vid, counters := range countersMap {
		updates := make(map[string]interface{})
		for field, val := range counters {
			updates[field] = val // 字段名与数据库列名一致
		}
		if len(updates) == 0 {
			continue
		}
		updates["updated_at"] = time.Now()
		if err := tx.Table("video").Where("id = ?", vid).Updates(updates).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
