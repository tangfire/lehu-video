# Lehu Video Backend

## 项目亮点

- Kratos 微服务结构：`videoApi / base / videoCore / videoChat`
- Docker Compose 一键启动 MySQL、Redis、Kafka、MinIO、Consul 和后端服务
- 统一 HTTP 响应与业务错误码
- JWT 登录态，HTTP 与 WebSocket 共用 token 校验
- Argon2id 密码存储，并兼容旧 MD5+salt 账号自动迁移
- Feed、点赞、评论、收藏、播放量使用 Redis counter + 异步落库 + 对账思路

架构说明见：[docs/architecture.md](docs/architecture.md)

## 本地 Docker 启动

启动前先打开 Docker Desktop。

```bash
cd /Users/firetang/Documents/lehu/lehu-video
docker compose up -d
```

后端默认访问地址：

```text
http://localhost:18080
```

如果修改了后端代码或 Docker 配置，需要重新构建：

```bash
docker compose up -d --build
```

查看容器状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f
```

常用开发命令：

```bash
make test        # 运行后端测试
make docker-up   # 重新构建并启动后端 Docker 服务
make docker-down # 停止后端 Docker 服务
make smoke       # 运行本地核心链路 smoke 检查
make proto       # 生成 protobuf 代码
```

停止后端：

```bash
docker compose down
```

建议先启动后端，再到前端项目目录启动前端：

```bash
cd /Users/firetang/Documents/lehu/lehu-video-frontend
docker compose up -d
```

前端默认访问地址：

```text
http://localhost:15173
```

## Kratos Project Template

## Install Kratos
```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```
## Create a service
```
# Create a template project
kratos new server

cd server
# Add a proto template
kratos proto add api/server/server.proto
# Generate the proto code
kratos proto client api/server/server.proto
# Generate the source code of service by proto file
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```
## Generate other auxiliary files by Makefile
```
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
# Generate all files
make all
```
## Automated Initialization (wire)
```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Docker
```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/your/configs>:/data/conf <your-docker-image-name>
```
