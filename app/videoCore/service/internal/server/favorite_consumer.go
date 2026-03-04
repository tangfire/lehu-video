// server/favorite_consumer.go
package server

import (
	"context"
	"lehu-video/app/videoCore/service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type FavoriteKafkaConsumerServer struct {
	consumer *biz.FavoriteConsumer
	log      *log.Helper
}

func NewFavoriteKafkaConsumerServer(consumer *biz.FavoriteConsumer, logger log.Logger) *FavoriteKafkaConsumerServer {
	return &FavoriteKafkaConsumerServer{
		consumer: consumer,
		log:      log.NewHelper(logger),
	}
}

func (s *FavoriteKafkaConsumerServer) Start(ctx context.Context) error {
	s.log.Info("启动 Favorite Kafka 消费者服务")
	s.consumer.Start() // 直接调用 Start，内部已启动 goroutine
	return nil
}

func (s *FavoriteKafkaConsumerServer) Stop(ctx context.Context) error {
	s.log.Info("停止 Favorite Kafka 消费者服务")
	s.consumer.Stop()
	return nil
}
