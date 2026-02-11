// biz/kafka_producer.go - Kafka生产者实现
package biz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducerImpl Kafka生产者实现
type KafkaProducerImpl struct {
	writers map[string]*kafka.Writer
}

func NewKafkaProducerImpl() *KafkaProducerImpl {
	return &KafkaProducerImpl{
		writers: make(map[string]*kafka.Writer),
	}
}

// SendMessage 发送消息
func (p *KafkaProducerImpl) SendMessage(topic string, key, value []byte) error {
	writer, exists := p.writers[topic]
	if !exists {
		writer = &kafka.Writer{
			Addr:                   kafka.TCP("localhost:9092"), // 从配置读取
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			MaxAttempts:            3,
			WriteBackoffMin:        time.Millisecond * 10,
			WriteBackoffMax:        time.Millisecond * 50,
			BatchSize:              100,
			BatchBytes:             1048576, // 1MB
			BatchTimeout:           time.Millisecond * 100,
			ReadTimeout:            time.Second * 10,
			WriteTimeout:           time.Second * 10,
			RequiredAcks:           kafka.RequireAll,
			Async:                  false,
			Completion:             nil,
			Compression:            kafka.Snappy,
			Logger:                 nil,
			ErrorLogger:            nil,
			AllowAutoTopicCreation: true,
		}
		p.writers[topic] = writer
	}

	return writer.WriteMessages(context.Background(), kafka.Message{
		Key:   key,
		Value: value,
	})
}

// MockKafkaProducer 用于测试的Mock生产者
type MockKafkaProducer struct{}

func (m *MockKafkaProducer) SendMessage(topic string, key, value []byte) error {
	// 在测试环境下，只打印日志
	// log.Infof("Kafka Mock - Topic: %s, Key: %s, Value: %s", topic, string(key), string(value))
	return nil
}

// VideoPublishMessage 视频发布消息结构
type VideoPublishMessage struct {
	VideoID   string   `json:"video_id"`
	AuthorID  string   `json:"author_id"`
	Timestamp int64    `json:"timestamp"`
	Followers []string `json:"followers,omitempty"`
	EventType string   `json:"event_type"`
}

// CreateVideoPublishMessage 创建视频发布消息
func CreateVideoPublishMessage(videoID, authorID string, timestamp int64, followers []string) []byte {
	msg := VideoPublishMessage{
		VideoID:   videoID,
		AuthorID:  authorID,
		Timestamp: timestamp,
		Followers: followers,
		EventType: "video_published",
	}

	data, _ := json.Marshal(msg)
	return data
}
