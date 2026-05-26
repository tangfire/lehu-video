// biz/video_producer.go - Kafka生产者实现
package biz

import (
	"context"
	"lehu-video/app/videoCore/service/internal/conf"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer interface {
	SendMessage(topic string, key, value []byte) error
}

// KafkaProducerImpl Kafka生产者实现
type KafkaProducerImpl struct {
	writers map[string]*kafka.Writer
	brokers []string
}

func NewVideoProducer(c *conf.Data) KafkaProducer {
	return &KafkaProducerImpl{
		writers: make(map[string]*kafka.Writer),
		brokers: c.Kafka.Brokers,
	}
}

// SendMessage 发送消息
func (p *KafkaProducerImpl) SendMessage(topic string, key, value []byte) error {
	writer, exists := p.writers[topic]
	if !exists {
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(p.brokers...),
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
