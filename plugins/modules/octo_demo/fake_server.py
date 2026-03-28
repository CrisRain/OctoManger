#!/usr/bin/env python3
"""
OctoDemo 假数据服务器
====================

这份实现既可以被 `main.py` 以内嵌方式启动，也保留命令行入口供本地调试：

    python3 fake_server.py              # 默认 127.0.0.1:18080
    python3 fake_server.py --port 9000  # 自定义端口
    python3 fake_server.py --host 0.0.0.0 --port 9000

内置测试账号:
    username : testuser
    api_key  : demo_testkey_12345678
    base_url : http://127.0.0.1:18080
"""
from __future__ import annotations

import argparse
import json
import random
import string
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import parse_qs, urlparse

DEFAULT_HOST = "127.0.0.1"
DEFAULT_PORT = 18080
DEFAULT_BASE_URL = f"http://{DEFAULT_HOST}:{DEFAULT_PORT}"
DEFAULT_TEST_USERNAME = "testuser"
DEFAULT_TEST_API_KEY = "demo_testkey_12345678"
MIN_LEASE_SECONDS = 30


# ---------------------------------------------------------------------------
# 内存数据存储
# ---------------------------------------------------------------------------

_data_lock = threading.Lock()
_users: dict[str, dict[str, Any]] = {}
_tasks: dict[str, dict[str, Any]] = {}
_api_keys: dict[str, str] = {}
_id_counter = 0
_seeded = False


def _new_id_unlocked() -> str:
    global _id_counter
    _id_counter += 1
    return str(_id_counter)


def _rand_str(n: int, chars: str = string.ascii_lowercase) -> str:
    return "".join(random.choices(chars, k=n))


def _now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _format_unix(ts: float) -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(ts))


def _past(max_days: int = 30) -> str:
    offset = random.randint(0, 86400 * max_days)
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - offset))


def _fake_task_unlocked(owner: str) -> dict[str, Any]:
    return {
        "id": _new_id_unlocked(),
        "title": random.choice([
            "审查代码变更",
            "更新 API 文档",
            "修复登录 Bug",
            "优化数据库查询",
            "添加单元测试",
            "部署到生产环境",
            "安全审计报告",
            "性能压测分析",
            "清理过期日志",
            "迁移旧版数据",
            "编写接口规范",
            "重构认证模块",
        ]),
        "status": random.choice(["pending", "in_progress", "completed", "cancelled"]),
        "priority": random.choice(["low", "medium", "high", "critical"]),
        "owner": owner,
        "created_at": _past(),
        "updated_at": _now(),
    }


def _seed_data(*, reset: bool = False) -> None:
    global _seeded, _id_counter

    with _data_lock:
        if _seeded and not reset:
            return

        if reset:
            _users.clear()
            _tasks.clear()
            _api_keys.clear()
            _id_counter = 0

        random.seed(42)
        first_names = ["Alice", "Bob", "Charlie", "Diana", "Eve"]
        last_names = ["Smith", "Jones", "Brown", "Wilson", "Taylor"]

        for i in range(5):
            uname = "user_" + _rand_str(6)
            akey = "demo_" + _rand_str(24, string.ascii_lowercase + string.digits)
            user = {
                "id": _new_id_unlocked(),
                "username": uname,
                "display_name": f"{first_names[i]} {last_names[i]}",
                "email": f"{uname}@example.com",
                "plan": random.choice(["free", "pro", "enterprise"]),
                "tasks_quota": random.choice([10, 100, 1000]),
                "created_at": _now(),
            }
            _users[uname] = user
            _api_keys[akey] = uname
            for _ in range(random.randint(3, 8)):
                task = _fake_task_unlocked(uname)
                _tasks[task["id"]] = task

        _users[DEFAULT_TEST_USERNAME] = {
            "id": _new_id_unlocked(),
            "username": DEFAULT_TEST_USERNAME,
            "display_name": "测试用户",
            "email": "testuser@example.com",
            "plan": "pro",
            "tasks_quota": 100,
            "created_at": _now(),
        }
        _api_keys[DEFAULT_TEST_API_KEY] = DEFAULT_TEST_USERNAME
        for _ in range(5):
            task = _fake_task_unlocked(DEFAULT_TEST_USERNAME)
            _tasks[task["id"]] = task

        _seeded = True


