package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
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

func NewGroupServiceClient(r registry.Discovery) chat.GroupServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return chat.NewGroupServiceClient(conn)
}

func NewMessageServiceClient(r registry.Discovery) chat.MessageServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return chat.NewMessageServiceClient(conn)
}

func NewFriendServiceClient(r registry.Discovery) chat.FriendServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.chat.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return chat.NewFriendServiceClient(conn)
}
