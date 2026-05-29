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


def send_feishu(payload):
    webhook = env("LEHU_ALERT_FEISHU_WEBHOOK")
    if not webhook:
        print(json.dumps({"event": "feishu_webhook_missing", "payload": payload}, ensure_ascii=False), flush=True)
        return {"delivered": False, "reason": "missing_webhook"}

    body = build_feishu_post(payload)
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
            self.write_json(200, {"ok": True})
            return
        self.write_json(404, {"ok": False, "error": "not_found"})

    def do_POST(self):
        parsed = urlparse(self.path)
        if parsed.path != "/grafana":
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

        result = send_feishu(payload)
        print(json.dumps({"event": "grafana_alert", "result": result}, ensure_ascii=False), flush=True)
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
