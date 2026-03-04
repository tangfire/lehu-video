// server/video_consumer.go
package server

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type VideoKafkaConsumerServer struct {
	consumer *biz.KafkaConsumer
	log      *log.Helper
}

func NewVideoKafkaConsumerServer(consumer *biz.KafkaConsumer, logger log.Logger) *VideoKafkaConsumerServer {
	return &VideoKafkaConsumerServer{
		consumer: consumer,
		log:      log.NewHelper(logger),
	}
}

// Start 启动消费者服务，实现 kratos.Service 接口
func (s *VideoKafkaConsumerServer) Start(ctx context.Context) error {
	s.log.Info("启动 Kafka 消费者服务")
	go func() {
		if err := s.consumer.Run(ctx); err != nil {
			s.log.Errorf("Kafka 消费者运行错误: %v", err)
		}
	}()
	return nil
}

// Stop 停止消费者服务，实现 kratos.Service 接口
func (s *VideoKafkaConsumerServer) Stop(ctx context.Context) error {
	s.log.Info("停止 Kafka 消费者服务")
	if err := s.consumer.Close(); err != nil {
		return err
	}
	return nil
}
