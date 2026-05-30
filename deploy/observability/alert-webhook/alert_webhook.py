#!/usr/bin/env python3
import base64
import hashlib
import hmac
import json
import os
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse


MAX_ALERTS_IN_MESSAGE = 5
MAX_BODY_BYTES = 1024 * 1024


def env(name, default=""):
    return os.environ.get(name, default).strip()


def truncate(value, limit=300):
    value = str(value or "").strip()
    if len(value) <= limit:
        return value
    return value[: limit - 1] + "..."


def first_present(*values):
    for value in values:
        if value:
            return value
    return ""


def feishu_sign(secret, timestamp):
    string_to_sign = f"{timestamp}\n{secret}"
    digest = hmac.new(string_to_sign.encode("utf-8"), b"", digestmod=hashlib.sha256).digest()
    return base64.b64encode(digest).decode("utf-8")


def alert_labels(alert):
    labels = alert.get("labels")
    return labels if isinstance(labels, dict) else {}


def alert_annotations(alert):
    annotations = alert.get("annotations")
    return annotations if isinstance(annotations, dict) else {}


def alert_target(alert):
    labels = alert_labels(alert)
    return first_present(labels.get("name"), labels.get("target"), labels.get("instance"), "-")


def alert_target_url(alert):
    labels = alert_labels(alert)
    return first_present(labels.get("target"), labels.get("instance"), "")


def alert_summary(alert):
    annotations = alert_annotations(alert)
    labels = alert_labels(alert)
    return first_present(annotations.get("summary"), labels.get("alertname"), "Campus alert")


def alert_description(alert):
    annotations = alert_annotations(alert)
    return first_present(annotations.get("description"), annotations.get("message"), "")


def alert_link(payload, alert):
    return first_present(
        alert.get("dashboardURL"),
        alert.get("panelURL"),
        alert.get("generatorURL"),
        payload.get("externalURL"),
        env("GRAFANA_ROOT_URL"),
    )


def line(*parts):
    return [{"tag": "text", "text": "".join(str(part) for part in parts)}]


def link_line(text, href):
    if href:
        return [{"tag": "a", "text": text, "href": href}]
    return line(text)


def build_feishu_post(payload):
    alerts = payload.get("alerts")
    if not isinstance(alerts, list):
        alerts = []

    common_labels = payload.get("commonLabels")
    if not isinstance(common_labels, dict):
        common_labels = {}

    status = str(payload.get("status") or "unknown").upper()
    severity = first_present(common_labels.get("severity"), "unknown")
    alertname = first_present(common_labels.get("alertname"), "CampusAlert")
    deploy_env = env("LEHU_ALERT_ENV", "local")
    title = f"校园 e站告警 [{deploy_env}] [{status}] {severity} {alertname}"

    content = [
        line("状态：", status, "    环境：", deploy_env, "    级别：", severity),
        line("告警组：", alertname, "    数量：", len(alerts)),
    ]

    for index, alert in enumerate(alerts[:MAX_ALERTS_IN_MESSAGE], start=1):
        content.extend([
            line(""),
            line("#", index, " 目标：", truncate(alert_target(alert), 160)),
            line("地址：", truncate(alert_target_url(alert), 220)),
            line("摘要：", truncate(alert_summary(alert), 220)),
        ])
        description = alert_description(alert)
        if description:
            content.append(line("说明：", truncate(description, 300)))
        starts_at = alert.get("startsAt")
        if starts_at:
            content.append(line("开始：", starts_at))
        link = alert_link(payload, alert)
        if link:
            content.append(link_line("打开 Grafana", link))

    if len(alerts) > MAX_ALERTS_IN_MESSAGE:
        content.append(line("另有 ", len(alerts) - MAX_ALERTS_IN_MESSAGE, " 条告警已合并，请进 Grafana 查看。"))

    content.extend([
        line(""),
        line("排查：先看健康面板定位组件，再按服务名或 request_id 到 Grafana 日志搜索。"),
    ])

    return {
        "msg_type": "post",
        "content": {
            "post": {
                "zh_cn": {
                    "title": title,
                    "content": content,
                }
            }
        },
    }


def list_items(values, key, limit=5):
    if not isinstance(values, list):
        return []
    out = []
    for item in values[:limit]:
        if isinstance(item, dict):
            text = first_present(item.get(key), item.get("title"), item.get("label"), item.get("source"), item.get("detail"))
            detail = first_present(item.get("detail"), item.get("priority"), item.get("severity"), item.get("path"), item.get("link"))
            out.append((truncate(text, 120), truncate(detail, 180)))
        else:
            out.append((truncate(item, 120), ""))
    return out


