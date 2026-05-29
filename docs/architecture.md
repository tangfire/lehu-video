# 校园 e站 Backend Architecture

校园 e站当前采用低风险收口方式：保留既有 Kratos 微服务骨架，但生产默认只运行校园业务需要的链路。

## Services

- `api`: 校园 e站 HTTP 入口，负责 JWT、运营后台、小程序接口、e仔任务编排和健康检查。
- `base`: 账号、验证码、文件签名上传和 MinIO 文件确认。
- `core`: 首发阶段作为用户资料服务使用，保留旧 user gRPC 能力。
- `campus-rag`: 知识库解析、切片、embedding、Qdrant 检索。
- `mysql / redis / minio / qdrant / consul`: 校园业务基础依赖。
- `grafana / loki / alloy / prometheus / health-exporter`: 浏览器内日志搜索和健康监控。

## Campus-Only Mode

默认 `LEHU_CAMPUS_ONLY=true`。API 在该模式下只注册用户、文件、校园接口和健康检查，不构造 chat gRPC、WebSocket、Kafka producer/consumer，也不注册旧视频/chat HTTP 路由。

旧短视频、IM chat、Kafka 链路保留在 `docker-compose.legacy.yml`、`sql/video.sql` 和 `sql/legacy/`，不进入默认启动。

## Data

默认数据库为 `lehu_campus_db`，初始化脚本为 `sql/campus.sql`。该脚本只包含账号、用户、文件、校园社区、通知、审核、安全、e仔/RAG 表。

媒体文件首发使用本机 MinIO，bucket 和文件域为 `campus`。首发默认 `LEHU_CAMPUS_ENABLE_VIDEO_POSTS=false`，只允许文字和图片帖子。

## Operations

健康状态先看 Grafana 的「校园 e站健康监控」；请求排障先用用户给的 `request_id` 在「校园 e站日志搜索」里查入口日志，再用同条日志里的 `trace_id` 搜下游调用。
