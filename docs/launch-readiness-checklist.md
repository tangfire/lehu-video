# 上线准备与验收清单

这份文档用于真正上线前逐项确认。它不解释系统设计，只回答：现在能不能开给第一批用户用。

建议在正式开放前按顺序勾完。没勾完的项不要硬上线，尤其是生产密钥、微信域名、COS/CDN、管理员权限和飞书告警。

## 结论标准

可以上线的标准：

- 核心小程序流程可用：登录、看帖、发帖、图片、评论、互动、举报、反馈、通知。
- 运营后台可用：登录、内容管理、审核设置、举报反馈、e仔、知识库、权限。
- 生产配置真实：没有 `change-me`、`example.com`、mock 登录、admin allow all。
- 公开媒体走 COS + CDN，不走 API 服务器中转。
- Grafana 和飞书都可用，出问题能定位。
- 发布脚本和回滚路径可用。

## 生产环境变量

复制示例文件：

```bash
cp .env.production.example .env.production
```

必须替换：

| 配置 | 要求 |
| --- | --- |
| `LEHU_MYSQL_DSN` | 云 MySQL 内网地址和业务账号 |
| `REDIS_PASSWORD` | 真实强密码 |
| `LEHU_JWT_SECRET` | 随机长密钥 |
| `GRAFANA_ADMIN_PASSWORD` | 真实强密码 |
| `WECHAT_APP_ID / WECHAT_APP_SECRET` | 真实小程序配置 |
| `COS_SECRET_ID / COS_SECRET_KEY` | 腾讯云 COS 密钥 |
| `COS_BUCKET / COS_REGION` | 真实 bucket |
| `COS_PUBLIC_CDN_BASE_URL` | 真实 CDN HTTPS 域名 |
| `LEHU_CAMPUS_ADMIN_USER_IDS` | 你的管理员用户 ID |
| `CAMPUS_AGENT_INTERNAL_TOKEN` | 随机长 token |
| `LEHU_ALERT_WEBHOOK_TOKEN` | 随机长 token |
| `LEHU_ALERT_FEISHU_WEBHOOK` | 飞书群机器人 webhook |
| `LEHU_FEISHU_CARD_VERIFY_TOKEN` | 飞书按钮回调校验 token；开启回调时不能为空 |
| `ADMIN_API_BASE_URL` | 真实 API HTTPS 地址 |
| `LEHU_PUBLIC_API_BASE_URL` | 真实 API HTTPS 地址，给飞书回调用 |
| `LEHU_ADMIN_ROOT_URL` | 真实后台 HTTPS 地址 |
| `GRAFANA_ROOT_URL` | 真实 Grafana HTTPS 地址 |

必须确认：

```text
LEHU_CAMPUS_ADMIN_ALLOW_ALL=false
LEHU_WECHAT_MOCK_LOGIN=false
LEHU_STORAGE_PROVIDER=cos
LEHU_ENABLE_LEGACY_UPLOAD=false
LEHU_ACCESS_LOG_RETENTION_DAYS=7
LEHU_REDIS_CACHE_ENABLED=true
```

检查命令：

```bash
docker compose --env-file .env.production -f docker-compose.yml -f docker-compose.prod.yml config >/tmp/campus-prod.yml
```

## 云资源

### 云 MySQL

- 云 MySQL 与应用服务器同地域。
- 3306 不开放公网。
- 安全组只允许应用服务器访问。
- 数据库名为 `lehu_campus_db`。
- 已初始化 `sql/campus.sql`。
- 至少确认云 MySQL 自带基础备份或快照能力。

### Redis

- Redis 继续在应用服务器 Docker 内。
- 设置 `REDIS_PASSWORD`。
- 不开放公网。
- 用于真实 IP 限流和短 TTL 热点缓存。

### COS + CDN

- COS bucket 已创建。
- CDN 域名已绑定。
- CDN 已开启 HTTPS。
- COS CORS 允许小程序直传。
- CDN 回源 COS 正常。
- 图片缓存规则已配置。
- 基础防盗刷策略已配置。

