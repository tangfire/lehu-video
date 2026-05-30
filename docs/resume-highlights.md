# 校园 e站简历技术亮点

这份文档用于把项目整理成简历和面试表达。写简历时不用把所有业务都列出来，重点突出架构、工程能力、上线能力和成本控制。

## 推荐简历项目描述

校园 e站是面向高校学生的微信小程序社区与运营后台，支持校园发帖、图片上传、评论互动、内容审核、通知、e仔 AI/RAG 问答、知识库管理、朋友圈素材生成和可观测性告警。项目采用轻量微服务架构，部署在 Docker Compose 上，兼顾低成本首发和工程完整性。

## 技术栈

```text
Go / Kratos / gRPC / Consul / MySQL / Redis
Python / FastAPI / Qdrant / RAG / Embedding / LangGraph / LangChain
React / Vite / 微信小程序
Docker Compose / COS / CDN
Grafana / Loki / Alloy / Prometheus / 飞书告警
```

## 可写进简历的要点

- 设计并实现轻量微服务架构，拆分 `campus-api`、`base`、`campus-user`、`campus-rag`、`campus-agent`，Go 服务内部使用 gRPC + Consul 服务发现，Python AI/RAG/Agent 服务通过 Docker 内网 HTTP 接入。
- 设计公开媒体上传链路：前端请求预签名 URL，客户端直传 COS/MinIO，后端确认文件并返回 CDN URL，避免轻量服务器带宽被图片流量占满。
- 引入 Redis 热点读缓存和真实 IP 限流，对帖子列表、帖子详情、分类、后台 summary、安全 overview 等接口做短 TTL 缓存，Redis 异常时回落 MySQL。
- 搭建 Grafana + Loki + Alloy + Prometheus 可观测体系，支持按 `request_id` 搜索容器日志、健康面板定位故障组件，并通过飞书群机器人接收 P0/P1 告警。
- 实现 e仔 AI/RAG 知识库链路：后台上传/录入资料，RAG 服务解析切片并写入 Qdrant，评论区 `@e仔` 时结合帖子上下文和知识库生成回复，并在后台支持质量标注和撤回。
- 设计 RAG 质量评测闭环：真实查询日志沉淀为评测集，批量运行固定问题集，记录命中率、平均分、失败样例，并展示 dense/BM25/词面重合等召回解释字段。
- 新增运营值班 Agent：独立 `campus-agent` 服务使用 LangGraph 编排工具调用流程，读取后台统计、审核队列、RAG 日志和安全数据，支持每日巡检、知识库缺口分析、治理建议，以及发帖审核 human-in-the-loop。
- 设计运营提醒队列：举报、重要反馈、SLA 超时和飞书发送失败先写入 `campus_ops_alert`，由后台任务负责去重、退避重试、飞书提醒和处理回执；这条链路不包装成 Agent 推理，避免把普通通知说成智能决策。
- 设计 AI 成本保护链路：发帖审核规则先行但不替代模型，普通帖子异步进入 Agent 初审；高置信低风险自动通过，中高风险进入飞书人工确认；所有模型调用写入 `campus_ai_usage_log`，后台展示日/月预算并在 70%/90% 触发飞书预警。
- 设计运营后台能力，包括内容审核、举报反馈、权限管理、安全面板、e仔人设配置、知识库测试、朋友圈九图素材包和审核策略配置。
- 围绕 300 人试运营做成本控制：2核4G 轻量服务器 + 1核1G 云 MySQL + 本机 Redis，视频关闭，图片走 COS/CDN，访问日志 7 天保留。

## 面试讲述结构

1. 先讲产品：校园社区、小程序、运营后台、e仔知识库。
2. 再讲架构：API 网关 + 基础服务 + 用户资料服务 + RAG 服务，gRPC/Consul/HTTP 如何配合。
3. 再讲关键链路：发帖无感审核、图片上传、e仔回复、Agent 飞书值班、日志排障。
4. 再讲取舍：为什么首发不开放视频，为什么不继续拆帖子/评论服务，为什么用 COS/CDN 和 Redis。
5. 最后讲上线：Docker Compose、生产端口收敛、Grafana/Loki/Prometheus、飞书告警、7 天访问日志保留。

## 面试时可以主动解释的取舍

- 单机 Docker 部署仍可以是微服务，因为服务有独立容器、独立进程、独立健康检查、独立日志和服务间通信。
- 首发没有拆 `forum-service`、`comment-service`、`notification-service`，是为了避免过度工程；这些模块在当前规模下强相关，留在 `campus-api` 更利于事务和排障。
- RAG 独立成 Python 服务，是因为模型、embedding、Qdrant 检索和文档解析与 Go 业务服务技术栈不同。
- RAG 不是只做“向量库问答”，而是有真实日志、人工标注、评测集和批量回归评测，能支撑后续切片策略、阈值、embedding 模型和 reranker 的可控迭代。
- Agent 采用 human-in-the-loop 边界：LangChain tool 主要封装只读接口，发帖审核用一次性 token 做人工确认，删帖/封禁等高风险治理仍回后台执行，兼顾展示 Agent 能力和生产安全。
- 生产公开图片不用本机 MinIO，是因为轻量服务器带宽小，COS/CDN 能把媒体流量从 API 服务器剥离出去。
