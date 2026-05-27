package data

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	grpc "google.golang.org/grpc"
	chat "lehu-video/api/videoChat/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type chatAdapterImpl struct {
	group   chat.GroupServiceClient
	message chat.MessageServiceClient
	friend  chat.FriendServiceClient
}

func NewChatAdapter(
	group chat.GroupServiceClient,
	message chat.MessageServiceClient,
	friend chat.FriendServiceClient,
) biz.ChatAdapter {
	return &chatAdapterImpl{
		group:   group,
		message: message,
		friend:  friend,
	}
}

func dialChatService(r registry.Discovery) (*grpc.ClientConn, error) {
	return kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		kgrpc.WithDiscovery(r),
		kgrpc.WithMiddleware(
			recovery.Recovery(),
			circuitbreaker.Client(), // 添加熔断器
		),
	)
}

func NewGroupServiceClient(r registry.Discovery) (chat.GroupServiceClient, error) {
	conn, err := dialChatService(r)
	if err != nil {
		return nil, fmt.Errorf("dial chat group service: %w", err)
	}
	return chat.NewGroupServiceClient(conn), nil
}

func NewMessageServiceClient(r registry.Discovery) (chat.MessageServiceClient, error) {
	conn, err := dialChatService(r)
	if err != nil {
		return nil, fmt.Errorf("dial chat message service: %w", err)
	}
	return chat.NewMessageServiceClient(conn), nil
}

func NewFriendServiceClient(r registry.Discovery) (chat.FriendServiceClient, error) {
	conn, err := dialChatService(r)
	if err != nil {
		return nil, fmt.Errorf("dial chat friend service: %w", err)
	}
	return chat.NewFriendServiceClient(conn), nil
}