def admin_href(path):
    path = str(path or "").strip()
    if not path:
        return ""
    if path.startswith("http://") or path.startswith("https://"):
        return path
    base = first_present(env("LEHU_ADMIN_ROOT_URL"), env("ADMIN_ROOT_URL"))
    if not base:
        return ""
    return base.rstrip("/") + "/" + path.lstrip("/")


def build_agent_feishu_post(payload):
    actions = payload.get("actions")
    if isinstance(actions, list) and actions:
        return build_agent_feishu_card(payload)

    deploy_env = env("LEHU_ALERT_ENV", "local")
    run_type = truncate(payload.get("run_type") or "-", 40)
    run_id = truncate(payload.get("run_id") or "-", 40)
    risk_level = truncate(payload.get("risk_level") or "low", 20)
    title = truncate(payload.get("title") or f"校园 e站运营值班 Agent [{deploy_env}] {risk_level}", 120)
    summary = truncate(payload.get("summary") or "暂无摘要", 400)

    content = [
        line("环境：", deploy_env, "    风险：", risk_level, "    类型：", run_type),
        line("Run ID：", run_id),
        line("摘要：", summary),
    ]

    findings = list_items(payload.get("findings"), "title")
    if findings:
        content.append(line(""))
        content.append(line("关键发现"))
        for index, (text, detail) in enumerate(findings, start=1):
            content.append(line(index, ". ", text, (" - " + detail) if detail else ""))

    recommendations = list_items(payload.get("recommendations"), "title")
    if recommendations:
        content.append(line(""))
        content.append(line("建议动作"))
        for index, (text, detail) in enumerate(recommendations, start=1):
            content.append(line(index, ". ", text, (" - " + detail) if detail else ""))

    actions = payload.get("next_actions")
    if isinstance(actions, list) and actions:
        content.append(line(""))
        content.append(line("后台入口"))
        for action in actions[:5]:
            if not isinstance(action, dict):
                continue
            label = truncate(action.get("label") or "打开后台", 80)
            href = admin_href(action.get("href") or action.get("path"))
            content.append(link_line(label, href))

    content.extend([
        line(""),
        line("说明：日报、举报和反馈只做提醒；发帖审核卡片可使用一次性链接通过/拒绝，其余治理动作回后台处理。"),
    ])

    return {
        "msg_type": "post",
        "content": {
            "post": {
                "zh_cn": {
                    "title": title,
                    "content": content,
                }
            }
        },
    }


def card_template(risk_level):
    risk = str(risk_level or "").lower()
    if risk == "high":
        return "red"
    if risk == "medium":
        return "orange"
    return "green"


def markdown_escape(value):
    return str(value or "").replace("\n", " ").strip()


def build_agent_feishu_card(payload):
    deploy_env = env("LEHU_ALERT_ENV", "local")
    run_type = truncate(payload.get("run_type") or "-", 40)
    run_id = truncate(payload.get("run_id") or "-", 40)
    risk_level = truncate(payload.get("risk_level") or "low", 20)
    title = truncate(payload.get("title") or f"校园 e站运营值班 [{deploy_env}]", 120)
    summary = truncate(payload.get("summary") or "暂无摘要", 500)
    elements = [
        {
            "tag": "div",
            "text": {
                "tag": "lark_md",
                "content": f"**环境**：{deploy_env}    **风险**：{risk_level}    **类型**：{run_type}\n**ID**：{run_id}\n**摘要**：{markdown_escape(summary)}",
            },
        }
    ]

    findings = list_items(payload.get("findings"), "title", limit=4)
    if findings:
        content = "\n".join(f"{idx}. {markdown_escape(text)}{(' - ' + markdown_escape(detail)) if detail else ''}" for idx, (text, detail) in enumerate(findings, start=1))
        elements.append({"tag": "div", "text": {"tag": "lark_md", "content": f"**关键发现**\n{content}"}})

    recommendations = list_items(payload.get("recommendations"), "title", limit=3)
    if recommendations:
        content = "\n".join(f"{idx}. {markdown_escape(text)}{(' - ' + markdown_escape(detail)) if detail else ''}" for idx, (text, detail) in enumerate(recommendations, start=1))
        elements.append({"tag": "div", "text": {"tag": "lark_md", "content": f"**建议动作**\n{content}"}})

    button_items = []
    for action in payload.get("actions")[:6]:
        if not isinstance(action, dict):
            continue
        label = truncate(action.get("label") or "打开", 20)
        href = first_present(action.get("url"), admin_href(action.get("href") or action.get("path")))
        if not href:
            continue
        style = str(action.get("style") or "default").lower()
        button_type = "default"
        if style in ("primary", "danger"):
            button_type = style
        button_items.append({
            "tag": "button",
            "text": {"tag": "plain_text", "content": label},
            "url": href,
            "type": button_type,
            "value": {"action": action.get("action") or label, "target_id": payload.get("target_id")},
        })
    if button_items:
        elements.append({"tag": "action", "actions": button_items})

    elements.append({
        "tag": "note",
        "elements": [{"tag": "plain_text", "content": "Agent 负责提醒和建议；通过/拒绝按钮使用一次性 token，其他治理动作回后台处理。"}],
    })
    return {
        "msg_type": "interactive",
        "card": {
            "config": {"wide_screen_mode": True},
            "header": {
                "template": card_template(risk_level),
                "title": {"tag": "plain_text", "content": title},
            },
            "elements": elements,
        },
    }


