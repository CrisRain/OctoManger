from __future__ import annotations

import contextlib
import contextvars
import json
import sys
from collections.abc import Callable
from typing import Any

from ._response import error

_IPC_PROTOCOL = "octo.ipc.v1"
_LOG_CONTEXT: contextvars.ContextVar[dict[str, Any]] = contextvars.ContextVar("octo_log_context", default={})


def emit_log(*args: Any, **fields: Any) -> None:
    level = fields.pop("level", "info")
    primary_message = ""
    if len(args) > 0:
        primary_message = str(args[0])
    if len(args) > 1 and args[1] is not None:
        level = args[1]

    if not primary_message.strip():
        event = fields.pop("event", None)
        if event is not None:
            primary_message = str(event)

    extra_message = fields.pop("message", None)
    if extra_message is None:
        extra_message = fields.pop("detail_message", None)

    if not primary_message.strip() and extra_message is not None:
        primary_message = str(extra_message)
        extra_message = None

    payload: dict[str, Any] = {
        "status": "log",
        "level": str(level).strip() or "info",
        "message": primary_message,
    }
    if extra_message is not None:
        payload["detail_message"] = str(extra_message)

    context_fields = _LOG_CONTEXT.get()
    for key, value in context_fields.items():
        if value is None:
            continue
        key_str = str(key)
        if key_str in payload or key_str in fields:
            continue
        payload[key_str] = value
    for key, value in fields.items():
        if value is None:
            continue
        payload[str(key)] = value
    print(json.dumps(payload, ensure_ascii=False), flush=True)


@contextlib.contextmanager
def log_context(**fields: Any) -> Any:
    current = dict(_LOG_CONTEXT.get() or {})
    merged = current
    for key, value in fields.items():
        if value is None:
            continue
        merged[str(key)] = value
    token = _LOG_CONTEXT.set(merged)
    try:
        yield merged
    finally:
        _LOG_CONTEXT.reset(token)


def emit_daemon_init_ok(message: str = "", **fields: Any) -> None:
    payload: dict[str, Any] = {"status": "init_ok"}
    if str(message).strip():
        payload["message"] = str(message)
    for key, value in fields.items():
        if value is None:
            continue
        payload[str(key)] = value
    print(json.dumps(payload, ensure_ascii=False), flush=True)


def emit_daemon_event(result: dict[str, Any] | None = None, message: str = "", **fields: Any) -> None:
    payload: dict[str, Any] = {"status": "event", "result": result or {}}
    if str(message).strip():
        payload["message"] = str(message)
    for key, value in fields.items():
        if value is None:
            continue
        payload[str(key)] = value
    print(json.dumps(payload, ensure_ascii=False), flush=True)


def emit_daemon_done(message: str = "", **fields: Any) -> None:
    payload: dict[str, Any] = {"status": "done"}
    if str(message).strip():
        payload["message"] = str(message)
    for key, value in fields.items():
        if value is None:
            continue
        payload[str(key)] = value
    print(json.dumps(payload, ensure_ascii=False), flush=True)


def emit_daemon_error(code: str, message: str, details: dict[str, Any] | None = None, **fields: Any) -> None:
    payload: dict[str, Any] = {
        "status": "error",
        "error_code": str(code).strip() or "UNEXPECTED_ERROR",
        "error_message": str(message).strip() or "unexpected error",
    }
    if details:
        payload["result"] = {"details": details}
    for key, value in fields.items():
        if value is None:
            continue
        payload[str(key)] = value
    print(json.dumps(payload, ensure_ascii=False), flush=True)


def run_module(handler: Callable[[dict[str, Any]], dict[str, Any]], argv: list[str] | None = None) -> int:
    _ = argv
    return _run_stream(handler)


def _run_stream(handler: Callable[[dict[str, Any]], dict[str, Any]]) -> int:
    for raw in sys.stdin:
        line = raw.strip()
        if not line:
            continue
        try:
            request_payload, envelope_id = _extract_stream_request(line)
        except Exception as exc:
            print(json.dumps(error("BAD_INPUT", f"invalid stream request: {exc}"), ensure_ascii=False), flush=True)
            continue
        output = _execute_handler(handler, request_payload)
        if envelope_id:
            wrapped = {
                "protocol": _IPC_PROTOCOL,
                "type": "response",
                "id": envelope_id,
                "payload": output,
            }
            print(json.dumps(wrapped, ensure_ascii=False), flush=True)
            continue
        print(json.dumps(output, ensure_ascii=False), flush=True)
    return 0


def _execute_handler(handler: Callable[[dict[str, Any]], dict[str, Any]], request_payload: dict[str, Any]) -> dict[str, Any]:
    try:
        output = handler(request_payload)
        if not isinstance(output, dict):
            output = error("BAD_OUTPUT", "handler must return a JSON object")
    except Exception as exc:  # pragma: no cover
        output = error("UNEXPECTED_ERROR", str(exc))
    return output


def _extract_stream_request(line: str) -> tuple[dict[str, Any], str]:
    raw_payload = json.loads(line)
    if not isinstance(raw_payload, dict):
        raise ValueError("request must be JSON object")

    protocol = str(raw_payload.get("protocol", "")).strip()
    if protocol != _IPC_PROTOCOL:
        return raw_payload, ""

    msg_type = str(raw_payload.get("type", "")).strip().lower()
    if msg_type != "request":
        raise ValueError("protocol message type must be request")

    payload = raw_payload.get("payload")
    if not isinstance(payload, dict):
        raise ValueError("protocol payload must be JSON object")
    return payload, str(raw_payload.get("id", "")).strip()
