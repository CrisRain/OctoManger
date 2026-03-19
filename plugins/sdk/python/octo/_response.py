from __future__ import annotations

from typing import Any


def success(result: dict[str, Any] | None = None, session: dict[str, Any] | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {"status": "success", "result": result or {}}
    if session is not None:
        payload["session"] = session
    return payload


def error(code: str, message: str, details: dict[str, Any] | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "status": "error",
        "error_code": str(code).strip() or "UNEXPECTED_ERROR",
        "error_message": str(message).strip() or "unexpected error",
    }
    if details:
        payload["result"] = {"details": details}
    return payload
