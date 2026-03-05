// data/user_be_liked_reconcile_job.go
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// UserBeLikedReconcileJob 用户获赞数对账任务
type UserBeLikedReconcileJob struct {
	db        *gorm.DB
	log       *log.Helper
	batchSize int64
	sleepMs   int64
	sinceDays int // 只对账最近几天有更新的用户
}

func NewUserBeLikedReconcileJob(db *gorm.DB, logger log.Logger) *UserBeLikedReconcileJob {
	return &UserBeLikedReconcileJob{
		db:        db,
		log:       log.NewHelper(logger),
		batchSize: 1000,
		sleepMs:   100,
		sinceDays: 7,
	}
}

// RunOnce 执行一次对账
func (j *UserBeLikedReconcileJob) RunOnce(ctx context.Context) error {
	j.log.Info("开始用户获赞数对账...")
	start := time.Now()

	sinceTime := time.Now().AddDate(0, 0, -j.sinceDays).Format("2006-01-02 15:04:05")

	// 获取符合条件的用户最小/最大ID
	var minID, maxID int64
	sqlCount := `SELECT MIN(id), MAX(id) FROM user WHERE updated_at >= ?`
	if err := j.db.WithContext(ctx).Raw(sqlCount, sinceTime).Row().Scan(&minID, &maxID); err != nil {
		j.log.Errorf("获取用户ID范围失败: %v", err)
		return err
	}

	if minID == 0 && maxID == 0 {
		j.log.Infof("最近 %d 天内无更新用户，跳过对账", j.sinceDays)
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

	j.log.Infof("用户获赞数对账完成，更新 %d 条记录，耗时 %v", totalAffected, time.Since(start))
	return nil
}

// reconcileBatch 处理单个ID范围的对账
func (j *UserBeLikedReconcileJob) reconcileBatch(ctx context.Context, startID, endID int64, sinceTime string) (int64, error) {
	sql := `
		UPDATE user u
		INNER JOIN (
			SELECT v.user_id, SUM(v.like_count) AS total_be_liked
			FROM video v
			WHERE v.user_id BETWEEN ? AND ?
			GROUP BY v.user_id
		) AS t ON u.id = t.user_id
		SET u.be_liked_count = t.total_be_liked, u.updated_at = NOW()
		WHERE u.be_liked_count != t.total_be_liked
		  AND u.updated_at >= ?
	`
	result := j.db.WithContext(ctx).Exec(sql, startID, endID, sinceTime)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// StartCron 启动定时任务（每天凌晨4点执行，错开视频对账时间）
func (j *UserBeLikedReconcileJob) StartCron(stopCh <-chan struct{}) {
	go j.RunOnce(context.Background())

	go func() {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())
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
				j.log.Info("用户获赞数对账任务已停止")
				return
			}
		}
	}()
}
