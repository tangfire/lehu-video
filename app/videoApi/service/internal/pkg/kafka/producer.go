package kafka

import (
	"context"
	"lehu-video/app/videoApi/service/internal/conf"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(conf *conf.Data) *Producer {
	return NewProducer(conf.Kafka.Brokers, conf.Kafka.ProducerTopic)
}

func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
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
		Compression:            kafka.Snappy,
		AllowAutoTopicCreation: true,
	}
	return &Producer{writer: w}
}

func (p *Producer) SendMessage(ctx context.Context, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
