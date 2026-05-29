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

### 校园 e站最小启动

如果只上线/内测校园 e站，不需要启动短视频和聊天相关容器。推荐最小启动：

```bash
export LEHU_DISABLE_VIDEO_KAFKA_CONSUMERS=true
export LEHU_DISABLE_API_KAFKA_CONSUMER=true
docker compose up -d mysql redis minio minio-init consul base core qdrant campus-rag api health-exporter prometheus loki alloy grafana
```

这套最小启动不包含 `kafka / kafka-init / chat`。如果要同时体验短视频 Feed 写扩散或聊天消息链路，再启动完整 `docker compose up -d`，并把上面两个禁用 Kafka 消费者的环境变量取消。

后台权限默认不会放开。正式环境请至少配置一个管理员：

```bash
export LEHU_CAMPUS_ADMIN_USER_IDS=你的用户ID
```

本地临时调试后台时，才使用：

```bash
export LEHU_CAMPUS_ADMIN_ALLOW_ALL=true
```

正式/体验服务器不要开启 `LEHU_CAMPUS_ADMIN_ALLOW_ALL=true`。

小程序体验版/正式版需要真实微信登录配置。正式服务器不要依赖 mock code：

```bash
export WECHAT_APP_ID=你的小程序AppID
export WECHAT_APP_SECRET=你的小程序AppSecret
export LEHU_WECHAT_MOCK_LOGIN=false
```

只有本地联调才可以临时开启：

```bash
export LEHU_WECHAT_MOCK_LOGIN=true
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

后端提供轻量健康检查，适合 Docker healthcheck、Prometheus 和 Grafana 告警：

```text
GET http://localhost:18080/healthz  # API 进程存活
GET http://localhost:18080/readyz   # MySQL / Redis 依赖可用性
```

启动内测监控面板：

```bash
docker compose up -d health-exporter prometheus loki alloy grafana
```

访问 Grafana：

```text
http://localhost:13002
```

Prometheus 本地调试入口：

```text
http://localhost:19090
```

本地默认账号：

```text
账号：admin
密码：admin
```

正式/体验服务器建议设置：

```bash
export GRAFANA_ADMIN_PASSWORD=换成强密码
```

Grafana 里已经预置两个面板：

```text
Dashboards -> Lehu -> 乐乎日志搜索
Dashboards -> Lehu -> 乐乎健康监控
```

本地开发环境常用访问地址：

```text
API 健康检查：http://localhost:18080/healthz
API 依赖检查：http://localhost:18080/readyz
Grafana 监控面板：http://localhost:13002
Prometheus 调试入口：http://localhost:19090
MinIO 文件/API：http://localhost:19000
MinIO 控制台：http://localhost:19001
```

MinIO 本地默认账号：

```text
账号：minioadmin
密码：minioadmin
```

`health-exporter` 会在 Docker 内网探测 API、Base、Core、RAG、MinIO、MySQL、Redis、Consul、Qdrant，并把结果暴露给 Prometheus。Grafana 的「乐乎健康监控」面板会直接显示失败目标和耗时；Prometheus / Grafana 里也已经有 `LehuProbeDown` 这类基础告警规则。

需要企业微信、邮件或 webhook 推送时，在 Grafana 里进入 `Alerting -> Contact points` 添加通知方式，再到 `Notification policies` 绑定默认策略。通知地址、邮箱账号这类敏感配置不要写进仓库；正式服务器建议用环境变量或 Grafana UI 保存。

Grafana + Loki 用来在浏览器里集中查看 Docker 容器日志，不需要进入服务器挨个容器搜。打开 `http://localhost:13002`，进入 `Dashboards -> Lehu -> 乐乎日志搜索`，选择容器或 All，再输入用户给的 `request_id`、`trace_id`、接口路径或错误关键词即可搜索全容器日志。

如果需要临时写更细的 LogQL，可以进入 Explore，数据源选择 Loki，常用查询：

```logql
{job="docker"} |= "用户提供的请求编号"
{job="docker", container="lehu-api"} |= "/v1/campus/forum/posts"
{job="docker"} |~ "status(=|\":) ?500"
{job="docker"} |= "trace_id"
```

首次接入时 Alloy 只回灌最近 30 分钟的 Docker 历史日志，避免把旧日志刷爆 Loki；后续正常运行产生的日志会按 Loki 本地 7 天留存。

定位具体请求时，先用用户反馈的 `request_id` 搜入口日志；如果日志里有 `trace_id`，再用同一个 `trace_id` 搜全容器，就能看到 `api / base / core / chat` 的同一次调用链。如果 Grafana/Loki 本身不可用，保留命令行兜底查询：

```bash
make logs-request RID=用户提供的请求编号 SINCE=30m
make logs-trace TID=上一条命令提示的trace_id SINCE=30m
make logs-search Q="/v1/campus/forum/posts" SINCE=2h
make logs-search Q="status=500" SINCE=2h
```

`request_id` 是每次接口请求的排障编号：

- 小程序接口失败时，错误弹窗会显示“请求编号”，用户可以截图或点“复制编号”发给你。
- 运营后台接口失败时，浏览器控制台会打印 `[request failed]`，里面有 `request_id`。
- 后端入口日志每条请求都会带同一个 `request_id`，同时带 `trace_id`。`request_id` 用来找到入口请求，`trace_id` 用来跨容器追踪这次请求触发的下游调用。
- 如果用户没有提供编号，也可以按大概时间、接口路径、用户 ID、IP、`status=500/429/403` 搜。

正式服务器不要把 Grafana 和 Prometheus 直接暴露给公网所有人，建议用防火墙限制访问 IP，或放到内网/VPN 后面。

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

生产环境建议先把每日上限设低一点，观察 Grafana 日志里的 `e仔 AI 回复任务`错误和 DeepSeek 控制台用量后再逐步调高。大模型回复只作为校园生活问答参考，不替代学校官方通知。

## e仔知识库 RAG

后台新增「e仔知识库」，运营可以上传 PDF/DOCX/TXT/MD 或手动录入学校资料。Go 后端负责权限、任务和 e仔回复编排，`campus-rag` 只在 Docker 内网提供解析、切片、embedding、Qdrant 检索能力。

本地启动知识库依赖：

```bash
docker compose up -d qdrant campus-rag
```

API 容器会通过内网访问：

```text
CAMPUS_RAG_BASE_URL=http://campus-rag:8090
```

需要配置低成本 embedding：

```text
SILICONFLOW_API_KEY=sk-xxx
SILICONFLOW_BASE_URL=https://api.siliconflow.cn/v1
CAMPUS_RAG_EMBEDDING_MODEL=BAAI/bge-m3
```

本地调试地址：

```text
Qdrant：http://localhost:16333
RAG 健康检查：docker compose exec campus-rag python -c "import urllib.request; print(urllib.request.urlopen('http://127.0.0.1:8090/healthz').read().decode())"
```

如果生产环境已有旧数据库，需要先执行：

```bash
mysql -h <host> -u <user> -p lehu_video_db < sql/20260529_campus_knowledge_rag.sql
```

RAG 服务没有公网端口，正式部署时保持内网访问即可。未配置 `CAMPUS_RAG_BASE_URL` 时，Go 后端会自动降级为普通 e仔回复；未配置 `SILICONFLOW_API_KEY` 时，知识库索引和测试提问会返回可读错误，不影响小程序主链路。

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
