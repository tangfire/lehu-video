# 校园 e站 Backend

校园 e站后端以小程序社区、课表、运营后台、e仔 AI/RAG 和浏览器内排障为主。短视频、IM chat、Kafka 链路已经作为历史栈归档到 `docker-compose.legacy.yml`，不再进入默认生产启动。

## 本地 Docker 启动

```bash
cd /Users/firetang/Documents/lehu/lehu-video
docker compose up -d --build
```

默认启动的校园 e站服务：

```text
mysql / redis / consul / minio / qdrant / campus-rag
base / core / api
health-exporter / prometheus / loki / alloy / grafana
```

默认不启动：

```text
kafka / kafka-init / chat
```

默认关键环境变量：

```bash
export LEHU_CAMPUS_ONLY=true
export LEHU_CAMPUS_ENABLE_VIDEO_POSTS=false
export LEHU_DISABLE_VIDEO_KAFKA_CONSUMERS=true
export LEHU_DISABLE_API_KAFKA_CONSUMER=true
```

本地地址：

```text
API：http://localhost:18080
Grafana：http://localhost:13002
Prometheus：http://localhost:19090
MinIO API：http://localhost:19000
MinIO 控制台：http://localhost:19001
```

## 生产必配

正式/体验服务器至少配置一个管理员：

```bash
export LEHU_CAMPUS_ADMIN_USER_IDS=你的用户ID
```

正式环境不要开启：

```bash
export LEHU_CAMPUS_ADMIN_ALLOW_ALL=true
export LEHU_WECHAT_MOCK_LOGIN=true
```

小程序正式登录需要：

```bash
export WECHAT_APP_ID=你的小程序AppID
export WECHAT_APP_SECRET=你的小程序AppSecret
export LEHU_WECHAT_MOCK_LOGIN=false
```

## 数据库与文件

默认初始化脚本：

```text
sql/campus.sql
```

默认数据库名：

```text
lehu_campus_db
```

默认 MinIO bucket / 文件域：

```text
campus
```

`sql/video.sql`、`sql/legacy/` 和 `docker-compose.legacy.yml` 仅保留历史短视频/chat 栈，不再作为校园 e站默认入口。

## e仔与 RAG

Go 后端负责任务、权限、e仔回复编排；`campus-rag` 只在 Docker 内网提供解析、切片、embedding、Qdrant 检索。

```text
CAMPUS_RAG_BASE_URL=http://campus-rag:8090
CAMPUS_RAG_EMBEDDING_MODEL=BAAI/bge-m3
SILICONFLOW_API_KEY=sk-xxx
```

e仔 AI 回复：

```text
DEEPSEEK_API_KEY=sk-xxx
CAMPUS_EZAI_BOT_USER_ID=123
CAMPUS_AI_DAILY_LIMIT=200
CAMPUS_AI_MODEL=deepseek-chat
CAMPUS_AI_BASE_URL=https://api.deepseek.com/chat/completions
```

未配置 API Key 时，e仔/RAG 会降级，不影响社区主链路。

## 监控与日志

健康检查：

```text
GET http://localhost:18080/healthz
GET http://localhost:18080/readyz
```

Grafana：

```text
http://localhost:13002
账号：admin
密码：admin
```

预置面板：

```text
Dashboards -> Campus e站 -> 校园 e站日志搜索
Dashboards -> Campus e站 -> 校园 e站健康监控
```

常用 LogQL：

```logql
{job="docker"} |= "用户提供的请求编号"
{job="docker", container="campus-api"} |= "/v1/campus/forum/posts"
{job="docker"} |~ "status(=|\":) ?500"
{job="docker"} |= "trace_id"
```

命令行兜底：

```bash
make logs-request RID=用户提供的请求编号 SINCE=30m
make logs-trace TID=trace_id SINCE=30m
make logs-search Q="/v1/campus/forum/posts" SINCE=2h
```

## 冒烟测试

```bash
API_BASE=http://127.0.0.1:18080/v1 ./scripts/smoke.sh
```

该脚本会注册测试用户、登录、读取校园版块并发布一条文字 smoke 帖。

## 成本建议

首发 400 人、关闭视频帖、本机 MinIO、图片压缩的前提下，建议：

```text
2核4G / 100GB / 7Mbps / 1000GB/月
```

不要降到 2G 内存。后续如果图片量、同时在线或活动峰值明显升高，再升级到 4核8G/更高带宽。
