#!/usr/bin/env python3
import argparse
import concurrent.futures
import json
import socket
import threading
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def label_escape(value):
    return str(value).replace("\\", "\\\\").replace("\n", "\\n").replace('"', '\\"')


def metric(name, labels, value):
    rendered = ",".join(f'{key}="{label_escape(val)}"' for key, val in labels.items())
    return f"{name}{{{rendered}}} {value}"


def check_http(target, timeout):
    req = urllib.request.Request(target["target"], method="GET")
    start = time.monotonic()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            status = resp.getcode()
            success = 1 if 200 <= status < 300 else 0
            error = ""
    except urllib.error.HTTPError as exc:
        status = exc.code
        success = 0
        error = exc.reason or "http_error"
    except Exception as exc:
        status = 0
        success = 0
        error = type(exc).__name__
    return {
        "success": success,
        "duration": time.monotonic() - start,
        "status": status,
        "error": error,
    }


def check_tcp(target, timeout):
    host, port_text = target["target"].rsplit(":", 1)
    start = time.monotonic()
    try:
        with socket.create_connection((host, int(port_text)), timeout=timeout):
            success = 1
            error = ""
    except Exception as exc:
        success = 0
        error = type(exc).__name__
    return {
        "success": success,
        "duration": time.monotonic() - start,
        "status": 0,
        "error": error,
    }


def check_target(target, timeout):
    if target["type"] == "http":
        result = check_http(target, timeout)
    elif target["type"] == "tcp":
        result = check_tcp(target, timeout)
    else:
        result = {
            "success": 0,
            "duration": 0,
            "status": 0,
            "error": "unsupported_type",
        }
    result["target"] = target
    return result


def timeout_result(target, timeout):
    return {
        "success": 0,
        "duration": timeout,
        "status": 0,
        "error": "timeout",
        "target": target,
    }


def collect_results(targets, timeout):
    started = time.monotonic()
    executor = concurrent.futures.ThreadPoolExecutor(max_workers=max(1, len(targets)))
    futures = {
        executor.submit(check_target, target, timeout): target
        for target in targets
    }
    done, pending = concurrent.futures.wait(
        futures.keys(),
        timeout=timeout + 0.5,
        return_when=concurrent.futures.ALL_COMPLETED,
    )
    results = []
    for future, target in futures.items():
        if future in done:
            try:
                results.append(future.result())
            except Exception as exc:
                results.append({
                    "success": 0,
                    "duration": 0,
                    "status": 0,
                    "error": type(exc).__name__,
                    "target": target,
                })
        else:
            future.cancel()
            results.append(timeout_result(target, timeout))
    if pending:
        executor.shutdown(wait=False, cancel_futures=True)
    else:
        executor.shutdown(wait=True)

    checked_at = time.time()
    for result in results:
        result["checked_at"] = checked_at
    return results, time.monotonic() - started


def initial_result(target):
    return {
        "success": 0,
        "duration": 0,
        "status": 0,
        "error": "pending",
        "target": target,
        "checked_at": 0,
    }


def probe_loop(handler):
    while True:
        results, duration = collect_results(handler.targets, handler.timeout)
        with handler.lock:
            handler.results = results
            handler.last_probe_duration = duration
        time.sleep(handler.interval)


class Handler(BaseHTTPRequestHandler):
    targets = []
    timeout = 3.0
    interval = 15.0
    results = []
    last_probe_duration = 0.0
    lock = threading.Lock()

    def log_message(self, fmt, *args):
        return

    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok\n")
            return
        if self.path != "/metrics":
            self.send_response(404)
            self.end_headers()
            return

        started = time.monotonic()
        with self.lock:
            results = list(self.results)
            last_probe_duration = self.last_probe_duration

        lines = [
            "# HELP lehu_probe_success Whether a configured Lehu health probe succeeded.",
            "# TYPE lehu_probe_success gauge",
        ]
        for result in results:
            target = result["target"]
            labels = {
                "name": target["name"],
                "type": target["type"],
                "target": target["target"],
                "error": result["error"],
            }
            lines.append(metric("lehu_probe_success", labels, result["success"]))
            lines.append(metric("lehu_probe_duration_seconds", labels, f'{result["duration"]:.6f}'))
            lines.append(metric("lehu_probe_status_code", labels, result["status"]))
            lines.append(metric("lehu_probe_last_checked_timestamp_seconds", labels, f'{result.get("checked_at", 0):.3f}'))
        lines.extend([
            "# HELP lehu_health_exporter_scrape_duration_seconds Time spent rendering cached metrics.",
            "# TYPE lehu_health_exporter_scrape_duration_seconds gauge",
            f"lehu_health_exporter_scrape_duration_seconds {time.monotonic() - started:.6f}",
            "# HELP lehu_health_exporter_last_probe_duration_seconds Time spent checking all targets in the last background probe.",
            "# TYPE lehu_health_exporter_last_probe_duration_seconds gauge",
            f"lehu_health_exporter_last_probe_duration_seconds {last_probe_duration:.6f}",
        ])

        body = ("\n".join(lines) + "\n").encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--listen", default="0.0.0.0:9115")
    parser.add_argument("--timeout", type=float, default=3.0)
    parser.add_argument("--interval", type=float, default=15.0)
    args = parser.parse_args()

    with open(args.config, "r", encoding="utf-8") as fp:
        Handler.targets = json.load(fp)
    Handler.timeout = args.timeout
    Handler.interval = args.interval
    Handler.results = [initial_result(target) for target in Handler.targets]

    host, port_text = args.listen.rsplit(":", 1)
    server = ThreadingHTTPServer((host, int(port_text)), Handler)
    threading.Thread(target=probe_loop, args=(Handler,), daemon=True).start()
    server.serve_forever()


if __name__ == "__main__":
    main()
