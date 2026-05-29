# lehu-campus 校园 e站

校园 e站后端以小程序社区、课表、运营后台、e仔 AI/RAG 和浏览器内排障为主。旧项目栈已经从当前项目移除。

## 本地 Docker 启动

```bash
cd /Users/firetang/Documents/lehu/lehu-campus
docker compose up -d --build
```

如果本机之前用旧 Compose 项目名启动过，第一次切换到 `lehu-campus` 前先停旧 stack，避免端口或容器名冲突：

```bash
docker compose -p lehu-video-backend down
docker compose -p campus-estation-backend down
```

默认启动的校园 e站服务：

```text
mysql / redis / consul / minio / qdrant / campus-rag
base / campus-user / api / admin-web
health-exporter / prometheus / loki / alloy / grafana
```

默认关键环境变量：

```bash
export LEHU_STORAGE_PROVIDER=minio
```

本地地址：

```text
API：http://localhost:18080
运营后台：http://localhost:15173/admin
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

生产公开媒体不再建议走服务器本机 MinIO。帖子图片、头像、反馈图片、运营发帖图片使用腾讯云 COS + CDN：

```bash
export LEHU_STORAGE_PROVIDER=cos
export COS_SECRET_ID=腾讯云SecretId
export COS_SECRET_KEY=腾讯云SecretKey
export COS_REGION=ap-guangzhou
export COS_BUCKET=campus-1250000000
export COS_PUBLIC_CDN_BASE_URL=https://cdn.example.com
```

`/v1/campus/upload/presign` 仍返回预签名 PUT 地址，前端直传后调用 `/v1/campus/upload/complete`。生产环境下公开访问 URL 会返回 CDN 域名，不再占用轻量服务器出网带宽。

微信公众平台需要配置：

```text
request 合法域名：API 域名
uploadFile 合法域名：COS 上传域名，例如 https://campus-1250000000.cos.ap-guangzhou.myqcloud.com
downloadFile 合法域名：CDN 下载域名，例如 https://cdn.example.com
```

腾讯云控制台需要配置 COS CORS、CDN 回源、图片缓存规则和基础防盗刷策略。MinIO 只作为本地开发和低频内部文件过渡；知识库/RAG 文件暂不在这一阶段做公开 CDN 化，后续可单独迁到私有 COS。

帖子只支持文字和图片，后端固定拒绝视频上传和视频帖。

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

首发 400 人、关闭视频帖、公开媒体走 COS + CDN、图片压缩的前提下，建议：

```text
2核4G / 100GB / 7Mbps / 1000GB/月
```

不要降到 2G 内存。后续如果图片量、同时在线或活动峰值明显升高，再升级到 4核8G/更高带宽。
