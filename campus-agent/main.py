import json
import os
import time
from typing import Any, Dict, List, Optional, Tuple, TypedDict

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
MODEL = (os.getenv("CAMPUS_AGENT_MODEL") or os.getenv("CAMPUS_AI_MODEL") or "deepseek-v4-flash").strip()
HTTP_TIMEOUT = float(os.getenv("CAMPUS_AGENT_HTTP_TIMEOUT", "12"))
MAX_TOOLS = int(os.getenv("CAMPUS_AGENT_MAX_TOOLS", "6"))
INPUT_PRICE_USD_PER_M = float(os.getenv("CAMPUS_AI_PRICE_INPUT_USD_PER_M", "0.14"))
OUTPUT_PRICE_USD_PER_M = float(os.getenv("CAMPUS_AI_PRICE_OUTPUT_USD_PER_M", "0.28"))
USD_CNY_RATE = float(os.getenv("CAMPUS_AI_USD_CNY_RATE", "7.2"))

app = FastAPI(title=LISTEN_TITLE, version="1.0.0")


class RunRequest(BaseModel):
    run_id: str
    run_type: str
    question: str = ""
    operator_id: str = ""
    model_allowed: bool = True


class ModerationAuditRequest(BaseModel):
    post_id: str = ""
    author_id: str = ""
    title: str = ""
    content: str = ""
    post_type: str = ""
    media_type: str = ""
    image_count: int = 0
    model_allowed: bool = True


class ModelUsage(BaseModel):
    model: str = MODEL
    prompt_tokens: int = 0
    completion_tokens: int = 0
    total_tokens: int = 0
    estimated_cost_usd: float = 0.0
    estimated_cost_cny: float = 0.0


class ModerationAuditResult(BaseModel):
    decision: str = "review"
    confidence: float = 0.5
    risk_level: str = "medium"
    reason: str = "需要人工复核"
    evidence: List[str] = Field(default_factory=list)
    rule_risk_level: str = "medium"
    model_used: bool = False
    model_usage: Optional[ModelUsage] = None
    model_skipped_reason: str = ""


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
    model_allowed: bool
    tool_names: List[str]
    tool_results: List[Dict[str, Any]]
    result: AgentResult
    model_used: bool
    model_usage: Optional[Dict[str, Any]]
    model_skipped_reason: str


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
    if state.get("model_allowed", True):
        result, usage, skipped_reason, attempted = call_model(state.get("run_type", ""), state.get("question", ""), tool_results)
        state["model_used"] = attempted
        state["model_usage"] = usage.model_dump() if usage else None
        state["model_skipped_reason"] = skipped_reason
    else:
        result = None
        state["model_used"] = False
        state["model_usage"] = None
        state["model_skipped_reason"] = "model_skipped_budget"
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
    label = TASK_LABELS.get(run_type, "值班 Agent")
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


def estimate_usage_cost(prompt_tokens: int, completion_tokens: int) -> Tuple[float, float]:
    usd = (prompt_tokens / 1_000_000.0) * INPUT_PRICE_USD_PER_M
    usd += (completion_tokens / 1_000_000.0) * OUTPUT_PRICE_USD_PER_M
    cny = usd * USD_CNY_RATE
    return round(usd, 8), round(cny, 6)


def usage_from_response(data: Dict[str, Any]) -> Optional[ModelUsage]:
    raw = data.get("usage")
    if not isinstance(raw, dict):
        return None
    prompt_tokens = int(raw.get("prompt_tokens") or raw.get("input_tokens") or 0)
    completion_tokens = int(raw.get("completion_tokens") or raw.get("output_tokens") or 0)
    total_tokens = int(raw.get("total_tokens") or (prompt_tokens + completion_tokens))
    estimated_usd, estimated_cny = estimate_usage_cost(prompt_tokens, completion_tokens)
    return ModelUsage(
        model=MODEL,
        prompt_tokens=prompt_tokens,
        completion_tokens=completion_tokens,
        total_tokens=total_tokens,
        estimated_cost_usd=estimated_usd,
        estimated_cost_cny=estimated_cny,
    )