小程序需要的域名：

| 微信配置项 | 域名 |
| --- | --- |
| request 合法域名 | API 域名 |
| uploadFile 合法域名 | COS 上传域名 |
| downloadFile 合法域名 | CDN 下载域名 |

## 反向代理和端口

生产 compose 只把这些服务绑定到宿主机 loopback：

| 服务 | 本机端口 | 对外域名 |
| --- | --- | --- |
| API | `127.0.0.1:18080` | `https://api.example.com` |
| 运营后台 | `127.0.0.1:15173` | `https://admin.example.com` |
| Grafana | `127.0.0.1:13002` | `https://grafana.example.com` |

反向代理的作用：

- 给微信小程序和后台提供 HTTPS 域名。
- 不把 Docker 端口直接暴露公网。
- 统一申请和续期证书。
- 后续蓝绿发布时通过切 upstream 实现用户无感。

必须确认：

- 外部只通过 HTTPS 访问。
- 生产默认不启动本地 MySQL/MinIO；Redis、Consul、Qdrant、Prometheus 不暴露公网。
- API 反代显式拒绝 `/v1/campus/internal/*`，公网访问 `/v1/campus/internal/ops-metrics` 返回 404/403。
- Grafana 使用强密码。

## GitHub Actions 部署

仓库 Secrets：

| Secret | 要求 |
| --- | --- |
| `DEPLOY_HOST` | 服务器 IP 或域名 |
| `DEPLOY_PORT` | SSH 端口 |
| `DEPLOY_USER` | 部署用户 |
| `DEPLOY_SSH_KEY` | 部署私钥 |
| `DEPLOY_PATH` | 服务器项目路径 |

服务器准备：

- `DEPLOY_PATH` 已 clone 仓库。
- 当前分支为 `campus-estation-cleanup`。
- `.env.production` 已存在且不提交 Git。
- 部署用户可执行 `docker compose`。
- 服务器能 `git fetch origin campus-estation-cleanup`。

本地或服务器发布前检查：

```bash
bash scripts/release-check.sh
```

这条检查会拦截真实 `.env.production` 里的占位域名/密钥、mock 登录、admin allow all、旧图片中转上传，并把 Go、运营后台、RAG、Agent 测试都纳入上线门槛。
Python 单测固定使用 3.12；本机没装 `python3.12` 时会通过 Docker 的 `python:3.12-slim` 跑。

服务器运行中健康检查：

```bash
RUN_HEALTH_CHECK=1 RUN_GO_TESTS=0 RUN_ADMIN_BUILD=0 RUN_PYTHON_TESTS=0 bash scripts/release-check.sh
```

## 微信小程序提审

必须确认：

- 小程序 AppID 与后端 `WECHAT_APP_ID` 一致。
- `LEHU_WECHAT_MOCK_LOGIN=false`。
- 微信合法域名已配置。
- 隐私保护指引已填写真实内容。
- 协议和社区规范入口可访问。
- 隐私说明覆盖头像、相册/相机、网络请求、反馈、内容发布。
- 小程序端没有视频发布入口。
- 发布页图片上传失败时能展示 request_id。

参考：`docs/wechat-submission.md`。

## 运营后台初始化

必须完成：

- 用真实微信登录一次，生成你的用户 ID。
- 配置 `LEHU_CAMPUS_ADMIN_USER_IDS`。
- 后台可登录。
- 权限管理可看到管理员账号。
- 审核模式确认。
- Agent 开关确认。
- AI 预算确认。
- 飞书运营通知开关确认。

推荐首发设置：

| 项 | 建议 |
| --- | --- |
| 发帖审核模式 | `AI/Agent 初审` 或 `人工审核` |
| Agent 模型能力 | 开启，预算控制开启 |
| AI 初审 | 开启；成本高时可关 |
| 飞书运营通知 | 开启 |
| 举报提醒 | 开启 |
| 重要反馈提醒 | 开启 |
| AI 月预算 | 默认 20 元 |
| AI 日预算 | 默认 2 元 |

