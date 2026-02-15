package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoCore/service/internal/service"
)

// LikeConsumerServer 用于启动点赞消费者
type LikeConsumerServer struct {
	consumer *service.LikeConsumer
	log      *log.Helper
}

func NewLikeConsumerServer(consumer *service.LikeConsumer, logger log.Logger) *LikeConsumerServer {
	return &LikeConsumerServer{
		consumer: consumer,
		log:      log.NewHelper(logger),
	}
}

// Start 启动消费者
func (s *LikeConsumerServer) Start(ctx context.Context) error {
	s.log.Info("starting like consumer server")
	go func() {
		if err := s.consumer.Run(ctx); err != nil {
			s.log.Errorf("like consumer run error: %v", err)
		}
	}()
	return nil
}

// Stop 停止消费者
func (s *LikeConsumerServer) Stop(ctx context.Context) error {
	s.log.Info("stopping like consumer server")
	return s.consumer.Close()
}