def snapshot_counts() -> dict[str, int]:
    _seed_data()
    with _data_lock:
        return {
            "user_count": len(_users),
            "task_count": len(_tasks),
        }


def parse_base_url(value: str | None) -> tuple[str, str, int]:
    raw = str(value or DEFAULT_BASE_URL).strip() or DEFAULT_BASE_URL
    parsed = urlparse(raw)

    if parsed.scheme and parsed.scheme != "http":
        raise ValueError("base_url 仅支持 http://host:port")
    if parsed.params or parsed.query or parsed.fragment:
        raise ValueError("base_url 不能包含参数或锚点")
    if parsed.path not in ("", "/"):
        raise ValueError("base_url 不能包含路径")

    host = (parsed.hostname or "").strip() or DEFAULT_HOST
    port = parsed.port or DEFAULT_PORT
    if port <= 0 or port > 65535:
        raise ValueError("base_url 端口必须在 1-65535 之间")

    normalized = f"http://{host}:{port}"
    return normalized, host, port


# ---------------------------------------------------------------------------
# HTTP 处理器
# ---------------------------------------------------------------------------

class Handler(BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):  # type: ignore[override]
        print(f"[fake-server] {self.address_string()} {fmt % args}")

    def _authed(self) -> str | None:
        key = self.headers.get("X-Api-Key", "").strip()
        with _data_lock:
            return _api_keys.get(key)

    def _send_json(self, data: object, status: int = 200) -> None:
        body = json.dumps({"code": 0, "data": data}, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _send_err(self, code: int, msg: str, status: int = 400) -> None:
        body = json.dumps({"code": code, "message": msg}, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_body(self) -> dict[str, Any]:
        try:
            length = int(self.headers.get("Content-Length", 0))
            return json.loads(self.rfile.read(length)) if length > 0 else {}
        except Exception:
            return {}

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/")
        qs = parse_qs(parsed.query)

        def q(key: str, default: str = "") -> str:
            vals = qs.get(key)
            return vals[0].strip() if vals else default

        parts = [p for p in path.split("/") if p]

        if len(parts) == 2 and parts[0] == "users":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            with _data_lock:
                user = _users.get(parts[1])
            if not user:
                return self._send_err(4040, "user not found", 404)
            return self._send_json(user)

        if len(parts) == 1 and parts[0] == "tasks":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)

            status_filter = q("status")
            priority_filter = q("priority")
            try:
                page = max(1, int(q("page", "1") or "1"))
                page_size = min(50, max(1, int(q("page_size", "20") or "20")))
            except ValueError:
                page, page_size = 1, 20

            with _data_lock:
                items = [task for task in _tasks.values() if task["owner"] == username]

            if status_filter:
                items = [task for task in items if task["status"] == status_filter]
            if priority_filter:
                items = [task for task in items if task["priority"] == priority_filter]

            total = len(items)
            start = (page - 1) * page_size
            return self._send_json({
                "items": items[start:start + page_size],
                "total": total,
                "page": page,
                "page_size": page_size,
                "has_more": start + page_size < total,
            })

        if len(parts) == 1 and parts[0] == "stats":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)

            with _data_lock:
                my_tasks = [task for task in _tasks.values() if task["owner"] == username]

            by_status: dict[str, int] = {}
            by_priority: dict[str, int] = {}
            for task in my_tasks:
                by_status[task["status"]] = by_status.get(task["status"], 0) + 1
                by_priority[task["priority"]] = by_priority.get(task["priority"], 0) + 1
            return self._send_json({
                "total": len(my_tasks),
                "by_status": by_status,
                "by_priority": by_priority,
            })

        self._send_err(4040, "not found", 404)

    def do_POST(self):
        parts = [p for p in urlparse(self.path).path.rstrip("/").split("/") if p]
        body = self._read_body()

        if parts == ["auth", "verify"]:
            username = str(body.get("username", "")).strip()
            api_key = str(body.get("api_key", "")).strip()
            with _data_lock:
                stored = _api_keys.get(api_key)
                user = _users.get(username) if stored == username else None
            if user:
                return self._send_json({"valid": True, "user": user})
            return self._send_json({"valid": False, "user": None})

        if parts == ["auth", "register"]:
            username = str(body.get("username", "")).strip()
            if not username:
                return self._send_err(1001, "username is required")
            with _data_lock:
                if username in _users:
                    return self._send_err(1002, "username already exists", 409)
                api_key = "demo_" + _rand_str(24, string.ascii_lowercase + string.digits)
                user = {
                    "id": _new_id_unlocked(),
                    "username": username,
                    "display_name": str(body.get("display_name", username)).strip() or username,
                    "email": f"{username}@example.com",
                    "plan": "free",
                    "tasks_quota": 10,
                    "created_at": _now(),
                }
                _users[username] = user
                _api_keys[api_key] = username
            return self._send_json({"user": user, "api_key": api_key})

        if parts == ["tasks"]:
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)
            title = str(body.get("title", "")).strip()
            if not title:
                return self._send_err(1001, "title is required")
            priority = str(body.get("priority", "medium")).strip()
            if priority not in ("low", "medium", "high", "critical"):
                return self._send_err(1002, "invalid priority; must be low/medium/high/critical")
            with _data_lock:
                task = {
                    "id": _new_id_unlocked(),
                    "title": title,
                    "status": "pending",
                    "priority": priority,
                    "owner": username,
                    "created_at": _now(),
                    "updated_at": _now(),
                }
                _tasks[task["id"]] = task
            return self._send_json(task)

        self._send_err(4040, "not found", 404)

    def do_PATCH(self):
        parts = [p for p in urlparse(self.path).path.rstrip("/").split("/") if p]

        if len(parts) == 3 and parts[0] == "tasks" and parts[2] == "complete":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)

            task_id = parts[1]
            with _data_lock:
                task = _tasks.get(task_id)
                if not task:
                    return self._send_err(4040, "task not found", 404)
                if task["owner"] != username:
                    return self._send_err(4030, "forbidden", 403)
                task["status"] = "completed"
                task["updated_at"] = _now()
            return self._send_json(task)

        self._send_err(4040, "not found", 404)

    def do_DELETE(self):
        parts = [p for p in urlparse(self.path).path.rstrip("/").split("/") if p]

        if len(parts) == 2 and parts[0] == "tasks":
            username = self._authed()
            if not username:
                return self._send_err(4010, "unauthorized", 401)

            task_id = parts[1]
            with _data_lock:
                task = _tasks.get(task_id)
                if not task:
                    return self._send_err(4040, "task not found", 404)
                if task["owner"] != username:
                    return self._send_err(4030, "forbidden", 403)
                del _tasks[task_id]
            return self._send_json({"deleted": True, "id": task_id})

        self._send_err(4040, "not found", 404)


