package data

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	grpc "google.golang.org/grpc"
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

func dialService(r registry.Discovery, endpoint string) (*grpc.ClientConn, error) {
	return kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint(endpoint),
		kgrpc.WithDiscovery(r),
		kgrpc.WithMiddleware(
			recovery.Recovery(),
			tracing.Client(),
			circuitbreaker.Client(), // 添加熔断器
		),
	)
}

func NewAccountServiceClient(r registry.Discovery) (base.AccountServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.base.service")
	if err != nil {
		return nil, fmt.Errorf("dial base account service: %w", err)
	}
	return base.NewAccountServiceClient(conn), nil
}

func NewAuthServiceClient(r registry.Discovery) (base.AuthServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.base.service")
	if err != nil {
		return nil, fmt.Errorf("dial base auth service: %w", err)
	}
	return base.NewAuthServiceClient(conn), nil
}

func NewFileServiceClient(r registry.Discovery) (base.FileServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.base.service")
	if err != nil {
		return nil, fmt.Errorf("dial base file service: %w", err)
	}
	return base.NewFileServiceClient(conn), nil
}
