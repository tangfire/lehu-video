package server

import (
	"context"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/service"
)

type KafkaConsumerServer struct {
	consumerSvc *service.KafkaConsumerService
	log         *log.Helper
}

func NewKafkaConsumerServer(
	consumerSvc *service.KafkaConsumerService,
	logger log.Logger,
) *KafkaConsumerServer {
	return &KafkaConsumerServer{
		consumerSvc: consumerSvc,
		log:         log.NewHelper(logger),
	}
}

// Start 启动消费者服务，实现 kratos.Service 接口
func (s *KafkaConsumerServer) Start(ctx context.Context) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LEHU_DISABLE_API_KAFKA_CONSUMER")), "true") {
		s.log.Info("API Kafka 消费者已关闭")
		return nil
	}
	s.log.Info("启动 Kafka 消费者服务")
	go func() {
		if err := s.consumerSvc.Run(ctx); err != nil {
			s.log.Errorf("Kafka 消费者运行错误: %v", err)
		}
	}()
	return nil
}

// Stop 停止消费者服务，实现 kratos.Service 接口
func (s *KafkaConsumerServer) Stop(ctx context.Context) error {
	s.log.Info("停止 Kafka 消费者服务")
	if err := s.consumerSvc.Close(); err != nil {
		return err
	}
	return nil
}