class _DemoHTTPServer(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = True


def _shutdown_http_server(server: _DemoHTTPServer, thread: threading.Thread | None) -> None:
    try:
        server.shutdown()
    except Exception:
        pass
    try:
        server.server_close()
    except Exception:
        pass
    if thread is not None:
        thread.join(timeout=5)


class DemoFakeServerManager:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._server: _DemoHTTPServer | None = None
        self._thread: threading.Thread | None = None
        self._watcher: threading.Thread | None = None
        self._base_url = ""
        self._owner_agent_id = ""
        self._lease_expires_at = 0.0
        self._started_at = ""

    def ensure_running(
        self,
        *,
        base_url: str | None = None,
        owner_agent_id: str = "",
        lease_seconds: int = 180,
    ) -> dict[str, Any]:
        normalized, host, port = parse_base_url(base_url)
        lease_seconds = max(MIN_LEASE_SECONDS, int(lease_seconds or MIN_LEASE_SECONDS))
        _seed_data()

        while True:
            existing_server: _DemoHTTPServer | None = None
            existing_thread: threading.Thread | None = None

            with self._lock:
                if self._server is not None and self._thread is not None and not self._thread.is_alive():
                    existing_server = self._server
                    existing_thread = self._thread
                    self._clear_runtime_locked()
                elif self._server is not None and self._base_url != normalized:
                    existing_server = self._server
                    existing_thread = self._thread
                    self._clear_runtime_locked()
                elif self._server is not None:
                    self._owner_agent_id = owner_agent_id or self._owner_agent_id
                    self._lease_expires_at = time.time() + lease_seconds
                    self._ensure_watcher_locked()
                    return self._snapshot_locked(state="reused")

            if existing_server is None:
                break
            _shutdown_http_server(existing_server, existing_thread)

        with self._lock:
            if self._server is None:
                server = _DemoHTTPServer((host, port), Handler)
                thread = threading.Thread(
                    target=server.serve_forever,
                    kwargs={"poll_interval": 0.5},
                    daemon=True,
                    name="octo-demo-fake-server",
                )
                thread.start()
                self._server = server
                self._thread = thread
                self._base_url = normalized
                self._owner_agent_id = owner_agent_id
                self._lease_expires_at = time.time() + lease_seconds
                self._started_at = _now()
                self._ensure_watcher_locked()
                return self._snapshot_locked(state="started")
            self._owner_agent_id = owner_agent_id or self._owner_agent_id
            self._lease_expires_at = time.time() + lease_seconds
            self._ensure_watcher_locked()
            return self._snapshot_locked(state="reused")

    def status(self) -> dict[str, Any]:
        with self._lock:
            return self._snapshot_locked(state="running" if self._server is not None else "stopped")

    def _ensure_watcher_locked(self) -> None:
        if self._watcher is not None and self._watcher.is_alive():
            return
        self._watcher = threading.Thread(
            target=self._watch_loop,
            daemon=True,
            name="octo-demo-fake-server-watchdog",
        )
        self._watcher.start()

    def _watch_loop(self) -> None:
        while True:
            time.sleep(2)

            server: _DemoHTTPServer | None = None
            thread: threading.Thread | None = None
            base_url = ""

            with self._lock:
                if self._server is None or self._lease_expires_at <= 0:
                    continue
                if time.time() < self._lease_expires_at:
                    continue
                server = self._server
                thread = self._thread
                base_url = self._base_url
                self._clear_runtime_locked()

            if server is not None:
                print(f"[fake-server] 租约过期，自动停止 {base_url}")
                _shutdown_http_server(server, thread)

    def _clear_runtime_locked(self) -> None:
        self._server = None
        self._thread = None
        self._base_url = ""
        self._owner_agent_id = ""
        self._lease_expires_at = 0.0
        self._started_at = ""

    def _snapshot_locked(self, *, state: str) -> dict[str, Any]:
        counts = snapshot_counts()
        return {
            "state": state,
            "running": self._server is not None,
            "base_url": self._base_url or DEFAULT_BASE_URL,
            "owner_agent_id": self._owner_agent_id,
            "started_at": self._started_at or None,
            "lease_expires_at": _format_unix(self._lease_expires_at) if self._lease_expires_at > 0 else None,
            "test_username": DEFAULT_TEST_USERNAME,
            "test_api_key": DEFAULT_TEST_API_KEY,
            **counts,
        }


_MANAGER = DemoFakeServerManager()


def get_server_manager() -> DemoFakeServerManager:
    return _MANAGER


def serve_forever(host: str = DEFAULT_HOST, port: int = DEFAULT_PORT) -> None:
    _seed_data()
    counts = snapshot_counts()
    print(f"[fake-server] 启动于 http://{host}:{port}")
    print(f"[fake-server] 测试账号 → username={DEFAULT_TEST_USERNAME}  api_key={DEFAULT_TEST_API_KEY}")
    print(f"[fake-server] 已初始化 {counts['user_count']} 个用户，{counts['task_count']} 条任务")
    print("[fake-server] 按 Ctrl+C 停止\n")

    server = _DemoHTTPServer((host, port), Handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[fake-server] 已停止")
    finally:
        server.server_close()


def main() -> None:
    parser = argparse.ArgumentParser(description="OctoDemo 假数据服务器")
    parser.add_argument("--host", default=DEFAULT_HOST, help=f"监听地址 (默认: {DEFAULT_HOST})")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help=f"监听端口 (默认: {DEFAULT_PORT})")
    args = parser.parse_args()
    serve_forever(host=args.host, port=args.port)


if __name__ == "__main__":
    main()
