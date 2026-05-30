import json
import os
import time
from typing import Any, Dict, List, Optional, TypedDict

import requests
from fastapi import FastAPI, Header, HTTPException
from langchain_core.tools import tool
from langgraph.graph import END, StateGraph
from pydantic import BaseModel, Field


LISTEN_TITLE = "campus-agent"
INTERNAL_TOKEN = os.getenv("CAMPUS_AGENT_INTERNAL_TOKEN", "local-agent-token")
CAMPUS_API_INTERNAL_BASE_URL = os.getenv("CAMPUS_API_INTERNAL_BASE_URL", "http://api:8080/v1").rstrip("/")
API_KEY = os.getenv("CAMPUS_AGENT_API_KEY") or os.getenv("CAMPUS_AI_API_KEY") or os.getenv("DEEPSEEK_API_KEY", "")
BASE_URL = (os.getenv("CAMPUS_AGENT_BASE_URL") or os.getenv("CAMPUS_AI_BASE_URL") or "https://api.deepseek.com/chat/completions").strip()
MODEL = (os.getenv("CAMPUS_AGENT_MODEL") or os.getenv("CAMPUS_AI_MODEL") or "deepseek-chat").strip()
HTTP_TIMEOUT = float(os.getenv("CAMPUS_AGENT_HTTP_TIMEOUT", "12"))
MAX_TOOLS = int(os.getenv("CAMPUS_AGENT_MAX_TOOLS", "6"))

app = FastAPI(title=LISTEN_TITLE, version="1.0.0")


class RunRequest(BaseModel):
    run_id: str
    run_type: str
    question: str = ""
    operator_id: str = ""


class Finding(BaseModel):
    title: str
    detail: str = ""
    severity: str = "low"


class Recommendation(BaseModel):
    title: str
    detail: str = ""
    priority: str = "normal"


class Evidence(BaseModel):
    source: str
    detail: str = ""
    link: str = ""


class NextAction(BaseModel):
    label: str
    path: str


class AgentResult(BaseModel):
    summary: str
    risk_level: str = "low"
    findings: List[Finding] = Field(default_factory=list)
    recommendations: List[Recommendation] = Field(default_factory=list)
    evidence: List[Evidence] = Field(default_factory=list)
    next_actions: List[NextAction] = Field(default_factory=list)


class AgentState(TypedDict, total=False):
    run_id: str
    run_type: str
    question: str
    operator_id: str
    tool_names: List[str]
    tool_results: List[Dict[str, Any]]
    result: AgentResult


TOOLS: Dict[str, Dict[str, str]] = {
    "admin_summary": {"path": "/campus/internal/copilot/tools/admin-summary"},
    "security_overview": {"path": "/campus/internal/copilot/tools/security-overview"},
    "ai_reply_overview": {"path": "/campus/internal/copilot/tools/ai-reply-overview"},
    "ai_reply_failed": {"path": "/campus/internal/copilot/tools/ai-reply-tasks?status=failed&size=10"},
    "rag_bad_logs": {"path": "/campus/internal/copilot/tools/rag-query-logs?quality_label=wrong,needs_fix,unsafe&size=12"},
    "rag_low_confidence": {"path": "/campus/internal/copilot/tools/rag-query-logs?need_knowledge=true&max_confidence=0.52&size=12"},
    "rag_eval_cases": {"path": "/campus/internal/copilot/tools/rag-eval-cases?status=1&size=20"},
    "moderation_posts": {"path": "/campus/internal/copilot/tools/moderation-posts?status=0&size=10"},
    "moderation_comments": {"path": "/campus/internal/copilot/tools/moderation-comments?status=0&size=10"},
    "reports": {"path": "/campus/internal/copilot/tools/reports?status=0&size=10"},
    "feedback": {"path": "/campus/internal/copilot/tools/feedback?status=0&size=10"},
}


@tool("campus_copilot_read_tool")
def campus_copilot_read_tool(name: str, operator_id: str = "") -> Dict[str, Any]:
    """Read one allowlisted campus operation tool by name."""
    if name not in TOOLS:
        return {"tool": name, "ok": False, "error": "tool is not allowlisted"}
    return call_tool(name, operator_id)

TASK_TOOLS: Dict[str, List[str]] = {
    "daily_ops": ["admin_summary", "security_overview", "ai_reply_overview", "rag_bad_logs", "rag_eval_cases"],
    "rag_gap": ["ai_reply_overview", "rag_bad_logs", "rag_low_confidence", "rag_eval_cases"],
    "moderation_advice": ["admin_summary", "moderation_posts", "moderation_comments", "reports", "feedback", "ai_reply_failed"],
}

TASK_LABELS = {
    "daily_ops": "每日运营巡检",
    "rag_gap": "RAG 知识库缺口分析",
    "moderation_advice": "内容治理建议",
}


def check_token(token: Optional[str]) -> None:
    if not INTERNAL_TOKEN or token != INTERNAL_TOKEN:
        raise HTTPException(status_code=401, detail="invalid internal token")


