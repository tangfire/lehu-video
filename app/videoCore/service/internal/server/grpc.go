package server

import (
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/registry"
	consulAPI "github.com/hashicorp/consul/api"
	v1 "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoCore/service/internal/conf"
	"lehu-video/app/videoCore/service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server,
	videoService *service.VideoServiceService,
	feedService *service.FeedServiceService,
	userService *service.UserServiceService,
	followService *service.FollowServiceService,
	favoriteService *service.FavoriteServiceService,
	commentService *service.CommentServiceService,
	collectionService *service.CollectionServiceService,
	logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterVideoServiceServer(srv, videoService)
	v1.RegisterUserServiceServer(srv, userService)
	v1.RegisterFollowServiceServer(srv, followService)
	v1.RegisterCommentServiceServer(srv, commentService)
	v1.RegisterCollectionServiceServer(srv, collectionService)
	v1.RegisterFavoriteServiceServer(srv, favoriteService)
	v1.RegisterFeedServiceServer(srv, feedService)
	return srv
}

func NewRegistrar(conf *conf.Registry) registry.Registrar {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}
