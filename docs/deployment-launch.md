# 上线与部署手册

这份文档写给真正要把校园 e站放到服务器上的人。它不解释代码细节，重点是上线前要配什么、怎么启动、怎么验收、出问题先看哪里。

上线当天逐项勾选用 `docs/launch-readiness-checklist.md`；这份文档偏部署说明，那份文档偏 Go/No-Go 验收。

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

为什么必须反代：

- 微信小程序正式环境要求 HTTPS 合法域名，不能直接请求 `http://IP:端口`。
- Docker 服务只绑定 `127.0.0.1`，公网不能直接访问内部端口。
- HTTPS 证书由 Caddy/Nginx 统一申请和续期，后端服务不处理证书。
- 后台和 Grafana 不裸露端口，后续可以加访问控制。
- 以后做蓝绿发布时，只需要改反代 upstream，用户访问域名不变。

推荐 Caddy 示例：

```caddyfile
api.example.com {
    reverse_proxy 127.0.0.1:18080
}

admin.example.com {
    reverse_proxy 127.0.0.1:15173
}

grafana.example.com {
    reverse_proxy 127.0.0.1:13002
}
```

实际部署时把 `example.com` 换成真实域名，并先把域名 DNS A 记录指向服务器公网 IP。Grafana 域名建议只给自己使用，至少配强密码；如果条件允许，再加 IP 白名单或 Basic Auth。

## 上线前准备

### 服务器

首发建议：

```text
2核4G 轻量服务器 + 1核1G 云 MySQL + 本机 Redis
```

这个配置建立在这些前提上：

- 首发只做文字和图片，不开放视频。
- 公开图片走腾讯云 COS + CDN，不走服务器本机出网。
- 业务数据统一放同一个云 MySQL，不做双 MySQL 拆库。
- Redis 承担真实 IP 限流和热点读缓存，降低首页刷帖和后台统计对 MySQL 的重复查询。
- 普通容器日志走 Loki；MySQL 里的 `campus_access_log` 默认只保留 7 天。
- 300 人试运营不是同时高并发刷图。

不建议 2G 内存。即使 MySQL 拆到云数据库，Redis、Grafana、Loki、Prometheus、Qdrant、RAG 和 Go 服务一起跑也会很紧。

数据库建议使用同地域、可内网连接的 1核1G 云 MySQL。核心用户数据、帖子、评论、点赞、收藏、通知、审核、权限、文件记录、e仔/RAG 质量数据都放云 MySQL；不要为了日志再拆一套 Docker MySQL，跨库统计和排障复杂度不划算。后续如果日活、慢查询或 MySQL CPU 明显升高，再升级到 2核4G 云 MySQL。

生产 compose 默认不会启动本地 Docker MySQL、MinIO 和 `minio-init`，它们只保留在 `local-stateful` profile 里给临时自建或本地调试使用。生产健康监控也不再探测本地 `mysql_tcp/minio_health`，云 MySQL 是否可用先由 `api_ready` 间接覆盖，细节看云厂商监控。

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
REDIS_PASSWORD=...
LEHU_JWT_SECRET=...
GRAFANA_ADMIN_PASSWORD=...
LEHU_MYSQL_DSN=campus_app:...@tcp(云 MySQL 内网地址:3306)/lehu_campus_db?parseTime=True&loc=Local
```

数据库和 Redis：

```bash
LEHU_MYSQL_DSN=业务账号:密码@tcp(云MySQL内网地址:3306)/lehu_campus_db?parseTime=True&loc=Local
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

RAG/Qdrant 资源限制：

```bash
QDRANT_MEM_LIMIT=768m
QDRANT_CPUS=0.75
CAMPUS_RAG_MEM_LIMIT=512m
CAMPUS_RAG_CPUS=0.5
```

