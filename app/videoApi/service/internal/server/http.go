package server

import (
	"context"
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
	whiteList["/ws"] = struct{}{} // 添加WebSocket路径到白名单
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
	wsService *service.WebSocketService,
	logger log.Logger) *http.Server {
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
	v1.RegisterCollectionServiceHTTPServer(srv, collectionService)
	v1.RegisterGroupServiceHTTPServer(srv, groupService)
	v1.RegisterMessageServiceHTTPServer(srv, messageService)

	// 注册WebSocket路由 - 使用标准HTTP处理器
	// 注意：WebSocket需要绕过Kratos的中间件，所以直接使用原始HTTP处理器
	srv.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsService.GetHandler().ServeHTTP(w, r)
	})

	// 可选的WebSocket连接测试页面
	srv.HandleFunc("/ws-test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
            <!DOCTYPE html>
            <html>
            <head>
                <title>WebSocket测试</title>
            </head>
            <body>
                <h1>WebSocket连接测试</h1>
                <div>
                    <input type="text" id="userId" placeholder="用户ID" value="1001">
                    <button onclick="connect()">连接</button>
                    <button onclick="disconnect()">断开</button>
                    <span id="status">未连接</span>
                </div>
                <div id="messages"></div>
                <script>
                    let ws = null;
                    function connect() {
                        const userId = document.getElementById('userId').value;
                        const url = 'ws://' + window.location.host + '/ws?token=' + userId;
                        ws = new WebSocket(url);
                        ws.onopen = () => {
                            document.getElementById('status').innerText = '已连接';
                            console.log('WebSocket连接成功');
                        };
                        ws.onmessage = (event) => {
                            console.log('收到消息:', event.data);
                            const msgDiv = document.createElement('div');
                            msgDiv.innerText = '收到: ' + event.data;
                            document.getElementById('messages').appendChild(msgDiv);
                        };
                        ws.onclose = () => {
                            document.getElementById('status').innerText = '已断开';
                        };
                    }
                    function disconnect() {
                        if (ws) {
                            ws.close();
                            ws = null;
                        }
                    }
                </script>
            </body>
            </html>
        `))
	})

	return srv
}