def compact(value: Any, limit: int = 700) -> Any:
    text = json.dumps(value, ensure_ascii=False, default=str)
    if len(text) <= limit:
        return value
    return text[:limit] + "..."


def call_tool(name: str, operator_id: str = "") -> Dict[str, Any]:
    tool = TOOLS[name]
    started = time.time()
    headers = {"X-Campus-Agent-Token": INTERNAL_TOKEN}
    if operator_id:
        headers["X-Campus-Agent-Operator-ID"] = operator_id
    try:
        resp = requests.get(
            CAMPUS_API_INTERNAL_BASE_URL + tool["path"],
            headers=headers,
            timeout=HTTP_TIMEOUT,
        )
        duration_ms = int((time.time() - started) * 1000)
        if resp.status_code >= 400:
            return {"tool": name, "ok": False, "duration_ms": duration_ms, "error": f"status={resp.status_code} {resp.text[:200]}"}
        data = resp.json()
        return {"tool": name, "ok": True, "duration_ms": duration_ms, "data": data}
    except Exception as exc:  # noqa: BLE001
        return {"tool": name, "ok": False, "duration_ms": int((time.time() - started) * 1000), "error": str(exc)}


def plan_tools_node(state: AgentState) -> AgentState:
    run_type = state.get("run_type", "")
    state["tool_names"] = TASK_TOOLS.get(run_type, [])[:MAX_TOOLS]
    return state


def call_tools_node(state: AgentState) -> AgentState:
    results = []
    operator_id = state.get("operator_id", "")
    for name in state.get("tool_names", []):
        results.append(campus_copilot_read_tool.invoke({"name": name, "operator_id": operator_id}))
    state["tool_results"] = results
    return state


def generate_report_node(state: AgentState) -> AgentState:
    tool_results = state.get("tool_results", [])
    result = call_model(state.get("run_type", ""), state.get("question", ""), tool_results)
    state["result"] = result or fallback_result(state.get("run_type", ""), state.get("question", ""), tool_results)
    return state


def build_graph():
    graph = StateGraph(AgentState)
    graph.add_node("plan_tools", plan_tools_node)
    graph.add_node("call_tools", call_tools_node)
    graph.add_node("generate_report", generate_report_node)
    graph.set_entry_point("plan_tools")
    graph.add_edge("plan_tools", "call_tools")
    graph.add_edge("call_tools", "generate_report")
    graph.add_edge("generate_report", END)
    return graph.compile()


COPILOT_GRAPH = build_graph()


