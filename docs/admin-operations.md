# 运营后台使用与功能地图

这份文档解释运营后台每个页面是干什么的，以及首发运营时推荐怎么用。

## 入口

本地：

```text
http://localhost:15173/admin
```

生产：

```text
https://admin.example.com/admin
```

后台接口走 API：

```text
/v1/campus/admin/**
```

后台权限来自：

```text
campus_operator
LEHU_CAMPUS_ADMIN_USER_IDS
LEHU_CAMPUS_OPERATOR_USER_IDS
```

生产不要开启 `LEHU_CAMPUS_ADMIN_ALLOW_ALL=true`。

## 页面地图

| 页面 | 路径 | 用途 |
| --- | --- | --- |
| 数据总览 | `/admin` | 今日访问、登录、互动、待处理事项 |
| 内容工作台 | `/admin/posts` | 查帖、审核、置顶、精选、下架、批量操作 |
| 运营发帖 | `/admin/compose` | 运营账号发布攻略、问答、公告 |
| 朋友圈素材 | `/admin/moments` | 生成今日热帖九宫格素材包 |
| 反馈与举报 | `/admin/moderation` | 举报、反馈、评论管理合并入口 |
| 审核设置 | `/admin/audit` | 不审核、人工审核、AI 初审 |
| e仔助手 | `/admin/assistant` | e仔状态、人设、知识库、测试、失败任务 |
| 系统通知 | `/admin/notifications` | 给用户发送站内通知 |
| 安全中心 | `/admin/security` | 请求量、限流、错误、IP 封禁 |
| 用户管理 | `/admin/users` | 用户列表、活跃、风险记录 |
| 权限管理 | `/admin/permissions` | 运营/管理员角色配置 |

旧入口会重定向：

```text
/admin/ai-replies -> /admin/assistant?tab=status
/admin/knowledge -> /admin/assistant?tab=knowledge
/admin/comments -> /admin/moderation?tab=comments
/admin/reports -> /admin/moderation?tab=reports
/admin/feedback -> /admin/moderation?tab=feedback
```

## 推荐日常工作流

每天打开后台后，建议顺序：

1. 看“数据总览”：有没有待处理、今日互动是否异常。
2. 看“反馈与举报”：先处理举报和用户反馈。
3. 看“内容工作台”：处理待审核、下架违规内容、置顶精选优质内容。
4. 看“e仔助手”：确认 e仔任务有没有失败，知识库是否健康。
5. 需要运营动作时，用“运营发帖”补官方内容。
6. 有热帖时，用“朋友圈素材”生成九图包手动发朋友圈。
7. 看“安全中心”：如果限流/错误/IP 异常，再进 Grafana 查日志。

## 审核模式

审核设置有三种：

| 模式 | 含义 | 适合场景 |
| --- | --- | --- |
| 不审核 | 新帖直接展示 | 内测小范围、信任用户 |
| 人工审核 | 新帖进入待审核 | 刚上线、活动期、敏感期 |
| AI 初审 | AI 低风险放行，不确定留人工 | 有模型 key 后降低人工压力 |

AI 初审不是全自动免责。当前设计里，AI 明显低风险才通过，不确定或高风险都留给人工。

## e仔助手

e仔助手有这些 tab：

| tab | 用途 |
| --- | --- |
| 回复状态 | 看模型、bot 账号、今日用量、RAG 健康 |
| 人设设定 | 配置名字、身份、性格、语气、默认回复 |
| 知识库 | 上传/录入资料、启用/下架、重建索引 |
| 知识库测试 | 测一个问题会不会命中知识库 |
| 审核设置 | 快速进入发帖审核策略 |
| 失败任务 | 查看和重试 e仔回复失败任务 |

更细设计见 `docs/ai-rag.md`。

## 朋友圈素材

这个功能不会自动发朋友圈。微信没有给普通微信号/小程序开放后台自动发朋友圈能力。

后台生成的是：

```text
9 张朋友圈图片 + 小程序码 + 推荐文案 + ZIP 下载
```

运营要用 e仔官方微信号手动发朋友圈。可以自动选择当天热帖，也可以手动选择帖子。

## 安全中心

安全中心是运营后台里离排障最近的页面：

- 今日请求、独立 IP、限流次数。
- 错误请求。
- 活跃封禁 IP。
- 手动封禁和解除封禁。

如果看到错误请求变多，下一步不是直接去服务器，而是打开 Grafana 日志搜索，按接口路径或 `request_id` 查。

## 权限建议

首发最简单：

- 你自己是 `admin`。
- 临时运营同学给 `operator`。
- 不熟的人不要给后台权限。

`admin` 可以管理权限；`operator` 做日常运营。具体权限判断以代码为准。

## 不建议做的事

- 不要在生产开启 `LEHU_CAMPUS_ADMIN_ALLOW_ALL`。
- 不要把后台公开给所有人知道的路径且没有 HTTPS。
- 不要在数据库里直接改帖子状态，优先用后台。
- 不要把未确认的学校信息放进知识库。
- 不要打开视频入口。
- 不要打开上传中转 fallback。
