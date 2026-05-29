# lehu-campus Architecture

校园 e站当前只保留校园业务需要的 Kratos 服务和观测链路，旧项目栈已经从默认项目中移除。

## Services

- `api`: 校园 e站 HTTP 入口，负责 JWT、运营后台、小程序接口、e仔任务编排和健康检查。
- `base`: 账号、验证码、文件签名上传和对象存储确认，本地使用 MinIO，生产公开媒体使用腾讯云 COS + CDN。
- `campus-user`: 用户资料服务，提供创建用户、资料查询/更新、搜索、统计和最后在线时间能力。
- `campus-rag`: 知识库解析、切片、embedding、Qdrant 检索。
- `mysql / redis / minio / qdrant / consul`: 校园业务基础依赖。MinIO 主要用于本地开发和低频内部文件过渡。
- `grafana / loki / alloy / prometheus / health-exporter`: 浏览器内日志搜索和健康监控。

## Data

默认数据库为 `lehu_campus_db`，初始化脚本为 `sql/campus.sql`。该脚本只包含账号、用户、文件、校园社区、通知、审核、安全、e仔/RAG 表。

公开媒体首发使用腾讯云 COS + CDN，bucket 和文件域仍保持 `campus`，文件 object key 仍是 `public/{file_id}.{ext}` 这一类格式，不改数据库结构。生产设置 `LEHU_STORAGE_PROVIDER=cos` 后，`base` 会用 COS 生成上传预签名 URL，并把确认后的公开 URL 拼成 `COS_PUBLIC_CDN_BASE_URL/{object_key}`。

本地开发默认 `LEHU_STORAGE_PROVIDER=minio`，继续启动 MinIO。首发只允许文字和图片帖子，后端固定拒绝视频上传和视频帖。

微信小程序生产域名需要同时配置 API request 域名、COS 上传域名和 CDN 下载域名。COS/CDN 控制台需要配置 CORS、回源、缓存规则和基础防盗刷策略。知识库/RAG 文件第一阶段保持后台低频上传链路，后续再单独迁私有 COS。

## Operations

健康状态先看 Grafana 的「校园 e站健康监控」；请求排障先用用户给的 `request_id` 在「校园 e站日志搜索」里查入口日志，再用同条日志里的 `trace_id` 搜下游调用。