def call_model(run_type: str, question: str, tool_results: List[Dict[str, Any]]) -> Tuple[Optional[AgentResult], Optional[ModelUsage], str, bool]:
    if not API_KEY:
        return None, None, "model_unavailable", False
    prompt = {
        "role": "user",
        "content": (
            "你是校园 e站运营值班 Agent。巡检、RAG 缺口和治理建议只能基于工具结果给建议，"
            "不能声称已经执行删除、封禁、改配置等高风险动作。发帖审核由专用审核接口处理。"
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
        data = resp.json() if resp.content else {}
        usage = usage_from_response(data)
        if resp.status_code >= 400:
            return None, usage, "model_error", True
        content = (((data.get("choices") or [{}])[0].get("message") or {}).get("content") or "")
        parsed = parse_model_json(content)
        if not parsed:
            return None, usage, "invalid_json", True
        return AgentResult(**parsed), usage, "", True
    except Exception:  # noqa: BLE001
        return None, None, "model_error", True


def normalize_moderation_result(data: Dict[str, Any]) -> ModerationAuditResult:
    decision = str(data.get("decision") or "review").strip().lower()
    if decision not in ("pass", "review", "reject"):
        decision = "review"
    risk = str(data.get("risk_level") or "medium").strip().lower()
    if risk not in ("low", "medium", "high"):
        risk = "medium"
    try:
        confidence = float(data.get("confidence", 0.5))
    except (TypeError, ValueError):
        confidence = 0.5
    confidence = max(0.0, min(confidence, 1.0))
    reason = str(data.get("reason") or "需要人工复核").strip()[:120]
    evidence_raw = data.get("evidence") or []
    evidence = []
    if isinstance(evidence_raw, list):
        evidence = [str(item).strip()[:160] for item in evidence_raw if str(item).strip()][:6]
    elif isinstance(evidence_raw, str) and evidence_raw.strip():
        evidence = [evidence_raw.strip()[:160]]
    return ModerationAuditResult(decision=decision, confidence=confidence, risk_level=risk, reason=reason, evidence=evidence)


def rule_moderation(req: ModerationAuditRequest) -> ModerationAuditResult:
    text = f"{req.title}\n{req.content}".lower()
    high_words = ["赌博", "裸聊", "诈骗", "代考", "代课", "身份证", "银行卡", "毒品", "买卖账号", "刷单", "套现"]
    medium_words = ["加微信", "兼职", "引战", "辱骂", "曝光", "挂人", "联系方式", "私聊", "群号", "二维码"]
    for word in high_words:
        if word in text:
            return ModerationAuditResult(
                decision="review",
                confidence=0.72,
                risk_level="high",
                rule_risk_level="high",
                reason=f"疑似包含高风险词：{word}",
                evidence=[f"keyword:{word}"],
            )
    for word in medium_words:
        if word in text:
            return ModerationAuditResult(
                decision="review",
                confidence=0.68,
                risk_level="medium",
                rule_risk_level="medium",
                reason=f"疑似需要人工确认：{word}",
                evidence=[f"keyword:{word}"],
            )
    if len((req.title + req.content).strip()) < 8:
        return ModerationAuditResult(
            decision="review",
            confidence=0.55,
            risk_level="medium",
            rule_risk_level="medium",
            reason="内容过短，语义不够明确",
            evidence=["too_short"],
        )
    return ModerationAuditResult(
        decision="pass",
        confidence=0.96,
        risk_level="low",
        rule_risk_level="low",
        reason="规则未发现明显风险",
        evidence=["rule_low_risk"],
        model_skipped_reason="rule_low_risk",
    )


heuristic_moderation = rule_moderation


def call_moderation_model(req: ModerationAuditRequest) -> Tuple[Optional[ModerationAuditResult], Optional[ModelUsage], str, bool]:
    if not API_KEY:
        return None, None, "model_unavailable", False
    prompt = (
        "你是校园社区内容安全审核 Agent。只输出 JSON，字段为 decision, confidence, risk_level, reason, evidence。"
        "decision 只能是 pass/review/reject；risk_level 只能是 low/medium/high；confidence 是 0 到 1。"
        "低风险正常校园分享给 pass；不确定、可能引战、隐私、广告、交易、联系方式、挂人曝光给 review；明显严重违规给 reject。"
        "第一版 reject 也会交给人工，不要因为可疑就轻易 reject。"
        f"帖子：{json.dumps(req.model_dump(), ensure_ascii=False)}"
    )
    try:
        resp = requests.post(
            BASE_URL,
            headers={"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"},
            json={"model": MODEL, "messages": [{"role": "user", "content": prompt}], "temperature": 0.1, "max_tokens": 360},
            timeout=HTTP_TIMEOUT,
        )
        data = resp.json() if resp.content else {}
        usage = usage_from_response(data)
        if resp.status_code >= 400:
            return None, usage, "model_error", True
        content = (((data.get("choices") or [{}])[0].get("message") or {}).get("content") or "")
        parsed = parse_model_json(content)
        if not parsed:
            return None, usage, "invalid_json", True
        return normalize_moderation_result(parsed), usage, "", True
    except Exception:  # noqa: BLE001
        return None, None, "model_error", True


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
        "model_allowed": req.model_allowed,
    })
    tool_results = state.get("tool_results", [])
    result = state.get("result") or fallback_result(run_type, req.question, tool_results)
    return {
        "run_id": req.run_id,
        "run_type": run_type,
        "framework": "langgraph",
        "model": MODEL,
        "model_used": bool(state.get("model_used", False)),
        "model_usage": state.get("model_usage"),
        "model_skipped_reason": state.get("model_skipped_reason", ""),
        "result": result.model_dump(),
        "tool_trace": build_tool_trace(tool_results),
    }


@app.post("/internal/moderation/audit")
def moderation_audit(req: ModerationAuditRequest, x_campus_agent_token: Optional[str] = Header(default=None)) -> Dict[str, Any]:
    check_token(x_campus_agent_token)
    rule = rule_moderation(req)
    if rule.rule_risk_level == "low":
        return rule.model_dump()
    if not req.model_allowed:
        rule.model_used = False
        rule.model_skipped_reason = "model_skipped_budget"
        return rule.model_dump()
    model_result, usage, skipped_reason, attempted = call_moderation_model(req)
    result = model_result or rule
    result.rule_risk_level = rule.rule_risk_level
    result.model_used = attempted
    result.model_usage = usage
    result.model_skipped_reason = skipped_reason
    if not model_result:
        result.decision = "review"
        result.confidence = min(result.confidence, 0.6)
        result.reason = rule.reason if skipped_reason == "" else f"{rule.reason}；{skipped_reason}"
        if skipped_reason and skipped_reason not in result.evidence:
            result.evidence = (result.evidence or []) + [skipped_reason]
    return result.model_dump()
