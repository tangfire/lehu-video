# 运营值班 Agent 设计

运营值班 Agent 是校园 e站的后台运营自动化服务。它不面向学生，主要目标是减少人工盯后台：定时巡检、举报/重要反馈主动飞书提醒、AI/Agent 发帖初审，以及不确定内容的飞书人工确认闭环。

边界很明确：`campus-agent` 只产出判断、理由和风险等级；真正写库的通过/拒绝、通知作者、审核日志都由 `campus-api` 执行。除低风险高置信帖子可自动通过外，高风险治理动作不自动执行。

## 架构

```mermaid
flowchart LR
    Admin[运营后台] --> API[campus-api]
    API --> Agent[campus-agent]
    Agent -->|internal token| Tools[campus-api 只读工具接口]
    Tools --> MySQL[(MySQL)]
    Tools --> Redis[(Redis缓存)]
    Agent --> Model[OpenAI-compatible 模型]
    API --> Runs[(campus_agent_run)]
    API --> Alerts[(campus_ops_alert)]
    API --> Tokens[(campus_ops_action_token)]
    API -->|日报/高风险/举报/反馈/审核| Alert[alert-webhook]
    Alert --> Feishu[飞书群机器人]
```

服务边界：

- `campus-agent` 是独立 Python 服务，使用 LangGraph 编排 Agent 工作流。
- `campus-api` 仍是唯一公网 HTTP 入口，负责后台鉴权、运行记录入库和内部工具接口。
- `campus-agent` 不直连 MySQL、Redis、Loki、Prometheus；巡检类任务只通过 `campus-api` 的只读工具取数。
- AI/Agent 发帖审核走 `campus-api -> campus-agent /internal/moderation/audit`，Agent 返回 `decision/confidence/risk_level/reason/evidence`。
- 飞书通知复用 `alert-webhook`，和 Grafana 告警共用同一个飞书机器人配置，但入口不同：Grafana 用 `/grafana`，Agent 运营通知用 `/agent`。

## LangGraph 工作流

第一版图很克制：

```mermaid
flowchart LR
    Plan[plan_tools] --> Call[call_tools]
    Call --> Report[generate_report]
    Report --> Done[AgentResult]
```

- `plan_tools`：根据任务类型选择 allowlist 工具，最多 6 个。
- `call_tools`：通过 LangChain tool 调用 `campus-api` 内部只读接口。
- `generate_report`：优先调用模型生成结构化 JSON；模型不可用或 JSON 不合法时，返回规则 fallback 报告。

这不是开放式“任意行动”的 Agent，而是受控运营 Agent。好处是可解释、可排障、权限边界清楚。

## 任务类型

| 类型 | 作用 |
| --- | --- |
| `daily_ops` | 每日运营巡检，汇总社区、审核、e仔、RAG、安全状态 |
| `rag_gap` | 知识库缺口分析，找出错误标注、低置信度和评测失败问题 |
| `moderation_advice` | 内容治理建议，按待审核、举报、反馈、失败任务给优先级 |

输出统一为：

- `summary`
- `risk_level`
- `findings`
- `recommendations`
- `evidence`
- `next_actions`

后台会把 `next_actions` 渲染为跳转按钮，例如去审核、去 e仔回复状态、去 RAG 评测、去安全中心。

## 安全边界

- 对外接口 `/v1/campus/admin/copilot/runs` 需要后台管理员或运营权限。
- 对外接口 `/v1/campus/admin/copilot/runs/{id}/send-feishu` 只允许后台管理员或运营手动发送已完成结果。
- 内部工具接口只接受 `X-Campus-Agent-Token`。
- 巡检、RAG 缺口、治理建议的工具全部只读。
- 发帖审核的写操作只由 `campus-api` 执行，`campus-agent` 不直接写库。
- 飞书审核按钮使用 `campus_ops_action_token` 一次性 token，默认 24 小时过期，token 绑定目标和动作。
- 举报和反馈第一版只飞书提醒并跳后台处理；飞书按钮闭环只用于发帖审核。
- 高风险或低置信审核不自动拒绝，只保留待审核并提醒人工确认。