def send_feishu_body(body, original_payload):
    webhook = env("LEHU_ALERT_FEISHU_WEBHOOK")
    if not webhook:
        print(json.dumps({"event": "feishu_webhook_missing", "payload": original_payload}, ensure_ascii=False), flush=True)
        return {"delivered": False, "reason": "missing_webhook"}

    secret = env("LEHU_ALERT_FEISHU_SECRET")
    if secret:
        timestamp = str(int(time.time()))
        body["timestamp"] = timestamp
        body["sign"] = feishu_sign(secret, timestamp)

    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        webhook,
        data=data,
        headers={"Content-Type": "application/json; charset=utf-8"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=8) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            status = resp.getcode()
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        return {"delivered": False, "status": exc.code, "body": raw}
    except Exception as exc:
        return {"delivered": False, "error": type(exc).__name__, "message": str(exc)}

    parsed = {}
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        pass
    code = parsed.get("code", parsed.get("StatusCode", 0))
    ok = 200 <= status < 300 and code in (0, None)
    return {"delivered": ok, "status": status, "body": parsed or raw}


def send_feishu(payload):
    return send_feishu_body(build_feishu_post(payload), payload)


def send_agent_feishu(payload):
    return send_feishu_body(build_agent_feishu_post(payload), payload)


class Handler(BaseHTTPRequestHandler):
    server_version = "campus-alert-webhook/1.0"

    def log_message(self, fmt, *args):
        return

    def write_json(self, status, payload):
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/healthz":
            self.write_json(200, {"ok": True, "feishu_webhook_configured": bool(env("LEHU_ALERT_FEISHU_WEBHOOK"))})
            return
        self.write_json(404, {"ok": False, "error": "not_found"})

    def do_POST(self):
        parsed = urlparse(self.path)
        if parsed.path not in ("/grafana", "/agent"):
            self.write_json(404, {"ok": False, "error": "not_found"})
            return

        expected_token = env("LEHU_ALERT_WEBHOOK_TOKEN")
        if not expected_token:
            self.write_json(503, {"ok": False, "error": "webhook_token_missing"})
            return
        token = parse_qs(parsed.query).get("token", [""])[0]
        if not hmac.compare_digest(token, expected_token):
            self.write_json(401, {"ok": False, "error": "unauthorized"})
            return

        try:
            length = int(self.headers.get("Content-Length", "0") or "0")
        except ValueError:
            self.write_json(400, {"ok": False, "error": "invalid_content_length"})
            return
        if length > MAX_BODY_BYTES:
            self.write_json(413, {"ok": False, "error": "payload_too_large"})
            return
        raw = self.rfile.read(length)
        try:
            payload = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            self.write_json(400, {"ok": False, "error": "invalid_json"})
            return

        if parsed.path == "/agent":
            result = send_agent_feishu(payload)
            event = "agent_notice"
        else:
            result = send_feishu(payload)
            event = "grafana_alert"
        print(json.dumps({"event": event, "result": result}, ensure_ascii=False), flush=True)
        if result.get("delivered") is False and result.get("reason") != "missing_webhook":
            self.write_json(502, {"ok": False, "result": result})
            return
        self.write_json(202, {"ok": True, "result": result})


def main():
    listen = env("LEHU_ALERT_LISTEN", "0.0.0.0:9120")
    host, port_text = listen.rsplit(":", 1)
    server = ThreadingHTTPServer((host, int(port_text)), Handler)
    print(json.dumps({"event": "alert_webhook_started", "listen": listen}, ensure_ascii=False), flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
