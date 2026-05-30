# 数据模型导览

这份文档帮助人快速理解 `sql/campus.sql` 里的表。它不是完整字段字典，完整建表语句仍以 `sql/campus.sql` 为准。

## 数据库边界

默认数据库：

```text
lehu_campus_db
```

初始化脚本：

```text
sql/campus.sql
```

当前项目只保留校园 e站需要的数据结构：账号、用户、文件、校园社区、通知、审核、安全、e仔/RAG。运行中数据库不自动 drop 历史表。

## 表分组

### 账号与用户

| 表 | 用途 |
| --- | --- |
| `account` | 账号基础表，支持手机号/邮箱等账号体系 |
| `user` | 用户基础信息 |
| `campus_wechat_identity` | 微信 openid/unionid 与用户绑定 |
| `campus_profile` | 校园用户资料、昵称、头像、认证、统计 |
| `campus_operator` | 运营后台权限，`operator/admin` |

### 文件

| 表 | 用途 |
| --- | --- |
| `file` | 文件主表 |
| `file_campus_public_hash_*` | campus/public 文件 hash 分片表 |
| `file_campus_public_id_*` | campus/public 文件 id 分片表 |
| `file_campus_post_media_hash_*` | 帖子媒体 hash 分片表 |
| `file_campus_post_media_id_*` | 帖子媒体 id 分片表 |

文件表只记录文件元数据和状态，真实公开图片在 MinIO 或 COS 里。

### 社区

| 表 | 用途 |
| --- | --- |
| `campus_forum_category` | 版块分类 |
| `campus_forum_post` | 帖子 |
| `campus_forum_comment` | 评论和回复 |
| `campus_forum_post_like` | 帖子点赞 |
| `campus_forum_comment_like` | 评论点赞 |
| `campus_forum_post_collection` | 收藏 |
| `campus_forum_report` | 举报 |

帖子只支持文字和图片。视频字段和视频链路不再作为首发能力。

### 反馈与通知

| 表 | 用途 |
| --- | --- |
| `campus_feedback` | 用户反馈 |
| `campus_notification` | 站内通知 |
| `campus_notification_outbox` | 通知可靠投递任务 |

`outbox` 的意义是先把要发的通知落库，再由后台任务投递，避免业务事务里直接做复杂投递。

### 审核与安全

| 表 | 用途 |
| --- | --- |
| `campus_ops_setting` | 运营配置，例如审核模式、值班 Agent/飞书开关、e仔人设 |
| `campus_ai_audit_task` | AI 发帖审核任务 |
| `campus_ai_usage_log` | 模型调用 token、预估成本和预算保护账本 |
| `campus_audit_log` | 审核记录 |
| `campus_access_log` | API 访问记录 |
| `campus_ip_block` | IP 封禁 |
| `campus_event` | 行为事件，例如访问、发布、互动 |

`campus_access_log` 会按 `LEHU_ACCESS_LOG_RETENTION_DAYS` 定期清理，生产默认 7 天。普通容器日志走 Loki，不进入 MySQL；首发不做双 MySQL 拆库，所有业务表继续使用同一个云 MySQL。

### e仔/RAG

| 表 | 用途 |
| --- | --- |
| `campus_ai_reply_task` | 评论区 `@e仔` 自动回复任务 |
| `campus_knowledge_document` | 知识库文档元数据 |
| `campus_knowledge_chunk` | 知识库切片预览 |
| `campus_rag_query_log` | RAG 查询日志 |
| `campus_rag_eval_case` | RAG 回归评测用例，含 Agent 自动沉淀的停用草稿 |

Qdrant 里也会保存知识库切片向量。MySQL 的 `campus_knowledge_chunk` 更偏后台预览和排查，Qdrant 才是线上语义检索主要索引。

`campus_rag_query_log` 首发继续保存在云 MySQL，用于 e仔回复复盘、知识库命中分析和质量标注。后续如果数据量明显增长，再单独增加 30 到 90 天保留期。

### 课表

| 表 | 用途 |
| --- | --- |
| `campus_timetable_course` | 用户导入的课表课程 |

## 常用状态

帖子状态：

| 值 | 含义 |
| --- | --- |
| `0` | 待审核 |
| `1` | 可见 |
| `2` | 审核拒绝或不可见 |
| `3` | 下架/删除类状态，具体以代码枚举为准 |

知识库文档状态：

| 值 | 含义 |
| --- | --- |
| `draft` | 草稿 |
| `indexing` | 索引中 |
| `active` | 已启用 |
| `disabled` | 已下架 |
| `failed` | 索引失败 |

任务状态：

| 值 | 含义 |
| --- | --- |
| `pending` | 等待处理 |
| `processing` | 处理中 |
| `done` | 已完成 |
| `failed` | 失败 |

## 核心数据流

### 发帖

```text
campus_forum_post
campus_forum_post_like
campus_forum_post_collection
campus_forum_comment
campus_notification_outbox
campus_notification
```

如果审核模式不是 `off`，帖子会先进入待审核，再由人工或 AI 审核变更状态。待审核帖不进入公共列表，但作者本人可以在自己的详情和“我的帖子”看到，小程序用 `publish_state/client_status_label/client_status_detail` 展示成“同步中/需修改”。

### 图片上传

```text
file / file_campus_* -> campus_forum_post.images
```

图片真实文件在 MinIO/COS，数据库保存 file id、URL 和对象信息。

### e仔自动回复

```text
campus_forum_comment -> campus_ai_reply_task -> campus_rag_query_log -> campus_forum_comment
```

用户评论触发任务，后台任务生成 e仔回复，最后回复仍然是一条普通评论。

### 知识库入库

```text
campus_knowledge_document -> campus-rag -> Qdrant -> campus_knowledge_chunk
```

文档元数据先入 MySQL，切片和向量由 `campus-rag` 生成。

## 读表建议

第一次看数据，建议按这个顺序：

1. `campus_profile`：用户是谁。
2. `campus_forum_post`：帖子主体。
3. `campus_forum_comment`：互动内容。
4. `campus_notification`：用户收到什么。
5. `campus_ops_setting`：后台配置，包括 `post_audit_mode`、Agent/飞书开关、AI 预算和 e仔人设。
6. `campus_knowledge_document` 和 `campus_knowledge_chunk`：知识库状态。
7. `campus_ai_usage_log`：模型调用成本是否异常。
8. `campus_access_log`：请求访问记录。

不要直接改业务表状态，优先通过后台操作。必须手动修数据时，先备份单条记录，再改最小字段。
