package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

type CoreAdapterRepo struct {
	user       core.UserServiceClient
	video      core.VideoServiceClient
	collection core.CollectionServiceClient
	comment    core.CommentServiceClient
	favorite   core.FavoriteServiceClient
	follow     core.FollowServiceClient
}

func NewCoreAdapterRepo(
	user core.UserServiceClient,
	video core.VideoServiceClient,
	collection core.CollectionServiceClient,
	comment core.CommentServiceClient,
	favorite core.FavoriteServiceClient,
	follow core.FollowServiceClient) biz.CoreAdapterRepo {
	return &CoreAdapterRepo{
		user:       user,
		video:      video,
		collection: collection,
		comment:    comment,
		favorite:   favorite,
		follow:     follow,
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

func (r *CoreAdapterRepo) GetUserInfo(ctx context.Context, userId, accountId int64) (*core.User, error) {
	resp, err := r.user.GetUserInfo(ctx, &core.GetUserInfoReq{
		UserId:    userId,
		AccountId: accountId,
	})
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	return resp.User, nil
}

func (r *CoreAdapterRepo) CountFollow4User(ctx context.Context, userId int64) ([]int64, error) {
	resp, err := r.follow.CountFollow(ctx, &core.CountFollowReq{UserId: userId})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	return []int64{resp.FollowingCount, resp.FollowerCount}, nil
}

func (r *CoreAdapterRepo) CountBeFavoriteNumber4User(ctx context.Context, userId int64) (int64, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		Id:            []int64{userId},
		AggregateType: core.FavoriteAggregateType_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.Items[0].Count, nil
}

func (r *CoreAdapterRepo) CreateUser(ctx context.Context, mobile, email string, accountId int64) (int64, error) {
	resp, err := r.user.CreateUser(ctx, &core.CreateUserReq{
		Mobile:    mobile,
		Email:     email,
		AccountId: accountId,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.UserId, nil
}
