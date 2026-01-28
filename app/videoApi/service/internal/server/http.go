package server

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	v1 "lehu-video/api/videoApi/service/v1"
	"lehu-video/app/videoApi/service/internal/conf"
	"lehu-video/app/videoApi/service/internal/pkg/resp"
	"lehu-video/app/videoApi/service/internal/pkg/utils/claims"
	"lehu-video/app/videoApi/service/internal/service"
)

func NewWhiteListMatcher() selector.MatchFunc {

	whiteList := make(map[string]struct{})
	whiteList["/api.videoApi.service.v1.UserService/Login"] = struct{}{}
	whiteList["/api.videoApi.service.v1.UserService/Register"] = struct{}{}
	whiteList["/api.videoApi.service.v1.UserService/GetVerificationCode"] = struct{}{}
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
	logger log.Logger) *http.Server {
	fmt.Println("ac api_key = " + ac.ApiKey)
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
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
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
			handlers.AllowedOrigins([]string{"*"}),
		)),
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
	return srv
}
