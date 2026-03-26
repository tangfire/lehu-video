// data/sync_comment_counters.go
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"lehu-video/app/videoCore/service/internal/biz"
)

// CommentCounterSyncJob 评论计数器同步任务（从 Redis 同步到 MySQL）
type CommentCounterSyncJob struct {
	db          *gorm.DB
	commentRepo biz.CommentRepo
	counterRepo biz.CommentCounterRepo
	log         *log.Helper
	batchSize   int64
	sleepMs     int64
	sinceDays   int // 只对账最近几天有更新的评论
	maxRetries  int
}

func NewCommentCounterSyncJob(db *gorm.DB, commentRepo biz.CommentRepo, counterRepo biz.CommentCounterRepo, logger log.Logger) *CommentCounterSyncJob {
	return &CommentCounterSyncJob{
		db:          db,
		commentRepo: commentRepo,
		counterRepo: counterRepo,
		log:         log.NewHelper(logger),
		batchSize:   1000,
		sleepMs:     100,
		sinceDays:   7,
		maxRetries:  3,
	}
}

// RunOnce 执行一次对账
func (j *CommentCounterSyncJob) RunOnce(ctx context.Context) error {
	// 获取所有脏评论 ID
	commentIDs, err := j.counterRepo.GetDirtyCommentIDs(ctx)
	if err != nil {
		j.log.Errorf("获取脏评论 ID 失败：%v", err)
		return err
	}

	if len(commentIDs) == 0 {
		return nil
	}

	j.log.Infof("开始同步评论计数器，共 %d 个评论", len(commentIDs))

	// 分批处理
	for i := 0; i < len(commentIDs); i += int(j.batchSize) {
		end := i + int(j.batchSize)
		if end > len(commentIDs) {
			end = len(commentIDs)
		}
		batch := commentIDs[i:end]

		// 批量从 Redis 获取计数
		fields := []string{"like_count"}
		countersMap, err := j.counterRepo.BatchGetCommentCounters(ctx, batch, fields...)
		if err != nil {
			j.log.Errorf("批量获取评论计数器失败：%v", err)
			continue
		}

		// 更新 MySQL
		err = j.batchUpdateMySQL(ctx, countersMap)
		if err != nil {
			j.log.Errorf("批量更新 MySQL 失败：%v", err)
			continue
		}

		// 清除脏标记
		for _, cid := range batch {
			if err := j.counterRepo.ClearDirtyFlag(ctx, cid); err != nil {
				j.log.Warnf("清除评论 %d 脏标记失败：%v", cid, err)
			}
		}

		// 稍微休眠，避免压力过大
		if j.sleepMs > 0 {
			time.Sleep(time.Duration(j.sleepMs) * time.Millisecond)
		}
	}

	j.log.Infof("评论计数器同步完成")
	return nil
}

func (j *CommentCounterSyncJob) batchUpdateMySQL(ctx context.Context, countersMap map[int64]map[string]int64) error {
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

	for cid, counters := range countersMap {
		updates := make(map[string]interface{})
		for field, val := range counters {
			updates[field] = val
		}
		if len(updates) == 0 {
			continue
		}
		updates["updated_at"] = time.Now()

		if err := tx.Table("comment").Where("id = ?", cid).Updates(updates).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// StartCron 启动定时任务（每 5 分钟执行一次）
func (j *CommentCounterSyncJob) StartCron(stopCh <-chan struct{}) {
	// 立即执行一次
	go j.RunOnce(context.Background())

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				j.RunOnce(context.Background())
			case <-stopCh:
				j.log.Info("评论计数器同步任务已停止")
				return
			}
		}
	}()
}
