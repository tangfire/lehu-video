# 校园 e站文档入口

建议按这个顺序读：

1. [开发者导览](developer-guide.md)：给第一次接手项目的人，解释产品边界、目录、服务、数据、核心链路、排障和上线。
2. [架构说明](architecture.md)：更偏运行拓扑、服务分工、核心流程和生产默认值。
3. [微服务边界与技术表达](microservices.md)：解释为什么保留轻量微服务、gRPC/Consul 怎么用、哪些不再继续拆。
4. [简历技术亮点](resume-highlights.md)：把项目沉淀成可以写进简历和面试讲述的工程亮点。
5. [上线与部署手册](deployment-launch.md)：生产环境变量、反向代理、启动、验收、上线当天流程。
6. [媒体存储与 COS/CDN](media-storage.md)：公开图片为什么走 COS + CDN，本地 MinIO 和上传链路怎么工作。
7. [数据模型导览](data-model.md)：按业务模块解释 `sql/campus.sql` 里的核心表。
8. [API 路由地图](api-map.md)：按功能列出小程序、运营后台和内部 RAG 接口。
9. [运营后台使用与功能地图](admin-operations.md)：后台页面、日常运营流程、权限和注意事项。
10. [e仔 AI 与 RAG 知识库设计](ai-rag.md)：解释 e仔人设、自动回复、本地知识库、向量检索、后台测试和降级策略。
11. [RAG 质量评测与优化手册](rag-quality-evaluation.md)：解释真实日志、人工标注、评测集、批量评测和召回解释组成的 AI 工程闭环。
12. [观测与告警](observability-alerting.md)：解释 Grafana、Loki、Alloy、Prometheus、health-exporter 和飞书告警怎么工作、怎么查日志、怎么处理告警。
13. [微信小程序提审与社区规则](wechat-submission.md)：隐私保护指引、社区规范、合法域名和提审验收清单。

如果只想快速知道“这个项目怎么回事”，先读开发者导览。
