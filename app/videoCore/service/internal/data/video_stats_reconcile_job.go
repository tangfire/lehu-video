// data/video_stats_reconcile_job.go
package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redsync/redsync/v4"
	"gorm.io/gorm"
)

// VideoStatsReconcileJob 视频点赞数对账任务
type VideoStatsReconcileJob struct {
	db           *gorm.DB
	log          *log.Helper
	batchSize    int64
	sleepMs      int64
	sinceDays    int              // 只对账最近几天有更新的视频
	redisLock    *redsync.Redsync // 分布式锁
	maxRetries   int              // 最大重试次数
	batchTimeout time.Duration    // 单个批次超时时间
}

// NewVideoStatsReconcileJob 创建视频点赞数对账任务（需外部注入 redisLock）
func NewVideoStatsReconcileJob(db *gorm.DB, logger log.Logger) *VideoStatsReconcileJob {
	return &VideoStatsReconcileJob{
		db:           db,
		log:          log.NewHelper(logger),
		batchSize:    1000,
		sleepMs:      100,
		sinceDays:    7,
		maxRetries:   3,
		batchTimeout: 30 * time.Second,
		// redisLock 需要在外部初始化后通过 SetRedisLock 或直接赋值
	}
}

// SetRedisLock 设置分布式锁（可选，若不设置则不会尝试加锁）
func (j *VideoStatsReconcileJob) SetRedisLock(rs *redsync.Redsync) {
	j.redisLock = rs
}

// RunOnce 执行一次对账
func (j *VideoStatsReconcileJob) RunOnce(ctx context.Context) error {
	// 尝试获取分布式锁，防止多实例重复执行
	if j.redisLock != nil {
		mu := j.redisLock.NewMutex("reconcile:video_stats")
		if err := mu.Lock(); err != nil {
			j.log.Warnf("获取锁失败，可能有其他实例在执行：%v", err)
			return nil // 跳过本次执行
		}
		defer mu.Unlock()
	}

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
	failedBatches := make([][2]int64, 0)

	for startID := minID; startID <= maxID; startID += j.batchSize {
		endID := startID + j.batchSize - 1
		if endID > maxID {
			endID = maxID
		}

		affected, err := j.reconcileBatch(ctx, startID, endID, sinceTime)
		if err != nil {
			j.log.Errorf("批次 [%d, %d] 对账失败: %v", startID, endID, err)
			failedBatches = append(failedBatches, [2]int64{startID, endID})
			continue
		}
		totalAffected += affected

		if j.sleepMs > 0 && endID < maxID {
			time.Sleep(time.Duration(j.sleepMs) * time.Millisecond)
		}
	}

	// 重试失败的批次
	if len(failedBatches) > 0 {
		j.log.Warnf("开始重试 %d 个失败的批次...", len(failedBatches))
		for retry := 0; retry < j.maxRetries && len(failedBatches) > 0; retry++ {
			remaining := make([][2]int64, 0)
			for _, batch := range failedBatches {
				startID, endID := batch[0], batch[1]
				affected, err := j.reconcileBatch(ctx, startID, endID, sinceTime)
				if err != nil {
					remaining = append(remaining, batch)
				} else {
					totalAffected += affected
				}
			}
			failedBatches = remaining
			if len(failedBatches) > 0 {
				j.log.Warnf("第 %d 次重试后仍有 %d 个批次失败，5 秒后再次重试", retry+1, len(failedBatches))
				time.Sleep(5 * time.Second)
			}
		}

		if len(failedBatches) > 0 {
			j.log.Errorf("最终仍有 %d 个批次对账失败，请手动检查：", len(failedBatches))
			for _, batch := range failedBatches {
				j.log.Errorf("  - 批次 [%d, %d]", batch[0], batch[1])
			}
			// 此处可添加告警通知
		}
	}

	j.log.Infof("视频点赞数对账完成，更新 %d 条记录，耗时 %v", totalAffected, time.Since(start))
	return nil
}

// reconcileBatch 处理单个ID范围的对账
func (j *VideoStatsReconcileJob) reconcileBatch(ctx context.Context, startID, endID int64, sinceTime string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, j.batchTimeout)
	defer cancel()

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
		if errors.Is(result.Error, context.DeadlineExceeded) {
			return 0, fmt.Errorf("批次 [%d, %d] 执行超时 (%v)", startID, endID, j.batchTimeout)
		}
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// StartCron 启动定时任务（每天凌晨3点执行）
func (j *VideoStatsReconcileJob) StartCron(stopCh <-chan struct{}) {
	// 立即执行一次（可选）
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
