# 运营 Copilot Agent 设计

运营 Copilot 是校园 e站的后台只读 Agent。它不面向学生，也不自动执行删帖、封禁、审核通过、改配置等高风险动作。第一版目标是把后台数据、RAG 质量、审核队列和安全状态串起来，给运营同学生成可解释的处理建议。

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
    API -->|日报/高风险/手动发送| Alert[alert-webhook]
    Alert --> Feishu[飞书群机器人]
```

服务边界：

- `campus-agent` 是独立 Python 服务，使用 LangGraph 编排 Agent 工作流。
- `campus-api` 仍是唯一公网 HTTP 入口，负责后台鉴权、运行记录入库和内部工具接口。
- `campus-agent` 不直连 MySQL、Redis、Loki、Prometheus，第一版只通过 `campus-api` 的只读工具取数。
- 飞书通知复用 `alert-webhook`，和 Grafana 告警共用同一个飞书机器人配置，但入口不同：Grafana 用 `/grafana`，Copilot 用 `/agent`。

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
- Agent 工具全部只读。
- Agent 结果只作为建议，页面明确显示“只读分析，运营确认后再处理”。
- 写操作仍然走原后台页面，由人点击确认。
- 飞书消息只放摘要、风险、建议和后台链接，不提供删帖、封禁、审核等交互式按钮。

## 飞书运营闭环

第一版运营闭环是：

```mermaid
flowchart LR
    Timer[每日 09:30 定时任务] --> Run[daily_ops Run]
    Admin[后台手动运行] --> Run
    Run --> Agent[campus-agent 分析]
    Agent --> Save[campus_agent_run]
    Save -->|daily report/high risk/manual| Webhook[alert-webhook /agent]
    Webhook --> Feishu[飞书群]
    Feishu --> Back[回后台处理]
```

触发方式：

| 场景 | 行为 |
| --- | --- |
| 每日巡检 | `campus-api` 后台任务默认每天 `09:30 Asia/Shanghai` 创建一次 `daily_ops`，完成后发送飞书日报 |
| 高风险提醒 | 手动运行完成后如果 `risk_level=high`，自动发送一条高风险提醒 |
| 手动发送 | 运营在 `/admin/copilot` 对任意 `done` 状态运行记录点击“发送到飞书” |

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

# Copilot 飞书通知
CAMPUS_AGENT_FEISHU_ENABLED=true
CAMPUS_AGENT_DAILY_REPORT_ENABLED=true
CAMPUS_AGENT_DAILY_REPORT_TIME=09:30
CAMPUS_AGENT_HIGH_RISK_NOTIFY_ENABLED=true
LEHU_ALERT_WEBHOOK_INTERNAL_URL=http://alert-webhook:9120
LEHU_ALERT_WEBHOOK_TOKEN=change-me-long-random-alert-token
LEHU_ADMIN_ROOT_URL=https://admin.example.com
```

本地如果没有模型 key，Agent 仍会生成规则 fallback 报告，方便开发演示。

本地如果没有配置 `LEHU_ALERT_FEISHU_WEBHOOK`，`alert-webhook` 会返回 `missing_webhook`，Copilot 运行会标记为 `skipped`，不会影响后台使用。

## 排障

| 现象 | 优先看 |
| --- | --- |
| 后台运行 Copilot 失败 | `campus-api` 日志、`campus-agent` 健康状态、`CAMPUS_AGENT_INTERNAL_TOKEN` |
| 工具调用失败 | 运行详情里的 `tool_trace`，以及 `campus-api` 内部工具接口日志 |
| 飞书未收到日报 | `CAMPUS_AGENT_DAILY_REPORT_ENABLED`、`CAMPUS_AGENT_DAILY_REPORT_TIME`、`alert-webhook` 日志 |
| 飞书显示未配置 | `api` 和 `alert-webhook` 容器里的 `LEHU_ALERT_FEISHU_WEBHOOK` 是否都已配置 |
| 飞书链接打不开 | `LEHU_ADMIN_ROOT_URL` 是否是可访问的运营后台 HTTPS 地址 |

## 面试表达

可以这样讲：

> 我在校园 e站里新增了运营 Copilot Agent，独立为 `campus-agent` 微服务，使用 LangGraph 编排 `plan_tools -> call_tools -> generate_report` 工作流，用 LangChain tool 封装只读后台工具接口。Agent 能读取运营统计、审核队列、RAG 查询日志、评测集和安全面板，生成每日巡检、知识库缺口和内容治理建议；结果会记录到 MySQL，并通过复用的飞书告警桥接服务发送每日运营日报和高风险提醒。所有高风险写操作都采用 human-in-the-loop，Agent 只给建议不自动执行。
