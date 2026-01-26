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

type CoreAdapterImpl struct {
	user       core.UserServiceClient
	video      core.VideoServiceClient
	collection core.CollectionServiceClient
	comment    core.CommentServiceClient
	favorite   core.FavoriteServiceClient
	follow     core.FollowServiceClient
}

func (r *CoreAdapterImpl) UpdateUserInfo(ctx context.Context, userId int64, name, avatar, backgroundImage, signature string) error {
	//TODO implement me
	panic("implement me")
}

func NewCoreAdapter(
	user core.UserServiceClient,
	video core.VideoServiceClient,
	collection core.CollectionServiceClient,
	comment core.CommentServiceClient,
	favorite core.FavoriteServiceClient,
	follow core.FollowServiceClient) biz.CoreAdapter {
	return &CoreAdapterImpl{
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

func (r *CoreAdapterImpl) GetUserInfo(ctx context.Context, userId, accountId int64) (*biz.UserInfo, error) {
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
	user := resp.User
	retUser := &biz.UserInfo{
		Id:              user.Id,
		Name:            user.Name,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Mobile:          user.Mobile,
		Email:           user.Email,
	}
	return retUser, nil
}

func (r *CoreAdapterImpl) CountFollow4User(ctx context.Context, userId int64) ([]int64, error) {
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

func (r *CoreAdapterImpl) CountBeFavoriteNumber4User(ctx context.Context, userId int64) (int64, error) {
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

func (r *CoreAdapterImpl) CreateUser(ctx context.Context, mobile, email string, accountId int64) (int64, error) {
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

func (r *CoreAdapterImpl) SaveVideoInfo(ctx context.Context, title, videoUrl, coverUrl, desc string, userId int64) (int64, error) {
	resp, err := r.video.PublishVideo(ctx, &core.PublishVideoReq{
		Title:       title,
		CoverUrl:    coverUrl,
		PlayUrl:     videoUrl,
		Description: desc,
		UserId:      userId,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.VideoId, nil
}

func (r *CoreAdapterImpl) GetVideoById(ctx context.Context, videoId int64) (*biz.Video, error) {
	resp, err := r.video.GetVideoById(ctx, &core.GetVideoByIdReq{
		VideoId: videoId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	video := resp.Video
	author := video.Author
	retVideo := &biz.Video{
		ID: video.Id,
		Author: &biz.VideoAuthor{
			ID:          author.Id,
			Name:        author.Name,
			Avatar:      author.Avatar,
			IsFollowing: author.IsFollowing != 0,
		},
		PlayUrl:       video.PlayUrl,
		CoverUrl:      video.CoverUrl,
		FavoriteCount: video.FavoriteCount,
		CommentCount:  video.CommentCount,
		IsFavorite:    video.IsFavorite != 0,
		Title:         video.Title,
	}
	return retVideo, nil
}