def build_tool_trace(tool_results: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    trace = []
    for item in tool_results:
        trace.append({
            "tool": item.get("tool"),
            "ok": bool(item.get("ok")),
            "duration_ms": item.get("duration_ms", 0),
            "summary": compact(item.get("data") if item.get("ok") else item.get("error"), 420),
        })
    return trace


def number(data: Dict[str, Any], key: str) -> int:
    try:
        return int(data.get(key) or 0)
    except (TypeError, ValueError):
        return 0


def data_by_tool(tool_results: List[Dict[str, Any]], name: str) -> Dict[str, Any]:
    for item in tool_results:
        if item.get("tool") == name and item.get("ok") and isinstance(item.get("data"), dict):
            return item["data"].get("data") or item["data"]
    return {}


def fallback_result(run_type: str, question: str, tool_results: List[Dict[str, Any]]) -> AgentResult:
    summary_data = data_by_tool(tool_results, "admin_summary").get("summary") or data_by_tool(tool_results, "admin_summary")
    security = data_by_tool(tool_results, "security_overview").get("security") or {}
    ai = data_by_tool(tool_results, "ai_reply_overview")
    ai = ai.get("overview") or ai
    bad_logs = data_by_tool(tool_results, "rag_bad_logs").get("logs") or []
    low_logs = data_by_tool(tool_results, "rag_low_confidence").get("logs") or []
    reports = data_by_tool(tool_results, "reports").get("reports") or []
    posts = data_by_tool(tool_results, "moderation_posts").get("posts") or []
    comments = data_by_tool(tool_results, "moderation_comments").get("comments") or []

    findings: List[Finding] = []
    recommendations: List[Recommendation] = []
    evidence: List[Evidence] = []
    next_actions: List[NextAction] = []

    pending_total = number(summary_data, "pending_reports") + number(summary_data, "pending_feedback") + number(summary_data, "pending_posts") + number(summary_data, "pending_comments")
    if pending_total:
        findings.append(Finding(title="存在运营待办", detail=f"当前待处理内容约 {pending_total} 条", severity="medium"))
        recommendations.append(Recommendation(title="优先清理待审核和举报", detail="先处理举报，再处理待审核帖子和评论。", priority="high"))
        next_actions.append(NextAction(label="去反馈与举报", path="/admin/moderation"))
    if number(ai, "failed"):
        findings.append(Finding(title="e仔回复失败需要复盘", detail=f"失败任务 {number(ai, 'failed')} 条", severity="medium"))
        recommendations.append(Recommendation(title="查看失败回复原因", detail="优先处理模型/RAG 配置错误和高频问题。", priority="high"))
        next_actions.append(NextAction(label="去e仔回复状态", path="/admin/assistant?tab=failed"))
    if bad_logs or low_logs:
        findings.append(Finding(title="RAG 存在可优化问题", detail=f"质量异常 {len(bad_logs)} 条，低置信度 {len(low_logs)} 条", severity="medium"))
        recommendations.append(Recommendation(title="把错误问题沉淀进评测集", detail="补充对应校园资料后运行 RAG 评测。", priority="high"))
        next_actions.append(NextAction(label="去RAG评测", path="/admin/assistant?tab=eval"))
    if number(security, "today_errors") or number(security, "today_rate_limited"):
        findings.append(Finding(title="存在异常请求", detail=f"错误 {number(security, 'today_errors')} 次，限流 {number(security, 'today_rate_limited')} 次", severity="medium"))
        recommendations.append(Recommendation(title="检查安全中心 Top IP/Path", detail="确认是否是正常调试、接口报错或异常访问。", priority="normal"))
        next_actions.append(NextAction(label="去安全中心", path="/admin/security"))

    if run_type == "rag_gap" and not (bad_logs or low_logs):
        findings.append(Finding(title="暂未发现明显 RAG 缺口", detail="最近异常标注和低置信度问题较少。"))
    if run_type == "moderation_advice":
        evidence.append(Evidence(source="待治理队列", detail=f"待审帖子 {len(posts)} 条，待审评论 {len(comments)} 条，举报 {len(reports)} 条"))
    evidence.append(Evidence(source="工具调用", detail=f"已读取 {sum(1 for item in tool_results if item.get('ok'))}/{len(tool_results)} 个只读工具"))

    if not findings:
        findings.append(Finding(title="整体状态平稳", detail="未发现明显高风险运营事项。"))
        recommendations.append(Recommendation(title="继续准备种子内容", detail="可以补充官方攻略、FAQ 和高频问题资料。"))
        next_actions.append(NextAction(label="去运营发帖", path="/admin/compose"))

    risk = "high" if any(item.severity == "high" for item in findings) else "medium" if any(item.severity == "medium" for item in findings) else "low"
    label = TASK_LABELS.get(run_type, "运营 Copilot")
    focus = f"；关注点：{question.strip()}" if question.strip() else ""
    return AgentResult(
        summary=f"{label}完成，风险等级 {risk}{focus}。",
        risk_level=risk,
        findings=findings[:6],
        recommendations=recommendations[:6],
        evidence=evidence[:8],
        next_actions=next_actions[:6],
    )


def parse_model_json(text: str) -> Optional[Dict[str, Any]]:
    text = (text or "").strip()
    if not text:
        return None
    if text.startswith("```"):
        text = text.strip("`")
        text = text.replace("json\n", "", 1).strip()
    start = text.find("{")
    end = text.rfind("}")
    if start >= 0 and end > start:
        text = text[start : end + 1]
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return None


def call_model(run_type: str, question: str, tool_results: List[Dict[str, Any]]) -> Optional[AgentResult]:
    if not API_KEY:
        return None
    prompt = {
        "role": "user",
        "content": (
            "你是校园 e站运营 Copilot，只能基于工具结果给只读建议，不能声称已执行删除、审核、封禁、改配置等操作。"
            "请输出 JSON：summary,risk_level,findings,recommendations,evidence,next_actions。"
            f"任务类型：{run_type}；运营关注点：{question or '无'}；工具结果："
            + json.dumps(build_tool_trace(tool_results), ensure_ascii=False)
        ),
    }
    try:
        resp = requests.post(
            BASE_URL,
            headers={"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"},
            json={"model": MODEL, "messages": [prompt], "temperature": 0.2, "max_tokens": 900},
            timeout=HTTP_TIMEOUT,
        )
        if resp.status_code >= 400:
            return None
        content = (((resp.json().get("choices") or [{}])[0].get("message") or {}).get("content") or "")
        parsed = parse_model_json(content)
        if not parsed:
            return None
        return AgentResult(**parsed)
    except Exception:  # noqa: BLE001
        return None


@app.get("/healthz")
def healthz() -> Dict[str, Any]:
    return {"status": "ok", "model": MODEL, "model_configured": bool(API_KEY)}


@app.post("/internal/copilot/run")
def run_copilot(req: RunRequest, x_campus_agent_token: Optional[str] = Header(default=None)) -> Dict[str, Any]:
    check_token(x_campus_agent_token)
    run_type = req.run_type.strip()
    if run_type not in TASK_TOOLS:
        raise HTTPException(status_code=400, detail="unsupported run_type")
    state = COPILOT_GRAPH.invoke({
        "run_id": req.run_id,
        "run_type": run_type,
        "question": req.question,
        "operator_id": req.operator_id,
    })
    tool_results = state.get("tool_results", [])
    result = state.get("result") or fallback_result(run_type, req.question, tool_results)
    return {
        "run_id": req.run_id,
        "run_type": run_type,
        "framework": "langgraph",
        "model": MODEL,
        "result": result.model_dump(),
        "tool_trace": build_tool_trace(tool_results),
    }
