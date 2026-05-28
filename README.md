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

## 内测监控与健康检查

后端提供轻量健康检查，适合 Uptime Kuma 和 Docker healthcheck：

```text
GET http://localhost:18080/healthz  # API 进程存活
GET http://localhost:18080/readyz   # MySQL / Redis 依赖可用性
```

启动内测监控面板：

```bash
docker compose up -d uptime-kuma node-exporter dozzle
```

访问 Uptime Kuma：

```text
http://localhost:13001
```

访问 Dozzle 日志面板：

```text
http://localhost:13002
```

推荐在 Uptime Kuma 里先添加这些监控项：

```text
API 存活：http://api:8080/healthz
API 依赖：http://api:8080/readyz
MinIO 控制台：http://minio:9001
Node Exporter：http://node-exporter:9100/metrics
```

本地开发环境常用访问地址：

```text
API 健康检查：http://localhost:18080/healthz
API 依赖检查：http://localhost:18080/readyz
Uptime Kuma：http://localhost:13001
Dozzle 日志面板：http://localhost:13002
MinIO 文件/API：http://localhost:19000
MinIO 控制台：http://localhost:19001
Node Exporter 指标：http://localhost:19100/metrics
```

MinIO 本地默认账号：

```text
账号：minioadmin
密码：minioadmin
```

内测阶段先用 Uptime Kuma 面板观察服务状态；后续需要微信、企业微信或邮件提醒时，可在 Kuma 的「通知」里补 webhook。磁盘建议预警阈值先按 80% 设置，图片/视频上传变多后重点关注 MinIO 和 MySQL volume。

Dozzle 用来在浏览器里看 Docker 容器日志。用户反馈请求失败时，优先在 Dozzle 里打开 `lehu-api`，搜索前端返回的 `request_id`，也可以搜索接口路径、`status`、`error`、`429`、`500` 等关键词。

`node-exporter` 在本地 Docker Desktop 下以只读方式挂载主机根目录，主要用于快速查看 CPU、内存、磁盘等基础指标；正式服务器如需更完整的主机指标，再按 Linux 环境调整挂载参数。

正式服务器不要把 Uptime Kuma 和 Dozzle 直接暴露给公网所有人，建议用防火墙限制访问 IP，或放到内网/VPN 后面。

## 深汕e仔 AI 回复

评论区支持用户通过 `@深汕e仔` 或 `@e仔` 召唤官方 AI 回复。用户评论会先正常发布，后台任务再异步调用大模型生成 e仔回复；未配置 API Key 时功能自动关闭，不影响普通评论。

需要配置：

```text
DEEPSEEK_API_KEY=sk-xxx                 # 或 CAMPUS_AI_API_KEY
CAMPUS_EZAI_BOT_USER_ID=123             # 深汕e仔官方账号 user_id
CAMPUS_AI_DAILY_LIMIT=200               # e仔每日总回复上限，默认 200
CAMPUS_AI_MODEL=deepseek-chat           # 默认 deepseek-chat
CAMPUS_AI_BASE_URL=https://api.deepseek.com/chat/completions
```

生产环境建议先把每日上限设低一点，观察 Dozzle 日志里的 `e仔 AI 回复任务`错误和 DeepSeek 控制台用量后再逐步调高。大模型回复只作为校园生活问答参考，不替代学校官方通知。

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
