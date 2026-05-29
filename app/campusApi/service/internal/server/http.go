package server

import (
	"context"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	v1 "lehu-video/api/campusApi/service/v1"
	"lehu-video/app/campusApi/service/internal/conf"
	"lehu-video/app/campusApi/service/internal/data"
	"lehu-video/app/campusApi/service/internal/pkg/resp"
	"lehu-video/app/campusApi/service/internal/pkg/utils/claims"
	"lehu-video/app/campusApi/service/internal/service"
)

func NewCampusWhiteListMatcher() selector.MatchFunc {
	whiteList := map[string]struct{}{
		"/api.campusApi.service.v1.UserService/Login":               {},
		"/api.campusApi.service.v1.UserService/Register":            {},
		"/api.campusApi.service.v1.UserService/GetVerificationCode": {},
		"/api.campusApi.service.v1.UserService/GetUserInfo":         {},
		"/api.campusApi.service.v1.UserService/BatchGetUserInfo":    {},
		"/api.campusApi.service.v1.UserService/SearchUsers":         {},
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
	}
	return func(ctx context.Context, operation string) bool {
		_, ok := whiteList[operation]
		return !ok
	}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, ac *conf.Auth,
	userService *service.UserServiceService,
	fileService *service.FileServiceService,
	campusService *service.CampusService,
	data *data.Data,
	logger log.Logger) *http.Server {
	authSecret := resolveAuthSecret(ac)
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			ratelimit.Server(),
			selector.Server(
				jwt.Server(func(token *jwt2.Token) (interface{}, error) {
					return []byte(authSecret), nil
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
	registerHealthRoutes(srv.HandleFunc, "campus-estation.api.service", serviceVersion(), data)
	return srv
}

func resolveAuthSecret(auth *conf.Auth) string {
	if value := strings.TrimSpace(os.Getenv("LEHU_JWT_SECRET")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("LEHU_AUTH_API_KEY")); value != "" {
		return value
	}
	if auth == nil || strings.TrimSpace(auth.ApiKey) == "" {
		return "fireshine"
	}
	return auth.ApiKey
}

func serviceVersion() string {
	if version := os.Getenv("SERVICE_VERSION"); version != "" {
		return version
	}
	return "docker"
}
