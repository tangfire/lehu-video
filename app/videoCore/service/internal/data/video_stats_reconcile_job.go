// data/video_stats_reconcile_job.go
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// VideoStatsReconcileJob 视频点赞数对账任务（只更新点赞数）
type VideoStatsReconcileJob struct {
	db        *gorm.DB
	log       *log.Helper
	batchSize int64
	sleepMs   int64
	sinceDays int // 只对账最近几天有更新的视频
}

func NewVideoStatsReconcileJob(db *gorm.DB, logger log.Logger) *VideoStatsReconcileJob {
	return &VideoStatsReconcileJob{
		db:        db,
		log:       log.NewHelper(logger),
		batchSize: 1000,
		sleepMs:   100,
		sinceDays: 7,
	}
}

// RunOnce 执行一次对账
func (j *VideoStatsReconcileJob) RunOnce(ctx context.Context) error {
	j.log.Info("开始视频点赞数对账...")
	start := time.Now()

	sinceTime := time.Now().AddDate(0, 0, -j.sinceDays).Format("2006-01-02 15:04:05")

	// 获取符合条件的视频最小/最大ID
	var minID, maxID int64
	sqlCount := `SELECT MIN(id), MAX(id) FROM video WHERE updated_at >= ?`
	if err := j.db.WithContext(ctx).Raw(sqlCount, sinceTime).Row().Scan(&minID, &maxID); err != nil {
		j.log.Errorf("获取视频ID范围失败: %v", err)
		return err
	}

	if minID == 0 && maxID == 0 {
		j.log.Infof("最近 %d 天内无更新视频，跳过对账", j.sinceDays)
		return nil
	}

	totalAffected := int64(0)
	for startID := minID; startID <= maxID; startID += j.batchSize {
		endID := startID + j.batchSize - 1
		if endID > maxID {
			endID = maxID
		}

		affected, err := j.reconcileBatch(ctx, startID, endID, sinceTime)
		if err != nil {
			j.log.Errorf("批次 [%d, %d] 对账失败: %v", startID, endID, err)
			continue
		}
		totalAffected += affected

		if j.sleepMs > 0 && endID < maxID {
			time.Sleep(time.Duration(j.sleepMs) * time.Millisecond)
		}
	}

	j.log.Infof("视频点赞数对账完成，更新 %d 条记录，耗时 %v", totalAffected, time.Since(start))
	return nil
}

// reconcileBatch 处理单个ID范围的对账，只更新点赞数
func (j *VideoStatsReconcileJob) reconcileBatch(ctx context.Context, startID, endID int64, sinceTime string) (int64, error) {
	sql := `
		UPDATE video v
		INNER JOIN (
			SELECT 
				target_id,
				COUNT(*) AS real_likes
			FROM favorite
			WHERE target_id BETWEEN ? AND ?
			  AND target_type = 0
			  AND favorite_type = 0
			  AND delete_at = 0
			GROUP BY target_id
		) AS t ON v.id = t.target_id
		SET 
			v.like_count = t.real_likes,
			v.updated_at = NOW()
		WHERE v.like_count != t.real_likes
		  AND v.updated_at >= ?
	`
	result := j.db.WithContext(ctx).Exec(sql, startID, endID, sinceTime)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// StartCron 启动定时任务（每天凌晨3点执行）
func (j *VideoStatsReconcileJob) StartCron(stopCh <-chan struct{}) {
	// 先立即执行一次（可选）
	go j.RunOnce(context.Background())

	go func() {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		timer := time.NewTimer(next.Sub(now))
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				j.RunOnce(context.Background())
				timer.Reset(24 * time.Hour)
			case <-stopCh:
				j.log.Info("视频点赞数对账任务已停止")
				return
			}
		}
	}()
}
