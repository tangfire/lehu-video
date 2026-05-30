# 上线与部署手册

这份文档写给真正要把校园 e站放到服务器上的人。它不解释代码细节，重点是上线前要配什么、怎么启动、怎么验收、出问题先看哪里。

## 部署形态

生产默认使用两个 Compose 文件叠加：

```bash
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

本地开发继续只用：

```bash
docker compose up -d --build
```

生产覆盖文件会收紧端口：

- MySQL、Redis、Consul、MinIO、Qdrant、Prometheus、base、campus-user 不暴露到公网。
- API、运营后台、Grafana 绑定宿主机 `127.0.0.1`。
- 外部 HTTPS 由 Caddy/Nginx 反向代理接入。

```text
公网用户 -> HTTPS 反代 -> 127.0.0.1:18080 API
运营人员 -> HTTPS 反代 -> 127.0.0.1:15173 运营后台
管理员 -> HTTPS 反代 -> 127.0.0.1:13002 Grafana
```

## 上线前准备

### 服务器

首发建议：

```text
2核4G / 100GB / 7Mbps / 1000GB/月
```

这个配置建立在这些前提上：

- 首发只做文字和图片，不开放视频。
- 公开图片走腾讯云 COS + CDN，不走服务器本机出网。
- 日志和访问记录都有保留期，不无限增长。
- 400 首批用户不是同时高并发刷图。

不建议 2G 内存，因为 MySQL、Redis、Grafana、Loki、Prometheus、Qdrant、RAG 和 Go 服务一起跑会很紧。

### 域名

至少准备：

| 域名 | 指向 |
| --- | --- |
| `api.example.com` | API 反代到 `127.0.0.1:18080` |
| `admin.example.com` | 运营后台反代到 `127.0.0.1:15173` |
| `grafana.example.com` | Grafana 反代到 `127.0.0.1:13002` |
| `cdn.example.com` | CDN 下载域名，回源 COS |

Grafana 域名可以只给自己访问，但也要用强密码和 HTTPS。

## 生产环境变量

从示例开始：

```bash
cp .env.production.example .env.production
```

必须改掉的密钥：

```bash
MYSQL_ROOT_PASSWORD=...
REDIS_PASSWORD=...
MINIO_ROOT_USER=...
MINIO_ROOT_PASSWORD=...
LEHU_JWT_SECRET=...
GRAFANA_ADMIN_PASSWORD=...
```

数据库和 Redis：

```bash
LEHU_MYSQL_DSN=root:密码@tcp(mysql:3306)/lehu_campus_db?parseTime=True&loc=Local
LEHU_REDIS_ADDR=redis:6379
LEHU_REDIS_PASSWORD=...
LEHU_REDIS_DB=0
LEHU_REDIS_CACHE_ENABLED=true
LEHU_CACHE_POST_LIST_TTL=10s
LEHU_CACHE_POST_DETAIL_TTL=30s
LEHU_CACHE_ADMIN_SUMMARY_TTL=60s
LEHU_CACHE_SECURITY_OVERVIEW_TTL=60s
LEHU_CACHE_CATEGORIES_TTL=30m
LEHU_CACHE_MOMENTS_CANDIDATES_TTL=3m
```

Redis 上线主要承担真实 IP 限流和热点读缓存；验证码能力仍保留在旧账号基础服务里，但小程序主链路不依赖它。热点缓存只覆盖公开帖子流、帖子详情、分类、后台 summary、安全 overview、朋友圈候选；MySQL 仍是最终数据源，Redis 异常时接口回落 MySQL。

公开媒体存储：

```bash
LEHU_STORAGE_PROVIDER=cos
COS_SECRET_ID=...
COS_SECRET_KEY=...
COS_REGION=ap-guangzhou
COS_BUCKET=campus-1250000000
COS_PUBLIC_CDN_BASE_URL=https://cdn.example.com
LEHU_ENABLE_LEGACY_UPLOAD=false
```

微信小程序：

```bash
WECHAT_APP_ID=wx...
WECHAT_APP_SECRET=...
WECHAT_MINIPROGRAM_QR_ENV_VERSION=release
LEHU_WECHAT_MOCK_LOGIN=false
```

后台权限：

```bash
LEHU_CAMPUS_ADMIN_USER_IDS=2060000000000000000
LEHU_CAMPUS_OPERATOR_USER_IDS=
LEHU_CAMPUS_ADMIN_ALLOW_ALL=false
```

AI/RAG：

```bash
DEEPSEEK_API_KEY=
CAMPUS_AI_API_KEY=
CAMPUS_AI_BASE_URL=https://api.deepseek.com/chat/completions
CAMPUS_AI_MODEL=deepseek-chat
CAMPUS_AI_DAILY_LIMIT=200
CAMPUS_AI_AUDIT_API_KEY=
CAMPUS_EZAI_BOT_USER_ID=
SILICONFLOW_API_KEY=
```

飞书告警：

```bash
LEHU_ALERT_ENV=prod
LEHU_ALERT_WEBHOOK_TOKEN=一段随机长token
LEHU_ALERT_FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/xxx
LEHU_ALERT_FEISHU_SECRET=
GRAFANA_ROOT_URL=https://grafana.example.com
```

真实 IP 和日志保留：

```bash
LEHU_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
LEHU_ACCESS_LOG_RETENTION_DAYS=15
```

## 启动前检查

```bash
docker compose config >/tmp/campus-compose.local.yml
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml config >/tmp/campus-compose.prod.yml
```

如果生产 config 阶段就报缺少环境变量，要先补 `.env.production`，不要临时改 compose 绕过去。

## 启动

```bash
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml up -d --build
docker compose ps
```

第一次启动后等 MySQL、Qdrant、RAG、Grafana 全部起来，再做接口检查。

```bash
curl http://127.0.0.1:18080/healthz
curl http://127.0.0.1:18080/readyz
```

## 首次管理员

生产不要开启：

```bash
LEHU_CAMPUS_ADMIN_ALLOW_ALL=true
LEHU_WECHAT_MOCK_LOGIN=true
```

推荐流程：

1. 用小程序真实登录一次，生成用户。
2. 在数据库或后台日志里确认自己的用户 ID。
3. 把用户 ID 写进 `LEHU_CAMPUS_ADMIN_USER_IDS`。
4. 重启 `api`。
5. 用运营后台登录，进入权限管理确认账号状态。

## 上线验收

最少要过这些 smoke：

| 模块 | 验收项 |
| --- | --- |
| API | `/healthz`、`/readyz` 正常 |
| 小程序登录 | 真实微信登录成功，mock 登录关闭 |
| 发帖 | 文字帖成功，图片帖成功，视频被拒绝 |
| 上传 | `/presign` 返回 COS URL，`/complete` 返回 CDN URL |
| 互动 | 评论、点赞、收藏、通知正常 |
| 审核 | 不审核、人工审核、AI 初审开关可保存 |
| 后台 | 内容工作台、举报反馈、权限管理可用 |
| e仔 | 人设保存、知识库测试、失败任务页可用 |
| 朋友圈素材 | 可生成素材包，扫码能进帖子详情 |
| Grafana | 日志搜索和健康监控有数据 |
| 飞书 | 模拟告警能发到群 |

命令行 smoke：

```bash
API_BASE=https://api.example.com/v1 ./scripts/smoke.sh
```

## 上线当天流程

1. 备好 `.env.production`，确认没有占位密码。
2. 确认 COS CORS、CDN 回源、微信合法域名已经配置。
3. 启动生产 compose。
4. 确认 API、运营后台、Grafana 都能通过 HTTPS 打开。
5. 跑 smoke。
6. 在运营后台确认审核模式。
7. 在 Grafana 看健康面板至少 5 分钟。
8. 发一条测试飞书告警。
9. 再开放给第一批用户。

## 常见问题

| 现象 | 优先看 |
| --- | --- |
| 后台打不开 | 反代、`admin-web` 容器、`ADMIN_API_BASE_URL` |
| 后台登录失败 | API 日志、管理员用户 ID、JWT secret |
| 小程序登录失败 | `WECHAT_APP_ID`、`WECHAT_APP_SECRET`、微信 request 域名 |
| 图片上传失败 | COS CORS、COS 密钥、bucket、CDN 域名 |
| Grafana no data | `health-exporter`、Prometheus datasource、Alloy/Loki |
| e仔不回复 | `CAMPUS_EZAI_BOT_USER_ID`、模型 key、失败任务页 |
| RAG 没命中 | 文档状态、切片、`SILICONFLOW_API_KEY`、置信度 |

## 回滚思路

首发阶段优先做简单回滚：

```bash
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

如果是配置问题，先改 `.env.production`，只重启相关服务：

```bash
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml up -d api base admin-web
```

运行中数据库不要随便 drop 表。新库结构以 `sql/campus.sql` 为准。