这组默认值是给 2核4G 首发服务器用的：Qdrant 和 RAG 可以工作，但不会无限吃内存。后续知识库明显变大时，优先调高这两个限制。

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
CAMPUS_AI_MODEL=deepseek-v4-flash
CAMPUS_AI_DAILY_LIMIT=200
CAMPUS_EZAI_BOT_USER_ID=
CAMPUS_EZAI_MIN_RAG_CONFIDENCE=0.56
CAMPUS_AI_BUDGET_ENABLED=true
CAMPUS_AI_MONTHLY_BUDGET_CNY=20
CAMPUS_AI_DAILY_BUDGET_CNY=2
CAMPUS_AI_BUDGET_WARN_RATIO=0.7,0.9
CAMPUS_AI_PRICE_INPUT_USD_PER_M=0.14
CAMPUS_AI_PRICE_OUTPUT_USD_PER_M=0.28
CAMPUS_AI_USD_CNY_RATE=7.2
SILICONFLOW_API_KEY=
```

运营值班 Agent：

```bash
CAMPUS_AGENT_INTERNAL_TOKEN=一段随机长token
CAMPUS_AGENT_SERVICE_URL=http://campus-agent:8091
CAMPUS_API_INTERNAL_BASE_URL=http://api:8080/v1
CAMPUS_AGENT_API_KEY=
CAMPUS_AGENT_BASE_URL=
CAMPUS_AGENT_MODEL=deepseek-v4-flash
CAMPUS_AGENT_ENABLED=true
CAMPUS_AGENT_FEISHU_ENABLED=true
CAMPUS_AGENT_DAILY_REPORT_ENABLED=true
CAMPUS_AGENT_DAILY_REPORT_TIME=09:30
CAMPUS_AGENT_HIGH_RISK_NOTIFY_ENABLED=true
CAMPUS_OPS_FEISHU_EVENTS_ENABLED=true
CAMPUS_OPS_FEISHU_REPORT_NOTIFY=true
CAMPUS_OPS_FEISHU_FEEDBACK_NOTIFY=true
CAMPUS_OPS_FEISHU_FEEDBACK_NOTIFY_TYPES=contact,cooperation,bug,content
CAMPUS_AGENT_AUDIT_ENABLED=true
CAMPUS_AGENT_AUDIT_AUTO_PASS_CONFIDENCE=0.9
CAMPUS_AGENT_AUDIT_TIMEOUT=10s
CAMPUS_AI_AUDIT_BATCH_SIZE=2
CAMPUS_AI_AUDIT_TASK_TIMEOUT=10s
CAMPUS_AGENT_RUN_STALE_AFTER=10m
CAMPUS_AGENT_MAX_CONCURRENT_RUNS=1
```

`campus-agent` 承担两类能力：巡检类任务只读，只生成每日巡检、RAG 缺口和治理建议；发帖审核通过 `/internal/moderation/audit` 返回结构化判断。审核链路规则先行：明显低风险帖子自动同步到首页且不调模型，不确定或高风险才调用 Agent；高风险规则不允许被 Agent 洗白，会保留待处理并推飞书确认。生产默认每天 `09:30 Asia/Shanghai` 自动跑一次 `daily_ops` 并发飞书日报；举报和重要反馈会进入 5 秒级飞书提醒队列，举报飞书卡片会带被举报内容摘要、举报原因、举报人和后台入口，举报人会收到站内“已收到”和“处理结果”消息。生产 compose 会把 `campus-agent` 限制在约 `384m / 0.5 CPU`，AI 审核 worker 默认每轮 2 条，避免挤占 API 主链路。

这些环境变量是新库默认值；运营后台 `/admin/audit` 的“值班 Agent 开关”和“AI 成本保护”保存后会写入 `campus_ops_setting`，之后以数据库设置为准，不需要重启容器。若后续模型成本过高，可以在后台关闭 `Agent 模型能力`、只关闭 `AI/Agent 初审`，或调低预算；飞书举报/反馈提醒仍可单独保留。

飞书告警和 Agent 运营通知：

```bash
LEHU_ALERT_ENV=prod
LEHU_ALERT_WEBHOOK_TOKEN=一段随机长token
LEHU_ALERT_WEBHOOK_INTERNAL_URL=http://alert-webhook:9120
LEHU_ALERT_FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/xxx
LEHU_ALERT_FEISHU_SECRET=
GRAFANA_ROOT_URL=https://grafana.example.com
LEHU_ADMIN_ROOT_URL=https://admin.example.com
LEHU_PUBLIC_API_BASE_URL=https://api.example.com/v1
LEHU_FEISHU_CARD_CALLBACK_ENABLED=true
LEHU_FEISHU_CARD_VERIFY_TOKEN=
CAMPUS_OPS_SLA_SCAN_ENABLED=true
CAMPUS_OPS_SLA_REPORT_OVERDUE=30m
CAMPUS_OPS_SLA_AUDIT_OVERDUE=2h
CAMPUS_OPS_SLA_FEISHU_FAILED=10m
```

Grafana 服务健康告警和 Agent 运营通知复用同一个飞书机器人。Grafana 调 `alert-webhook /grafana`，Agent 调 `alert-webhook /agent`。日报和反馈只做提醒和后台跳转；发帖审核卡片可以通过一次性链接“通过/拒绝”，举报卡片可以“下架内容/忽略举报”。真正写库仍由 `campus-api` 校验一次性 token 后完成。举报超过 30 分钟、待审超过 2 小时、飞书发送失败或积压超过 10 分钟时，后台任务会按类型每小时聚合推一次 SLA 提醒；Grafana 的「校园 e站值班 Agent」面板也会显示 Agent 调用、AI 成本、审核决策、飞书队列和 SLA 超时。

真实 IP 和日志保留：

```bash
LEHU_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
LEHU_ACCESS_LOG_RETENTION_DAYS=7
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

第一次启动前先确认云 MySQL 内网地址可连。启动后等 Qdrant、RAG、Grafana 全部起来，再做接口检查。

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
| 值班 Agent | 三种任务可运行，手动发送飞书可用，举报/重要反馈/审核待确认能触发提醒 |
| 朋友圈素材 | 可生成素材包，扫码能进帖子详情 |
| Grafana | 日志搜索和健康监控有数据，`campus_agent_health`、`alert_webhook_health` 为 up |
| 飞书 | 模拟 Grafana 告警和 Agent 通知都能发到群 |

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
| Agent 飞书提醒没收到 | `/admin/audit` 开关、`campus_ops_alert` 状态、`alert-webhook` 日志 |
| Agent 健康 down | `campus-agent` 容器、内部 token、模型配置；主社区链路应继续可用 |
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
