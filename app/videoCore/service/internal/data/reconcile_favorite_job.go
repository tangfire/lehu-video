package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// FavoriteReconcileJob 视频点赞数对账任务（分批处理版）
type FavoriteReconcileJob struct {
	db        *gorm.DB
	log       *log.Helper
	batchSize int64 // 每批处理的视频数量
	sleepMs   int64 // 批次间休眠毫秒数
}

func NewFavoriteReconcileJob(db *gorm.DB, logger log.Logger) *FavoriteReconcileJob {
	return &FavoriteReconcileJob{
		db:        db,
		log:       log.NewHelper(logger),
		batchSize: 1000, // 每批 1000 个视频
		sleepMs:   100,  // 每批后休眠 100ms
	}
}

// RunOnce 执行一次对账，按 ID 范围分批处理
func (j *FavoriteReconcileJob) RunOnce(ctx context.Context) error {
	j.log.Info("开始视频点赞数对账（分批模式）...")
	start := time.Now()

	// 获取当前视频表的最小/最大 ID
	var minID, maxID int64
	if err := j.db.WithContext(ctx).Raw("SELECT MIN(id), MAX(id) FROM video").Row().Scan(&minID, &maxID); err != nil {
		j.log.Errorf("获取视频ID范围失败: %v", err)
		return err
	}

	if minID == 0 && maxID == 0 {
		j.log.Info("视频表为空，跳过对账")
		return nil
	}

	totalAffected := int64(0)
	// 按 batchSize 步进处理
	for startID := minID; startID <= maxID; startID += j.batchSize {
		endID := startID + j.batchSize - 1
		if endID > maxID {
			endID = maxID
		}

		// 执行当前批次的更新
		affected, err := j.reconcileBatch(ctx, startID, endID)
		if err != nil {
			j.log.Errorf("批次 [%d, %d] 对账失败: %v", startID, endID, err)
			// 根据策略，可以选择继续或中断
			continue
		}
		totalAffected += affected

		// 批次间休眠，让出数据库资源
		if j.sleepMs > 0 && endID < maxID {
			time.Sleep(time.Duration(j.sleepMs) * time.Millisecond)
		}
	}

	j.log.Infof("对账完成，总计更新 %d 条视频记录，耗时 %v", totalAffected, time.Since(start))
	return nil
}

// reconcileBatch 处理单个 ID 范围的对账
func (j *FavoriteReconcileJob) reconcileBatch(ctx context.Context, startID, endID int64) (int64, error) {
	sql := `
		UPDATE video v
		INNER JOIN (
			SELECT target_id, COUNT(*) AS real_likes
			FROM favorite
			WHERE target_id BETWEEN ? AND ?
			  AND target_type = 0
			  AND favorite_type = 0
			  AND delete_at = 0
			GROUP BY target_id
		) AS t ON v.id = t.target_id
		SET v.like_count = t.real_likes
		WHERE v.like_count != t.real_likes
	`

	result := j.db.WithContext(ctx).Exec(sql, startID, endID)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// StartCron 启动定时任务（每天凌晨3点执行）
func (j *FavoriteReconcileJob) StartCron(stopCh <-chan struct{}) {
	// 先立即执行一次（可选）
	go j.RunOnce(context.Background())

	go func() {
		// 计算首次执行时间：下一个凌晨3点
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
				j.log.Info("对账任务已停止")
				return
			}
		}
	}()
}
