package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"

	"lehu-video/app/videoCore/service/internal/biz"
	"lehu-video/app/videoCore/service/internal/conf"
)

// LikeConsumer 点赞事件消费者
type LikeConsumer struct {
	reader         *kafka.Reader
	favoriteUC     *biz.FavoriteUsecase
	log            *log.Helper
	batchProcessor *biz.BatchProcessor[*biz.LikeEvent]
}

// NewLikeConsumer 创建点赞消费者
func NewLikeConsumer(conf *conf.Data, favoriteUC *biz.FavoriteUsecase, logger log.Logger) *LikeConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  conf.Kafka.Brokers,
		Topic:    conf.Kafka.Topic.Like, // 需要你在配置中定义 Like 主题
		GroupID:  conf.Kafka.GroupId + "-like",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	c := &LikeConsumer{
		reader:     reader,
		favoriteUC: favoriteUC,
		log:        log.NewHelper(logger),
	}

	// 初始化批量处理器：每500条或每2秒刷新一次
	c.batchProcessor = biz.NewBatchProcessor[*biz.LikeEvent](
		500,
		2*time.Second,
		c.batchProcess,
		logger,
	)

	return c
}

// Run 启动消费者
func (c *LikeConsumer) Run(ctx context.Context) error {
	c.log.Info("LikeConsumer started")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				c.log.Errorf("read message error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			c.processMessage(ctx, msg)
		}
	}
}

// processMessage 处理单条消息
func (c *LikeConsumer) processMessage(ctx context.Context, msg kafka.Message) {
	var event biz.LikeEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.log.Errorf("unmarshal like event error: %v", err)
		c.reader.CommitMessages(ctx, msg)
		return
	}
	// 加入批量处理器
	c.batchProcessor.Add(&event)
	// 提交 offset（批量处理器异步落库，此处提交避免重复消费）
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		c.log.Errorf("commit offset error: %v", err)
	}
}

// batchProcess 批量处理（由 BatchProcessor 回调）
func (c *LikeConsumer) batchProcess(events []*biz.LikeEvent) error {
	if len(events) == 0 {
		return nil
	}

	// 合并相同 (user_id, target_id, target_type) 的事件，保留最新时间戳
	merged := make(map[string]*biz.LikeEvent)
	for _, e := range events {
		key := fmt.Sprintf("%d:%d:%d", e.UserID, e.TargetID, e.TargetType)
		if old, ok := merged[key]; ok {
			if e.Timestamp > old.Timestamp {
				merged[key] = e
			}
		} else {
			merged[key] = e
		}
	}

	// 调用业务层批量处理
	return c.favoriteUC.BatchProcessLikes(context.Background(), merged)
}

// Close 关闭消费者
func (c *LikeConsumer) Close() error {
	if c.batchProcessor != nil {
		c.batchProcessor.Stop()
	}
	return c.reader.Close()
}