## AI/Agent 发帖审核

后台“审核设置”里的 `AI/Agent 初审` 开启后，新帖先进入 `campus_ai_audit_task` 队列，由 `campus-api` 后台任务调用 `campus-agent`。

策略：

| Agent 结果 | 系统行为 |
| --- | --- |
| `pass + low + confidence >= 0.85` | `campus-api` 自动设为可见，不打扰作者 |
| `review` 或 `confidence < 0.85` | 保持待审核，生成飞书审批卡片 |
| `reject` 或 `high` | 不自动拒绝，作为高风险待审推飞书 |
| Agent 不可用 | 保持待审核，飞书提醒“审核 Agent 不可用” |

飞书审核卡片包含帖子摘要、风险等级、Agent 理由、后台链接，以及“通过/拒绝”按钮。按钮背后是一次性 token 调用 `campus-api /v1/campus/feishu/card/callback`；如果公网回调或飞书能力不完整，仍可降级为打开后台处理。

小程序端采用“作者可见优先”：待审核帖子不进入公共首页，但作者本人可在详情和“我的帖子”看到；客户端优先展示 `publish_state/client_status_label/client_status_detail`，不要直接展示后台审核原因。

## 飞书运营闭环

第一版运营闭环是：

```mermaid
flowchart LR
    Timer[每日 09:30 定时任务] --> Run[daily_ops Run]
    Admin[后台手动运行] --> Run
    Run --> Agent[campus-agent 分析]
    Agent --> Save[campus_agent_run]
    Event[举报/重要反馈/待人工审核] --> OpsAlert[campus_ops_alert]
    OpsAlert --> Webhook[alert-webhook /agent]
    Save -->|daily report/high risk/manual| Webhook
    Webhook --> Feishu[飞书群]
    Feishu --> Back[回后台/按钮确认]
```

触发方式：

| 场景 | 行为 |
| --- | --- |
| 每日巡检 | `campus-api` 后台任务默认每天 `09:30 Asia/Shanghai` 创建一次 `daily_ops`，完成后发送飞书日报 |
| 高风险提醒 | 手动运行完成后如果 `risk_level=high`，自动发送一条高风险提醒 |
| 手动发送 | 运营在 `/admin/copilot` 对任意 `done` 状态运行记录点击“发送到飞书” |
| 举报提醒 | 用户举报帖子/评论后写入 `campus_ops_alert`，后台任务 5 秒级扫描并推飞书 |
| 重要反馈 | `contact/cooperation/bug/content` 类型即时提醒，普通 `suggestion` 进入日报 |
| 审核确认 | Agent 拿不准的帖子推飞书卡片，可点通过/拒绝或回后台 |

发送失败不会改写 Agent 的分析结果，只会更新运行记录里的飞书状态。后台列表会展示 `pending/sent/failed/skipped`。

`campus_agent_run` 记录这些字段：

| 字段 | 含义 |
| --- | --- |
| `source` | `manual` 或 `scheduled` |
| `feishu_status` | `pending/sent/failed/skipped` |
| `feishu_sent_at` | 成功发送时间 |
| `feishu_error` | 短错误原因，不保存完整飞书响应正文 |

## 配置

