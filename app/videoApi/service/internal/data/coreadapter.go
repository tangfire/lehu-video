package data

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
)

type CoreAdapterImpl struct {
	user       core.UserServiceClient
	video      core.VideoServiceClient
	feed       core.FeedServiceClient
	collection core.CollectionServiceClient
	comment    core.CommentServiceClient
	favorite   core.FavoriteServiceClient
	follow     core.FollowServiceClient
	log        *log.Helper
}

func NewCoreAdapter(
	user core.UserServiceClient,
	video core.VideoServiceClient,
	feed core.FeedServiceClient,
	collection core.CollectionServiceClient,
	comment core.CommentServiceClient,
	favorite core.FavoriteServiceClient,
	follow core.FollowServiceClient,
	logger log.Logger) biz.CoreAdapter {
	return &CoreAdapterImpl{
		user:       user,
		video:      video,
		feed:       feed,
		collection: collection,
		comment:    comment,
		favorite:   favorite,
		follow:     follow,
		log:        log.NewHelper(logger),
	}
}

func NewUserServiceClient(r registry.Discovery) (core.UserServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core user service: %w", err)
	}
	return core.NewUserServiceClient(conn), nil
}

func NewFeedServiceClient(r registry.Discovery) (core.FeedServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core feed service: %w", err)
	}
	return core.NewFeedServiceClient(conn), nil
}

func NewVideoServiceClient(r registry.Discovery) (core.VideoServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core video service: %w", err)
	}
	return core.NewVideoServiceClient(conn), nil
}

func NewCollectionServiceClient(r registry.Discovery) (core.CollectionServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core collection service: %w", err)
	}
	return core.NewCollectionServiceClient(conn), nil
}

func NewCommentServiceClient(r registry.Discovery) (core.CommentServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core comment service: %w", err)
	}
	return core.NewCommentServiceClient(conn), nil
}

func NewFavoriteServiceClient(r registry.Discovery) (core.FavoriteServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core favorite service: %w", err)
	}
	return core.NewFavoriteServiceClient(conn), nil
}

func NewFollowServiceClient(r registry.Discovery) (core.FollowServiceClient, error) {
	conn, err := dialService(r, "discovery:///lehu-video.core.service")
	if err != nil {
		return nil, fmt.Errorf("dial core follow service: %w", err)
	}
	return core.NewFollowServiceClient(conn), nil
}
