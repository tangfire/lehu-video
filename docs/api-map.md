# API 路由地图

这份文档按功能列出当前 `campus-api` 的主要 HTTP 路由。真实注册位置在 `app/campusApi/service/internal/service/campusservice.go`。

## 基本约定

业务接口前缀：

```text
/v1
```

认证：

- 标注“公开”的接口可以不登录。
- 标注“用户”的接口需要登录。
- 标注“运营”的接口需要后台运营权限。

响应里如果出错，一般会包含 `request_id` 或 `requestId`，用户报错时优先复制这个编号到 Grafana 搜日志。

## 登录与用户

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/v1/auth/wechat-login` | 公开 | 微信登录 |
| `GET` | `/v1/campus/profile` | 用户 | 当前用户资料 |
| `PUT` | `/v1/campus/profile` | 用户 | 更新资料 |
| `PUT` | `/v1/campus/me/avatar` | 用户 | 更新头像 |
| `GET` | `/v1/campus/users/{id}` | 公开 | 公开用户主页 |
| `GET` | `/v1/campus/users/{id}/posts` | 公开 | 用户公开帖子 |

后台账号密码登录走 Kratos 用户服务相关接口，不在 `RegisterRoutes` 这段手写校园路由里。

## 课表与事件

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/v1/campus/timetable` | 用户 | 课表列表 |
| `POST` | `/v1/campus/timetable/import` | 用户 | 导入课表 |
| `POST` | `/v1/campus/analytics/track` | 公开 | 行为埋点 |

## 上传

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/v1/campus/upload/presign` | 用户 | 获取直传签名 |
| `POST` | `/v1/campus/upload/complete` | 用户 | 确认上传完成 |
| `POST` | `/v1/campus/upload/image` | 用户 | 旧中转上传，生产关闭 |

生产公开图片走 COS + CDN，详见 `docs/media-storage.md`。

## 社区帖子

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/v1/campus/forum/categories` | 公开 | 版块分类 |
| `GET` | `/v1/campus/forum/posts` | 公开 | 帖子列表 |
| `POST` | `/v1/campus/forum/posts` | 用户 | 发帖 |
| `GET` | `/v1/campus/forum/my-posts` | 用户 | 我的帖子 |
| `GET` | `/v1/campus/forum/my-collections` | 用户 | 我的收藏 |
| `GET` | `/v1/campus/forum/my-comments` | 用户 | 我的评论 |
| `GET` | `/v1/campus/forum/posts/{id}` | 公开 | 帖子详情 |
| `DELETE` | `/v1/campus/forum/posts/{id}` | 用户 | 删除自己的帖子 |

帖子只支持文字和图片，不支持视频。

帖子响应保留后台字段 `status/audit_reason`，同时给小程序提供 `publish_state/public_visible/client_status_label/client_status_detail`。公共列表和他人主页只返回公开可见帖；作者本人访问详情或“我的帖子”时可以看到自己的同步中/需修改内容。

## 评论、点赞、收藏、举报

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/v1/campus/forum/posts/{id}/comments` | 公开 | 评论列表 |
| `POST` | `/v1/campus/forum/posts/{id}/comments` | 用户 | 发表评论 |
| `POST` | `/v1/campus/forum/posts/{id}/like` | 用户 | 点赞帖子 |
| `DELETE` | `/v1/campus/forum/posts/{id}/like` | 用户 | 取消点赞 |
| `POST` | `/v1/campus/forum/posts/{id}/collection` | 用户 | 收藏 |
| `DELETE` | `/v1/campus/forum/posts/{id}/collection` | 用户 | 取消收藏 |
| `POST` | `/v1/campus/forum/posts/{id}/report` | 用户 | 举报帖子 |
| `GET` | `/v1/campus/forum/comments/{id}/replies` | 公开 | 评论回复 |
| `POST` | `/v1/campus/forum/comments/{id}/like` | 用户 | 点赞评论 |
| `DELETE` | `/v1/campus/forum/comments/{id}/like` | 用户 | 取消点赞评论 |
| `DELETE` | `/v1/campus/forum/comments/{id}` | 用户 | 删除自己的评论 |
| `POST` | `/v1/campus/forum/comments/{id}/report` | 用户 | 举报评论 |

评论里 `@e仔` 会触发 e仔回复任务，详见 `docs/ai-rag.md`。

## 反馈与通知

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/v1/campus/feedback` | 用户 | 提交反馈 |
| `GET` | `/v1/campus/notifications` | 用户 | 通知列表 |
| `GET` | `/v1/campus/notifications/unread-count` | 用户 | 未读数 |
| `POST` | `/v1/campus/notifications/read-all` | 用户 | 全部已读 |
| `POST` | `/v1/campus/notifications/{id}/read` | 用户 | 单条已读 |

## 用户审核入口