```bash
# campus-api 调用 campus-agent 的内网服务地址。
CAMPUS_AGENT_SERVICE_URL=http://campus-agent:8091
CAMPUS_AGENT_INTERNAL_TOKEN=change-me-long-random-agent-token

# campus-agent 调用 campus-api 内部只读工具接口。
CAMPUS_API_INTERNAL_BASE_URL=http://api:8080/v1

# 可选：独立 Agent 模型配置。这里的 BASE_URL 是 OpenAI-compatible 模型接口地址，
# 不要填成 campus-agent 服务地址。
CAMPUS_AGENT_API_KEY=
CAMPUS_AGENT_BASE_URL=
CAMPUS_AGENT_MODEL=

# 未配置独立模型时回退这组
CAMPUS_AI_API_KEY=
CAMPUS_AI_BASE_URL=https://api.deepseek.com/chat/completions
CAMPUS_AI_MODEL=deepseek-chat

# 值班 Agent 飞书通知
CAMPUS_AGENT_FEISHU_ENABLED=true
CAMPUS_AGENT_DAILY_REPORT_ENABLED=true
CAMPUS_AGENT_DAILY_REPORT_TIME=09:30
CAMPUS_AGENT_HIGH_RISK_NOTIFY_ENABLED=true
CAMPUS_OPS_FEISHU_EVENTS_ENABLED=true
CAMPUS_OPS_FEISHU_REPORT_NOTIFY=true
CAMPUS_OPS_FEISHU_FEEDBACK_NOTIFY_TYPES=contact,cooperation,bug,content
CAMPUS_AGENT_AUDIT_AUTO_PASS_CONFIDENCE=0.85
CAMPUS_AI_AUDIT_BATCH_SIZE=2
CAMPUS_AI_AUDIT_TASK_TIMEOUT=10s
CAMPUS_AGENT_RUN_STALE_AFTER=10m
CAMPUS_AGENT_MAX_CONCURRENT_RUNS=1
LEHU_ALERT_WEBHOOK_INTERNAL_URL=http://alert-webhook:9120
LEHU_ALERT_WEBHOOK_TOKEN=change-me-long-random-alert-token
LEHU_PUBLIC_API_BASE_URL=https://api.example.com/v1
LEHU_ADMIN_ROOT_URL=https://admin.example.com
LEHU_FEISHU_CARD_CALLBACK_ENABLED=true
LEHU_FEISHU_CARD_VERIFY_TOKEN=
```

本地如果没有模型 key，Agent 仍会生成规则 fallback 报告，方便开发演示。

本地如果没有配置 `LEHU_ALERT_FEISHU_WEBHOOK`，`alert-webhook` 会返回 `missing_webhook`，Agent 运行会标记为 `skipped`，不会影响后台使用。

## 排障

| 现象 | 优先看 |
| --- | --- |
| 后台运行 Agent 失败 | `campus-api` 日志、`campus-agent` 健康状态、`CAMPUS_AGENT_INTERNAL_TOKEN` |
| 工具调用失败 | 运行详情里的 `tool_trace`，以及 `campus-api` 内部工具接口日志 |
| 飞书未收到日报 | `CAMPUS_AGENT_DAILY_REPORT_ENABLED`、`CAMPUS_AGENT_DAILY_REPORT_TIME`、`alert-webhook` 日志 |
| 举报/反馈没提醒 | `CAMPUS_OPS_FEISHU_EVENTS_ENABLED`、`campus_ops_alert` 状态、`alert-webhook` 日志 |
| 飞书按钮不能处理审核 | `LEHU_PUBLIC_API_BASE_URL` 是否公网 HTTPS、`campus_ops_action_token` 是否过期 |
| 飞书显示未配置 | `api` 和 `alert-webhook` 容器里的 `LEHU_ALERT_FEISHU_WEBHOOK` 是否都已配置 |
| 飞书链接打不开 | `LEHU_ADMIN_ROOT_URL` 是否是可访问的运营后台 HTTPS 地址 |

## 面试表达

可以这样讲：

> 我在校园 e站里做了一个运营值班 Agent，独立为 `campus-agent` 微服务，使用 LangGraph 编排巡检类工具调用，用 LangChain tool 封装只读后台工具接口；同时把发帖 AI 审核迁到 Agent 服务。系统支持每日巡检、RAG 缺口分析、举报/重要反馈飞书主动提醒，以及发帖审核的 human-in-the-loop 闭环：低风险高置信自动通过，不确定或高风险内容推飞书卡片，由运营点击通过/拒绝，真正写库仍由 `campus-api` 完成。
