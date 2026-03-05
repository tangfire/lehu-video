package biz

import (
	"context"
	"encoding/json"
	"lehu-video/app/videoCore/service/internal/conf"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

// FavoriteConsumer 处理点赞事件，持久化到数据库并更新作者获赞数
type FavoriteConsumer struct {
	reader       *kafka.Reader
	favoriteRepo FavoriteRepo
	userCounter  UserCounterRepo
	videoRepo    VideoRepo
	log          *log.Helper
	batchProc    *BatchProcessor[*FavoriteEvent]
	stopCh       chan struct{}
}

func NewFavoriteConsumer(
	conf *conf.Data,
	favoriteRepo FavoriteRepo,
	userCounter UserCounterRepo,
	videoRepo VideoRepo,
	logger log.Logger,
) *FavoriteConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  conf.Kafka.Brokers,
		Topic:    conf.Kafka.Topic.Favorite,
		GroupID:  "favorite-consumer",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	c := &FavoriteConsumer{
		reader:       reader,
		favoriteRepo: favoriteRepo,
		userCounter:  userCounter,
		videoRepo:    videoRepo,
		log:          log.NewHelper(logger),
		stopCh:       make(chan struct{}),
	}
	c.batchProc = NewBatchProcessor[*FavoriteEvent](
		500,
		2*time.Second,
		c.batchInsert,
		logger,
	)
	return c
}

func (c *FavoriteConsumer) Start() {
	go c.run()
}

func (c *FavoriteConsumer) Stop() {
	close(c.stopCh)
	c.reader.Close()
	c.batchProc.Stop()
}

func (c *FavoriteConsumer) run() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
			msg, err := c.reader.ReadMessage(context.Background())
			if err != nil {
				c.log.Errorf("读取Kafka消息失败: %v", err)
				time.Sleep(time.Second)
				continue
			}
			var event FavoriteEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				c.log.Errorf("解析消息失败: %v", err)
				continue
			}
			c.batchProc.Add(&event)
		}
	}
}

// batchInsert 批量插入/更新 favorite 表，并批量更新作者获赞数
func (c *FavoriteConsumer) batchInsert(events []*FavoriteEvent) error {
	if len(events) == 0 {
		return nil
	}
	ctx := context.Background()

	// 第一步：在MySQL事务中处理favorite表
	err := c.favoriteRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, e := range events {
			if e.Action == 1 { // 添加
				// 先尝试查找包含软删除的记录
				existing, err := c.favoriteRepo.GetFavoriteIncludeDeleted(txCtx, e.UserId, e.TargetId, e.TargetType)
				if err != nil {
					return err
				}
				if existing != nil {
					if existing.DeleteAt != 0 {
						// 软删除的记录，重新激活，同时更新点赞类型
						existing.DeleteAt = 0
						existing.FavoriteType = e.FavoriteType
						existing.UpdatedAt = time.Unix(e.Timestamp, 0)
						// 使用时间戳乐观更新
						if err := c.favoriteRepo.UpdateFavoriteIfNewer(txCtx, existing); err != nil {
							return err
						}
					} else {
						// 已存在且有效，但点赞类型可能已变更（例如从踩改为赞）
						if existing.FavoriteType != e.FavoriteType {
							existing.FavoriteType = e.FavoriteType
							existing.UpdatedAt = time.Unix(e.Timestamp, 0)
							if err := c.favoriteRepo.UpdateFavoriteIfNewer(txCtx, existing); err != nil {
								return err
							}
						}
						// 完全相同则忽略
					}
					continue
				}
				// 不存在，创建
				fav := &Favorite{
					Id:           0,
					UserId:       e.UserId,
					TargetType:   e.TargetType,
					TargetId:     e.TargetId,
					FavoriteType: e.FavoriteType,
					DeleteAt:     0,
					CreatedAt:    time.Unix(e.Timestamp, 0),
					UpdatedAt:    time.Unix(e.Timestamp, 0),
				}
				if err := c.favoriteRepo.CreateFavorite(txCtx, fav); err != nil {
					return err
				}
			} else { // 取消
				// 查找有效记录（不包含删除的）
				fav, err := c.favoriteRepo.GetFavoriteByUserTarget(txCtx, e.UserId, e.TargetId, e.TargetType)
				if err != nil {
					return err
				}
				if fav != nil && fav.DeleteAt == 0 {
					fav.DeleteAt = time.Now().Unix()
					fav.UpdatedAt = time.Unix(e.Timestamp, 0)
					if err := c.favoriteRepo.UpdateFavoriteIfNewer(txCtx, fav); err != nil {
						return err
					}
				}
				// 如果已经是软删除或不存在，忽略
			}
		}
		return nil
	})
	if err != nil {
		c.log.Errorf("批量插入失败: %v", err)
		return err
	}

	// 第二步：批量更新作者获赞数（仅针对视频点赞/取消点赞）
	videoIDSet := make(map[int64]struct{})
	for _, e := range events {
		if e.TargetType == 0 && e.FavoriteType == 0 {
			videoIDSet[e.TargetId] = struct{}{}
		}
	}
	if len(videoIDSet) == 0 {
		return nil
	}
	videoIDs := make([]int64, 0, len(videoIDSet))
	for vid := range videoIDSet {
		videoIDs = append(videoIDs, vid)
	}

	authorMap, err := c.videoRepo.BatchGetVideoAuthors(ctx, videoIDs)
	if err != nil {
		c.log.Errorf("批量获取视频作者失败: %v", err)
		return err
	}

	authorCounts := make(map[int64]map[string]int64)
	for _, e := range events {
		if e.TargetType == 0 && e.FavoriteType == 0 {
			authorID, ok := authorMap[e.TargetId]
			if !ok {
				c.log.Warnf("视频 %d 找不到作者，跳过更新获赞数", e.TargetId)
				continue
			}
			delta := int64(0)
			if e.Action == 1 {
				delta = 1
			} else if e.Action == -1 {
				delta = -1
			}
			if delta == 0 {
				continue
			}
			if _, ok := authorCounts[authorID]; !ok {
				authorCounts[authorID] = make(map[string]int64)
			}
			authorCounts[authorID]["be_liked_count"] += delta
		}
	}

	if len(authorCounts) > 0 {
		if err := c.userCounter.BatchIncrUserCounters(ctx, authorCounts); err != nil {
			c.log.Errorf("批量更新作者获赞数失败: %v", err)
		}
	}
	return nil
}
