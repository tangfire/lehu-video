package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	base "lehu-video/api/base/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

const (
	DomainName = "shortvideo"
	BizName    = "short_video"
	Public     = "public"
)

type baseAdapterImpl struct {
	account base.AccountServiceClient
	auth    base.AuthServiceClient
	file    base.FileServiceClient
}

func NewBaseAdapter(account base.AccountServiceClient, auth base.AuthServiceClient, file base.FileServiceClient) biz.BaseAdapter {
	return &baseAdapterImpl{
		account: account,
		auth:    auth,
		file:    file,
	}
}

func NewAccountServiceClient(r registry.Discovery) base.AccountServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.base.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return base.NewAccountServiceClient(conn)
}

func NewAuthServiceClient(r registry.Discovery) base.AuthServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.base.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return base.NewAuthServiceClient(conn)
}

func NewFileServiceClient(r registry.Discovery) base.FileServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.base.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return base.NewFileServiceClient(conn)
}
