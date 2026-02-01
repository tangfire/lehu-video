package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type CoreAdapterImpl struct {
	user       core.UserServiceClient
	video      core.VideoServiceClient
	collection core.CollectionServiceClient
	comment    core.CommentServiceClient
	favorite   core.FavoriteServiceClient
	follow     core.FollowServiceClient
	log        *log.Helper
}

func NewCoreAdapter(
	user core.UserServiceClient,
	video core.VideoServiceClient,
	collection core.CollectionServiceClient,
	comment core.CommentServiceClient,
	favorite core.FavoriteServiceClient,
	follow core.FollowServiceClient,
	logger log.Logger) biz.CoreAdapter {
	return &CoreAdapterImpl{
		user:       user,
		video:      video,
		collection: collection,
		comment:    comment,
		favorite:   favorite,
		follow:     follow,
		log:        log.NewHelper(logger),
	}
}

func NewUserServiceClient(r registry.Discovery) core.UserServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewUserServiceClient(conn)
}

func NewVideoServiceClient(r registry.Discovery) core.VideoServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewVideoServiceClient(conn)
}

func NewCollectionServiceClient(r registry.Discovery) core.CollectionServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewCollectionServiceClient(conn)
}

func NewCommentServiceClient(r registry.Discovery) core.CommentServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewCommentServiceClient(conn)
}

func NewFavoriteServiceClient(r registry.Discovery) core.FavoriteServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewFavoriteServiceClient(conn)
}

func NewFollowServiceClient(r registry.Discovery) core.FollowServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewFollowServiceClient(conn)
}