## e仔与知识库

上线前至少准备：

- e仔官方 bot 用户 ID。
- e仔人设：身份、语气、安全边界、默认回复。
- 基础知识库资料：校区介绍、宿舍、校园网、快递、交通、报到/常见问题。
- 知识库测试能命中资料。
- 人设预览能生成一条可接受回复。
- 低置信度问题不会乱答。

建议首发先少量高质量资料，不要把未确认的传言塞进知识库。

## 飞书和监控

必须测试：

- Grafana 告警能进飞书。
- Agent 手动发送能进飞书。
- 用户举报能进飞书。
- `contact/cooperation/bug/content` 反馈能进飞书。
- 飞书按钮下架/忽略能回调 API。
- Grafana 健康面板有数据。
- Grafana「校园 e站值班 Agent」面板有数据，Prometheus target `campus-api-ops` 为 up。
- Loki 能按 `request_id` 查日志。
- 公网 `https://api.example.com/v1/campus/internal/ops-metrics` 返回 404/403。

健康面板必须看到这些目标：

```text
api_health
api_ready
base_health
campus_user_health
campus_rag_health
campus_agent_health
alert_webhook_health
redis_tcp
qdrant_tcp
consul_tcp
```

生产健康目标默认不包含本地 `mysql_tcp` 和 `minio_health`；云 MySQL 可用性先看 `api_ready`，细节看云厂商监控。

注意：`alert-webhook` 自己挂掉时，Grafana 能看到 down，但飞书可能收不到这条通知。后续可用腾讯云云监控做外部兜底。

## 小程序核心 smoke

学生端：

- 微信真实登录成功。
- 首页帖子列表正常。
- 分类正常。
- 发文字帖成功。
- 发图片帖成功，图片 URL 为 CDN 域名。
- 视频帖被拒绝。
- 作者发帖后可看到自己的同步中帖子。
- 公共首页不展示未通过内容。
- 评论成功。
- 点赞、收藏成功。
- 举报帖子成功。
- 提交联系我们/反馈成功。
- 通知中心能看到系统消息。
- `@e仔` 不阻塞评论发布。

运营端：

- 后台登录成功。
- 数据总览有数据。
- 内容工作台可查帖。
- 审核设置可保存。
- 反馈举报可处理。
- e仔人设可保存。
- 知识库测试可用。
- RAG 评测页可打开。
- 朋友圈素材可生成。
- 安全中心可查看请求数据。
- 权限页可用。

## 上线当天流程

1. 确认 `.env.production`。
2. 确认云 MySQL、COS、CDN、微信域名。
3. 确认 GitHub Secrets。
4. 运行 `bash scripts/release-check.sh`。
5. 合并到 `campus-estation-cleanup` 或手动触发 workflow。
6. 等 GitHub Actions 部署完成。
7. 打开 API、后台、Grafana。
8. 跑小程序核心 smoke。
9. 发测试举报和测试反馈。
10. 看飞书和 Grafana。
11. 开放第一批用户。

## Go / No-Go

可以开放：

- 上面核心 smoke 全部通过。
- Grafana 健康目标全绿或只有明确可接受的非核心 warning。
- 飞书能收到技术告警和运营提醒。
- 发布后 10 分钟内 API 无明显 5xx。
- 后台能随时下架内容和处理举报。

暂缓开放：

- 微信真实登录失败。
- 图片上传失败。
- 后台无法登录。
- 举报无法处理。
- Grafana/Loki 没数据。
- MySQL/Redis 健康异常。
- `.env.production` 仍有占位符。

## 上线后第一天观察

重点看：

- MySQL CPU、连接数和慢查询。
- Redis 内存。
- API 错误日志。
- 上传失败。
- 举报和反馈。
- e仔错误回复。
- AI 今日成本。
- `campus_access_log` 清理是否正常。

第一天不要频繁加功能，只修影响登录、发帖、上传、审核、举报和后台处理的问题。
