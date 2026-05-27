// data/user_be_liked_reconcile_job.go
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

// UserBeLikedReconcileJob 用户统计对账任务
type UserBeLikedReconcileJob struct {
	db           *gorm.DB
	log          *log.Helper
	batchSize    int64
	sleepMs      int64
	sinceDays    int              // 只对账最近几天视频有更新的用户
	redisLock    *redsync.Redsync // 分布式锁
	maxRetries   int              // 最大重试次数
	batchTimeout time.Duration    // 单个批次超时时间
}

// NewUserBeLikedReconcileJob 创建用户获赞数对账任务（需外部注入 redisLock）
func NewUserBeLikedReconcileJob(db *gorm.DB, logger log.Logger) *UserBeLikedReconcileJob {
	return &UserBeLikedReconcileJob{
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
func (j *UserBeLikedReconcileJob) SetRedisLock(rs *redsync.Redsync) {
	j.redisLock = rs
}

// RunOnce 执行一次对账
func (j *UserBeLikedReconcileJob) RunOnce(ctx context.Context) error {
	// 尝试获取分布式锁，防止多实例重复执行
	if j.redisLock != nil {
		mu := j.redisLock.NewMutex("reconcile:user_be_liked")
		if err := mu.Lock(); err != nil {
			j.log.Warnf("获取用户统计对账锁失败，可能有其他实例在执行：%v", err)
			return nil // 跳过本次执行
		}
		defer mu.Unlock()
	}

	j.log.Info("开始用户统计对账...")
	start := time.Now()

	sinceTime := time.Now().AddDate(0, 0, -j.sinceDays).Format("2006-01-02 15:04:05")

	totalAffected := int64(0)
	totalBatches := 0
	failedBatches := make([][]int64, 0)

	lastID := int64(0)
	for {
		ids, err := j.listChangedUserIDs(ctx, sinceTime, lastID, int(j.batchSize))
		if err != nil {
			j.log.Errorf("获取变化用户 ID 失败：%v", err)
			return err
		}
		if len(ids) == 0 {
			break
		}
		lastID = ids[len(ids)-1]
		totalBatches++

		affected, err := j.reconcileBatch(ctx, ids)
		if err != nil {
			j.log.Errorf("用户统计批次 [%d, %d] 对账失败：%v", ids[0], ids[len(ids)-1], err)
			failedBatches = append(failedBatches, append([]int64(nil), ids...))
			continue
		}
		totalAffected += affected

		if j.sleepMs > 0 && len(ids) == int(j.batchSize) {
			time.Sleep(time.Duration(j.sleepMs) * time.Millisecond)
		}
	}

	if totalBatches == 0 {
		j.log.Infof("最近 %d 天内无用户统计变化，跳过对账", j.sinceDays)
		return nil
	}

	// 重试失败的批次
	if len(failedBatches) > 0 {
		j.log.Warnf("开始重试 %d 个失败的批次...", len(failedBatches))
		for retry := 0; retry < j.maxRetries && len(failedBatches) > 0; retry++ {
			remaining := make([][]int64, 0)
			for _, ids := range failedBatches {
				affected, err := j.reconcileBatch(ctx, ids)
				if err != nil {
					remaining = append(remaining, ids)
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
			for _, ids := range failedBatches {
				j.log.Errorf("  - 批次 [%d, %d]", ids[0], ids[len(ids)-1])
			}
			// 此处可添加告警通知
		}
	}

	j.log.Infof("用户统计对账完成，处理 %d 个批次，更新 %d 条记录，耗时 %v", totalBatches, totalAffected, time.Since(start))
	return nil
}

func (j *UserBeLikedReconcileJob) listChangedUserIDs(ctx context.Context, sinceTime string, lastID int64, limit int) ([]int64, error) {
	sql := `
		SELECT user_id
		FROM (
			SELECT user_id FROM video WHERE updated_at >= ?
			UNION
			SELECT v.user_id
			FROM favorite f
			INNER JOIN video v ON v.id = f.target_id
			WHERE f.updated_at >= ?
			  AND f.target_type = 0
			  AND f.favorite_type = 0
			UNION
			SELECT user_id FROM follow WHERE updated_at >= ?
			UNION
			SELECT target_user_id AS user_id FROM follow WHERE updated_at >= ?
			UNION
			SELECT user_id FROM collection_video WHERE updated_at >= ?
		) changed_users
		WHERE user_id > ?
		ORDER BY user_id ASC
		LIMIT ?
	`
	ids := make([]int64, 0, limit)
	if err := j.db.WithContext(ctx).Raw(sql, sinceTime, sinceTime, sinceTime, sinceTime, sinceTime, lastID, limit).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// reconcileBatch 处理一批统计发生变化的用户。
func (j *UserBeLikedReconcileJob) reconcileBatch(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, j.batchTimeout)
	defer cancel()

	sql := `
		UPDATE user u
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS work_count, COALESCE(SUM(like_count), 0) AS total_be_liked
			FROM video
			WHERE user_id IN ?
			GROUP BY user_id
		) AS v ON u.id = v.user_id
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS follow_count
			FROM follow
			WHERE user_id IN ?
			  AND is_deleted = 0
			GROUP BY user_id
		) AS fg ON u.id = fg.user_id
		LEFT JOIN (
			SELECT target_user_id AS user_id, COUNT(*) AS follower_count
			FROM follow
			WHERE target_user_id IN ?
			  AND is_deleted = 0
			GROUP BY target_user_id
		) AS fr ON u.id = fr.user_id
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS collection_count
			FROM collection_video
			WHERE user_id IN ?
			  AND is_deleted = 0
			GROUP BY user_id
		) AS cv ON u.id = cv.user_id
		SET
			u.be_liked_count = COALESCE(v.total_be_liked, 0),
			u.work_count = COALESCE(v.work_count, 0),
			u.follow_count = COALESCE(fg.follow_count, 0),
			u.follower_count = COALESCE(fr.follower_count, 0),
			u.collection_count = COALESCE(cv.collection_count, 0),
			u.updated_at = NOW()
		WHERE u.id IN ?
		  AND (
			u.be_liked_count != COALESCE(v.total_be_liked, 0)
			OR u.work_count != COALESCE(v.work_count, 0)
			OR u.follow_count != COALESCE(fg.follow_count, 0)
			OR u.follower_count != COALESCE(fr.follower_count, 0)
			OR u.collection_count != COALESCE(cv.collection_count, 0)
		  )
	`
	result := j.db.WithContext(ctx).Exec(sql, ids, ids, ids, ids, ids)
	if result.Error != nil {
		if errors.Is(result.Error, context.DeadlineExceeded) {
			return 0, fmt.Errorf("批次 [%d, %d] 执行超时 (%v)", ids[0], ids[len(ids)-1], j.batchTimeout)
		}
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// StartCron 启动定时任务（每天凌晨4点执行，错开视频对账时间）
func (j *UserBeLikedReconcileJob) StartCron(stopCh <-chan struct{}) {
	// 立即执行一次（可选）
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
				j.log.Info("用户统计对账任务已停止")
				return
			}
		}
	}()
}
