package main

import (
	"flag"
	"fmt"
	"github.com/go-kratos/kratos/v2/registry"
	logger2 "lehu-video/app/videoApi/service/internal/pkg/logger"
	"lehu-video/app/videoApi/service/internal/server"
	"math/rand"
	"os"
	"time"

	"lehu-video/app/videoApi/service/internal/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "lehu-video.api.service"
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id = generateUniqueID()
)

func generateUniqueID() string {
	hostname, _ := os.Hostname()

	// 获取进程 ID
	pid := os.Getpid()

	// 获取当前时间戳
	timestamp := time.Now().UnixNano()

	// 组合成唯一 ID
	return fmt.Sprintf("%s-%d-%d-%d", hostname, pid, timestamp, rand.Intn(1000))
}

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, rr registry.Registrar, hs *http.Server, kcs *server.KafkaConsumerServer) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			kcs,
		),
		kratos.Registrar(rr),
	)
}

func main() {
	flag.Parse()

	zapLogger := logger2.NewZapLogger("debug")
	defer zapLogger.Sync()

	logger := log.With(zapLogger,
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	var rc conf.Registry
	if err := c.Scan(&rc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Server, &rc, bc.Data, bc.Auth, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
