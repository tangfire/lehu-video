// biz/kafka_consumer.go - 修复版
package biz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
)

type VideoPublishEvent struct {
	VideoID   string `json:"video_id"`
	AuthorID  string `json:"author_id"`
	Timestamp int64  `json:"timestamp"`
}

type KafkaConsumer struct {
	reader      *kafka.Reader
	feedUsecase *FeedUsecase
	log         *log.Helper
}

func NewKafkaConsumer(brokers []string, topic string, feedUsecase *FeedUsecase, logger log.Logger) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  "feed_service",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &KafkaConsumer{
		reader:      reader,
		feedUsecase: feedUsecase,
		log:         log.NewHelper(logger),
	}
}

func (c *KafkaConsumer) Run(ctx context.Context) {
	c.log.Infof("Kafka消费者启动，topic: %s", c.reader.Config().Topic)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			c.log.Errorf("读取Kafka消息失败: %v", err)
			time.Sleep(time.Second)
			continue
		}
		c.processMessage(ctx, msg)
	}
}

func (c *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) {
	var event VideoPublishEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.log.Errorf("解析消息失败: %v", err)
		return
	}
	c.log.Infof("处理视频发布事件: video_id=%s, author_id=%s", event.VideoID, event.AuthorID)
	// 分页获取粉丝并推送
	c.pushToAllFollowers(ctx, event)
}

// pushToAllFollowers 分页获取大V的所有粉丝并推送
func (c *KafkaConsumer) pushToAllFollowers(ctx context.Context, event VideoPublishEvent) {
	item := &TimelineItem{
		VideoID:   event.VideoID,
		AuthorID:  event.AuthorID,
		Timestamp: event.Timestamp,
	}
	batchSize := c.feedUsecase.strategy.FollowerBatchSize
	offset := 0
	for {
		followers, total, err := c.feedUsecase.followRepo.GetFollowersPaginated(ctx, event.AuthorID, offset, batchSize)
		if err != nil {
			c.log.Errorf("分页获取粉丝失败: %v", err)
			return
		}
		for _, followerID := range followers {
			_ = c.feedUsecase.PushToUserTimeline(ctx, followerID, []*TimelineItem{item})
		}
		if int64(len(followers)+offset) >= total {
			break
		}
		offset += batchSize
	}
}