| 方法 | 路径 | 权限 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/v1/campus/moderation/posts` | 用户/审核权限 | 待审核帖子 |
| `GET` | `/v1/campus/moderation/comments` | 用户/审核权限 | 待审核评论 |
| `POST` | `/v1/campus/moderation/posts/{id}/review` | 用户/审核权限 | 审核帖子 |
| `POST` | `/v1/campus/moderation/comments/{id}/review` | 用户/审核权限 | 审核评论 |

运营后台主要使用 `/v1/campus/admin/**`。

## 运营后台

### 总览和设置

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/v1/campus/admin/summary` | 后台数据总览 |
| `GET` | `/v1/campus/admin/settings/audit` | 获取审核设置 |
| `PUT` | `/v1/campus/admin/settings/audit` | 保存审核设置 |
| `POST` | `/v1/campus/admin/stats/reconcile` | 统计重算 |

### 内容管理

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/v1/campus/admin/posts` | 帖子列表 |
| `POST` | `/v1/campus/admin/posts` | 运营发帖 |
| `POST` | `/v1/campus/admin/posts/batch` | 批量操作 |
| `PUT` | `/v1/campus/admin/posts/{id}` | 更新帖子 |
| `DELETE` | `/v1/campus/admin/posts/{id}` | 删除/下架帖子 |
| `GET` | `/v1/campus/admin/comments` | 评论列表 |
| `DELETE` | `/v1/campus/admin/comments/{id}` | 删除评论 |

### 朋友圈素材

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/v1/campus/admin/moments/candidates` | 九图候选帖子 |
| `POST` | `/v1/campus/admin/moments/packages` | 生成素材包 |
| `GET` | `/v1/campus/admin/moments/packages/{id}/images/{slot}.png` | 单图预览 |
| `GET` | `/v1/campus/admin/moments/packages/{id}/download.zip` | ZIP 下载 |

### e仔与知识库

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/v1/campus/admin/ai-replies/summary` | e仔状态 |
| `GET` | `/v1/campus/admin/ai-replies/tasks` | e仔任务 |
| `POST` | `/v1/campus/admin/ai-replies/tasks/{id}/retry` | 重试任务 |
| `GET` | `/v1/campus/admin/ezai/persona` | 获取人设 |
| `PUT` | `/v1/campus/admin/ezai/persona` | 保存人设 |
| `POST` | `/v1/campus/admin/ezai/persona/preview` | 预览 e仔回复 |
| `GET` | `/v1/campus/admin/knowledge/documents` | 知识文档 |
| `POST` | `/v1/campus/admin/knowledge/documents` | 创建知识文档 |
| `PUT` | `/v1/campus/admin/knowledge/documents/{id}` | 更新知识文档 |
| `POST` | `/v1/campus/admin/knowledge/documents/{id}/reindex` | 重建索引 |
| `GET` | `/v1/campus/admin/knowledge/documents/{id}/chunks` | 切片列表 |
| `POST` | `/v1/campus/admin/knowledge/test-query` | 知识库测试 |
| `GET` | `/v1/campus/admin/knowledge/query-logs` | RAG 查询日志 |
| `POST` | `/v1/campus/admin/knowledge/upload` | 上传知识库文件 |

### 风险、反馈、安全、用户

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/v1/campus/admin/reports` | 举报列表 |
| `POST` | `/v1/campus/admin/reports/{id}/review` | 处理举报 |
| `GET` | `/v1/campus/admin/feedback` | 用户反馈 |
| `POST` | `/v1/campus/admin/feedback/{id}/review` | 处理反馈 |
| `GET` | `/v1/campus/admin/security` | 安全概览 |
| `POST` | `/v1/campus/admin/security/ip-blocks` | 封禁 IP |
| `DELETE` | `/v1/campus/admin/security/ip-blocks/{id}` | 解除封禁 |
| `GET` | `/v1/campus/admin/users` | 用户列表 |
| `PUT` | `/v1/campus/admin/users/{id}/role` | 更新用户角色 |
| `POST` | `/v1/campus/admin/notifications` | 创建系统通知 |

## 内部 RAG 接口

这些只在 Docker 内网使用，不给前端直接调用：

| 方法 | 路径 | 服务 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/healthz` | `campus-rag` | RAG 健康 |
| `POST` | `/internal/rag/index-text` | `campus-rag` | 文本入库 |
| `POST` | `/internal/rag/index-document` | `campus-rag` | 文件入库 |
| `POST` | `/internal/rag/delete-document` | `campus-rag` | 删除文档切片 |
| `POST` | `/internal/rag/query` | `campus-rag` | 检索知识库 |

## 已移除的旧链路

当前项目不再提供旧短视频、IM chat、Kafka、WebSocket 路由。看到 `/v1/video/**`、`/ws`、chat/group/friend 之类接口请求，应视为旧客户端或错误流量。
