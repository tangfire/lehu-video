package server

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	v1 "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/conf"
	"lehu-video/app/videoApi/service/internal/data"
	"lehu-video/app/videoApi/service/internal/pkg/resp"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"lehu-video/app/videoApi/service/internal/service"
	"os"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := map[string]struct{}{
		"/api.videoApi.service.v1.UserService/Login":                {},
		"/api.videoApi.service.v1.UserService/Register":             {},
		"/api.videoApi.service.v1.UserService/GetVerificationCode":  {},
		"/api.videoApi.service.v1.VideoService/FeedShortVideo":      {},
		"/api.videoApi.service.v1.VideoService/GetVideoById":        {},
		"/api.videoApi.service.v1.CommentService/ListComment4Video": {},
		"/api.videoApi.service.v1.CommentService/ListChildComment":  {},
		"/v1/auth/wechat-login":                                     {},
		"/v1/campus/forum/categories":                               {},
		"/v1/campus/forum/posts":                                    {},
		"/v1/campus/forum/posts/{id}":                               {},
		"/v1/campus/forum/posts/{id}/comments":                      {},
		"/v1/campus/users/{id}":                                     {},
		"/v1/campus/users/{id}/posts":                               {},
		"/v1/campus/analytics/track":                                {},
		"/healthz":                                                  {},
		"/readyz":                                                   {},
		"/ws":                                                       {},
	}
	return func(ctx context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

func NewCampusWhiteListMatcher() selector.MatchFunc {
	whiteList := map[string]struct{}{
		"/api.videoApi.service.v1.UserService/Login":               {},
		"/api.videoApi.service.v1.UserService/Register":            {},
		"/api.videoApi.service.v1.UserService/GetVerificationCode": {},
		"/v1/auth/wechat-login":                                    {},
		"/v1/campus/forum/categories":                              {},
		"/v1/campus/forum/posts":                                   {},
		"/v1/campus/forum/posts/{id}":                              {},
		"/v1/campus/forum/posts/{id}/comments":                     {},
		"/v1/campus/users/{id}":                                    {},
		"/v1/campus/users/{id}/posts":                              {},
		"/v1/campus/analytics/track":                               {},
		"/healthz":                                                 {},
		"/readyz":                                                  {},
	}
	return func(ctx context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, ac *conf.Auth,
	userService *service.UserServiceService,
	fileService *service.FileServiceService,
	videoService *service.VideoServiceService,
	commentService *service.CommentServiceService,
	favoriteService *service.FavoriteServiceService,
	followService *service.FollowServiceService,
	collectionService *service.CollectionServiceService,
	groupService *service.GroupServiceService,
	messageService *service.MessageServiceService,
	friendService *service.FriendServiceService,
	wsService *service.WebSocketService,
	campusService *service.CampusService,
	data *data.Data,
	logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			ratelimit.Server(), // 添加限流器
			selector.Server(
				jwt.Server(func(token *jwt2.Token) (interface{}, error) {
					return []byte(ac.ApiKey), nil
				}, jwt.WithSigningMethod(jwt2.SigningMethodHS256), jwt.WithClaims(func() jwt2.Claims {
					return &claims.Claims{}
				})),
			).
				Match(NewWhiteListMatcher()).
				Build(),
		),
		http.Filter(handlers.CORS(
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "Origin", "X-Request-ID"}),
			handlers.ExposedHeaders([]string{"X-Request-ID"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
			handlers.AllowedOrigins([]string{"*"}),
		), accessLogFilter(logger)),
		http.ResponseEncoder(resp.ResponseEncoder),
		http.ErrorEncoder(resp.ErrorEncoder),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterUserServiceHTTPServer(srv, userService)
	v1.RegisterFileServiceHTTPServer(srv, fileService)
	v1.RegisterVideoServiceHTTPServer(srv, videoService)
	v1.RegisterCommentServiceHTTPServer(srv, commentService)
	v1.RegisterFavoriteServiceHTTPServer(srv, favoriteService)
	v1.RegisterFollowServiceHTTPServer(srv, followService)
	v1.RegisterCollectionServiceHTTPServer(srv, collectionService)
	v1.RegisterGroupServiceHTTPServer(srv, groupService)
	v1.RegisterMessageServiceHTTPServer(srv, messageService)
	v1.RegisterFriendServiceHTTPServer(srv, friendService)
	campusService.RegisterRoutes(srv)
	registerHealthRoutes(srv.HandleFunc, "lehu-video.api.service", serviceVersion(), data)

	// 注册WebSocket路由 - 使用标准HTTP处理器
	// 注意：WebSocket需要绕过Kratos的中间件，所以直接使用原始HTTP处理器
	srv.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsService.GetHandler().ServeHTTP(w, r)
	})

	return srv
}

func NewCampusHTTPServer(c *conf.Server, ac *conf.Auth,
	userService *service.UserServiceService,
	fileService *service.FileServiceService,
	campusService *service.CampusService,
	data *data.Data,
	logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			ratelimit.Server(),
			selector.Server(
				jwt.Server(func(token *jwt2.Token) (interface{}, error) {
					return []byte(ac.ApiKey), nil
				}, jwt.WithSigningMethod(jwt2.SigningMethodHS256), jwt.WithClaims(func() jwt2.Claims {
					return &claims.Claims{}
				})),
			).
				Match(NewCampusWhiteListMatcher()).
				Build(),
		),
		http.Filter(handlers.CORS(
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "Origin", "X-Request-ID"}),
			handlers.ExposedHeaders([]string{"X-Request-ID"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
			handlers.AllowedOrigins([]string{"*"}),
		), accessLogFilter(logger)),
		http.ResponseEncoder(resp.ResponseEncoder),
		http.ErrorEncoder(resp.ErrorEncoder),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterUserServiceHTTPServer(srv, userService)
	v1.RegisterFileServiceHTTPServer(srv, fileService)
	campusService.RegisterRoutes(srv)
	registerHealthRoutes(srv.HandleFunc, "lehu-video.api.service", serviceVersion(), data)
	return srv
}

func serviceVersion() string {
	if version := os.Getenv("SERVICE_VERSION"); version != "" {
		return version
	}
	return "docker"
}
