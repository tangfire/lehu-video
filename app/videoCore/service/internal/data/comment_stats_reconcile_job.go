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

// CommentStatsReconcileJob 对账评论点赞数和直接回复数。
type CommentStatsReconcileJob struct {
	db           *gorm.DB
	log          *log.Helper
	batchSize    int64
	sleepMs      int64
	sinceDays    int
	redisLock    *redsync.Redsync
	maxRetries   int
	batchTimeout time.Duration
}

func NewCommentStatsReconcileJob(db *gorm.DB, logger log.Logger) *CommentStatsReconcileJob {
	return &CommentStatsReconcileJob{
		db:           db,
		log:          log.NewHelper(logger),
		batchSize:    1000,
		sleepMs:      100,
		sinceDays:    7,
		maxRetries:   3,
		batchTimeout: 30 * time.Second,
	}
}

func (j *CommentStatsReconcileJob) SetRedisLock(rs *redsync.Redsync) {
	j.redisLock = rs
}

func (j *CommentStatsReconcileJob) RunOnce(ctx context.Context) error {
	if j.redisLock != nil {
		mu := j.redisLock.NewMutex("reconcile:comment_stats")
		if err := mu.Lock(); err != nil {
			j.log.Warnf("获取评论统计对账锁失败，可能有其他实例在执行：%v", err)
			return nil
		}
		defer mu.Unlock()
	}

	j.log.Info("开始评论统计对账...")
	start := time.Now()
	sinceTime := time.Now().AddDate(0, 0, -j.sinceDays).Format("2006-01-02 15:04:05")

	totalAffected := int64(0)
	totalBatches := 0
	failedBatches := make([][]int64, 0)
	lastID := int64(0)
	for {
		ids, err := j.listChangedCommentIDs(ctx, sinceTime, lastID, int(j.batchSize))
		if err != nil {
			j.log.Errorf("获取变化评论 ID 失败：%v", err)
			return err
		}
		if len(ids) == 0 {
			break
		}
		lastID = ids[len(ids)-1]
		totalBatches++

		affected, err := j.reconcileBatch(ctx, ids)
		if err != nil {
			j.log.Errorf("评论统计批次 [%d, %d] 对账失败：%v", ids[0], ids[len(ids)-1], err)
			failedBatches = append(failedBatches, append([]int64(nil), ids...))
			continue
		}
		totalAffected += affected

		if j.sleepMs > 0 && len(ids) == int(j.batchSize) {
			time.Sleep(time.Duration(j.sleepMs) * time.Millisecond)
		}
	}
	if totalBatches == 0 {
		j.log.Infof("最近 %d 天内无评论统计变化，跳过对账", j.sinceDays)
		return nil
	}

	for retry := 0; retry < j.maxRetries && len(failedBatches) > 0; retry++ {
		remaining := make([][]int64, 0)
		for _, ids := range failedBatches {
			affected, err := j.reconcileBatch(ctx, ids)
			if err != nil {
				remaining = append(remaining, ids)
				continue
			}
			totalAffected += affected
		}
		failedBatches = remaining
		if len(failedBatches) > 0 {
			j.log.Warnf("第 %d 次重试后仍有 %d 个评论统计批次失败，5 秒后再次重试", retry+1, len(failedBatches))
			time.Sleep(5 * time.Second)
		}
	}

	if len(failedBatches) > 0 {
		j.log.Errorf("最终仍有 %d 个评论统计批次对账失败", len(failedBatches))
	}

	j.log.Infof("评论统计对账完成，处理 %d 个批次，更新 %d 条记录，耗时 %v", totalBatches, totalAffected, time.Since(start))
	return nil
}

func (j *CommentStatsReconcileJob) listChangedCommentIDs(ctx context.Context, sinceTime string, lastID int64, limit int) ([]int64, error) {
	sql := `
		SELECT comment_id
		FROM (
			SELECT id AS comment_id FROM comment WHERE updated_at >= ?
			UNION
			SELECT parent_id AS comment_id FROM comment WHERE updated_at >= ? AND parent_id > 0
			UNION
			SELECT target_id AS comment_id FROM favorite
			WHERE updated_at >= ?
			  AND target_type = 1
			  AND favorite_type = 0
		) changed_comments
		WHERE comment_id > ?
		ORDER BY comment_id ASC
		LIMIT ?
	`
	ids := make([]int64, 0, limit)
	if err := j.db.WithContext(ctx).Raw(sql, sinceTime, sinceTime, sinceTime, lastID, limit).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (j *CommentStatsReconcileJob) reconcileBatch(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, j.batchTimeout)
	defer cancel()

	sql := `
		UPDATE comment c
		LEFT JOIN (
			SELECT target_id, COUNT(*) AS real_likes
			FROM favorite
			WHERE target_id IN ?
			  AND target_type = 1
			  AND favorite_type = 0
			  AND delete_at = 0
			GROUP BY target_id
		) AS fav ON c.id = fav.target_id
		LEFT JOIN (
			SELECT parent_id, COUNT(*) AS real_replies
			FROM comment
			WHERE parent_id IN ?
			  AND is_deleted = 0
			GROUP BY parent_id
		) AS reply ON c.id = reply.parent_id
		SET
			c.like_count = COALESCE(fav.real_likes, 0),
			c.reply_count = COALESCE(reply.real_replies, 0),
			c.updated_at = NOW()
		WHERE c.id IN ?
		  AND (
			c.like_count != COALESCE(fav.real_likes, 0)
			OR c.reply_count != COALESCE(reply.real_replies, 0)
		  )
	`
	result := j.db.WithContext(ctx).Exec(sql, ids, ids, ids)
	if result.Error != nil {
		if errors.Is(result.Error, context.DeadlineExceeded) {
			return 0, fmt.Errorf("批次 [%d, %d] 执行超时 (%v)", ids[0], ids[len(ids)-1], j.batchTimeout)
		}
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (j *CommentStatsReconcileJob) StartCron(stopCh <-chan struct{}) {
	go j.RunOnce(context.Background())

	go func() {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, now.Location())
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
				j.log.Info("评论统计对账任务已停止")
				return
			}
		}
	}()
}
