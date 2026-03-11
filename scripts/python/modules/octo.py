"""OctoManger internal SDK for OctoModule scripts.

Usage in a module::

    import octo

    ctx = request_payload.get("context", {})
    client = octo.from_context(ctx)
    if client:
        account = client.get_account_by_identifier("github", "alice")
        if account:
            client.patch_account_spec(account["id"], {"last_checked_at": "2026-01-01T00:00:00Z"})
"""
from __future__ import annotations

import json
import os
import sys
from typing import Any, Callable
from urllib import error as urllib_error
from urllib import parse, request as urllib_request

_DEFAULT_TIMEOUT = 60  # seconds
_IPC_PROTOCOL = "octo.ipc.v1"


def as_dict(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


class OctoAPIError(RuntimeError):
    """Raised when OctoManger internal API returns an error response."""

    def __init__(
        self,
        message: str,
        *,
        code: int | None = None,
        status: int | None = None,
        details: str = "",
    ) -> None:
        super().__init__(message)
        self.code = code
        self.status = status
        self.details = details


class OctoClient:
    """HTTP client for OctoManger internal REST APIs.

    Credentials are injected automatically via the module context.
    """

    def __init__(self, api_url: str, api_token: str, *, timeout_seconds: int = _DEFAULT_TIMEOUT) -> None:
        self.api_url = api_url.rstrip("/")
        self.api_token = api_token
        self.timeout_seconds = max(5, int(timeout_seconds or _DEFAULT_TIMEOUT))

    def request(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | list[Any] | None = None,
    ) -> Any:
        normalized_path = path if path.startswith("/") else f"/{path}"
        url = self.api_url + normalized_path
        if query:
            query_items: dict[str, Any] = {}
            for key, value in query.items():
                if value is None:
                    continue
                query_items[key] = value
            if query_items:
                url += "?" + parse.urlencode(query_items, doseq=True)

        raw_body: bytes | None = None
        if body is not None:
            raw_body = json.dumps(body, ensure_ascii=False).encode("utf-8")

        req = urllib_request.Request(url, method=method.upper(), data=raw_body)
        req.add_header("X-Api-Key", self.api_token)
        req.add_header("Accept", "application/json")
        if raw_body is not None:
            req.add_header("Content-Type", "application/json; charset=utf-8")

        try:
            with urllib_request.urlopen(req, timeout=self.timeout_seconds) as resp:
                text = resp.read().decode("utf-8")
        except urllib_error.HTTPError as exc:
            details = ""
            try:
                details = exc.read().decode("utf-8", errors="replace")
            except Exception:
                details = ""

            message = f"http {exc.code}"
            code: int | None = None
            if details:
                try:
                    payload = json.loads(details)
                    if isinstance(payload, dict):
                        message = str(payload.get("message", "")).strip() or message
                        code_value = payload.get("code")
                        if isinstance(code_value, int):
                            code = code_value
                except Exception:
                    pass
            raise OctoAPIError(message, code=code, status=exc.code, details=details) from exc
        except Exception as exc:
            raise OctoAPIError(f"request failed: {exc}") from exc

        try:
            payload = json.loads(text)
        except Exception as exc:
            raise OctoAPIError(f"invalid json response: {exc}", details=text) from exc

        if not isinstance(payload, dict):
            raise OctoAPIError("invalid response payload", details=text)

        code_value = payload.get("code")
        if isinstance(code_value, int) and code_value != 0:
            message = str(payload.get("message", "")).strip() or f"api error ({code_value})"
            raise OctoAPIError(message, code=code_value, details=text)

        return payload.get("data")

    def get(self, path: str, *, query: dict[str, Any] | None = None) -> Any:
        return self.request("GET", path, query=query)

    def post(self, path: str, *, body: dict[str, Any] | list[Any] | None = None) -> Any:
        return self.request("POST", path, body=body)

    def put(self, path: str, *, body: dict[str, Any] | list[Any] | None = None) -> Any:
        return self.request("PUT", path, body=body)

    def patch(self, path: str, *, body: dict[str, Any] | list[Any] | None = None) -> Any:
        return self.request("PATCH", path, body=body)

    def delete(self, path: str, *, body: dict[str, Any] | list[Any] | None = None) -> Any:
        return self.request("DELETE", path, body=body)

    def get_latest_email(self, account_id: int, mailbox: str = "INBOX") -> dict[str, Any] | None:
        """Return latest email detail dict, or None when inbox is empty/error."""
        try:
            query = {"mailbox": mailbox} if mailbox else None
            data = self.get(
                f"/api/v1/octo-modules/internal/email/accounts/{account_id}/messages/latest",
                query=query,
            )
            if not isinstance(data, dict) or not data.get("found"):
                return None
            item = data.get("item")
            return item if isinstance(item, dict) else None
        except Exception:
            return None

    def get_account(self, account_id: int) -> dict[str, Any] | None:
        """Return account details, or None on error."""
        try:
            data = self.get(f"/api/v1/octo-modules/internal/accounts/{account_id}")
            return data if isinstance(data, dict) else None
        except Exception:
            return None

    def get_account_by_identifier(self, type_key: str, identifier: str) -> dict[str, Any] | None:
        """Return account by (type_key, identifier), or None on error/not found."""
        try:
            data = self.get(
                "/api/v1/octo-modules/internal/accounts/by-identifier",
                query={"type_key": type_key, "identifier": identifier},
            )
            return data if isinstance(data, dict) else None
        except Exception:
            return None

    def patch_account_spec(self, account_id: int, spec: dict[str, Any]) -> dict[str, Any] | None:
        """Patch account spec and return updated account, or None on error."""
        try:
            data = self.patch(
                f"/api/v1/octo-modules/internal/accounts/{account_id}/spec",
                body={"spec": spec},
            )
            return data if isinstance(data, dict) else None
        except Exception:
            return None


def from_context(ctx: dict[str, Any] | None) -> OctoClient | None:
    """Create an OctoClient from the InputContext dict.

    Returns None if context does not contain api_url/api_token.
    """
    context = as_dict(ctx)
    api_url = str(context.get("api_url", "")).strip()
    api_token = str(context.get("api_token", "")).strip()
    if not api_url or not api_token:
        return None
    timeout_seconds = _coerce_timeout_seconds(
        context.get("api_timeout_seconds"),
        _coerce_timeout_seconds(os.getenv("OCTO_API_TIMEOUT_SECONDS"), _DEFAULT_TIMEOUT),
    )
    return OctoClient(api_url, api_token, timeout_seconds=timeout_seconds)


def _coerce_timeout_seconds(value: Any, default: int) -> int:
    if isinstance(value, bool):
        return default
    if isinstance(value, (int, float)):
        parsed = int(value)
    else:
        text = str(value).strip() if value is not None else ""
        if not text:
            return default
        try:
            parsed = int(text)
        except ValueError:
            return default
    return max(5, parsed)


class ModuleRequest:
    """Typed request view used by OctoModule action handlers."""

    def __init__(self, payload: dict[str, Any]) -> None:
        request = as_dict(payload)
        self.raw: dict[str, Any] = request
        self.action: str = str(request.get("action", "")).strip().upper()
        self.account: dict[str, Any] = as_dict(request.get("account"))
        self.spec: dict[str, Any] = as_dict(self.account.get("spec"))
        self.params: dict[str, Any] = as_dict(request.get("params"))
        self.context: dict[str, Any] = as_dict(request.get("context"))
        self.identifier: str = str(self.account.get("identifier", "")).strip()

    @property
    def request_id(self) -> str:
        return str(self.context.get("request_id", "")).strip()

    @property
    def protocol(self) -> str:
        return str(self.context.get("protocol", "")).strip()

    def client(self) -> OctoClient | None:
        return from_context(self.context)

    def require_identifier(self) -> str:
        if self.identifier:
            return self.identifier
        raise ValueError("account.identifier is required")


def request(payload: dict[str, Any]) -> ModuleRequest:
    return ModuleRequest(payload)


class ActionRouter:
    """Action dispatcher for module handlers.

    Example:
        router = octo.ActionRouter()
        @router.action("VERIFY")
        def handle_verify(req: octo.ModuleRequest) -> dict[str, Any]:
            req.require_identifier()
            return octo.success({"verified": True})
    """

    def __init__(self) -> None:
        self._handlers: dict[str, Callable[[ModuleRequest], dict[str, Any]]] = {}

    def action(self, action: str) -> Callable[[Callable[[ModuleRequest], dict[str, Any]]], Callable[[ModuleRequest], dict[str, Any]]]:
        normalized = str(action).strip().upper()
        if not normalized:
            raise ValueError("action is required")

        def decorator(handler: Callable[[ModuleRequest], dict[str, Any]]) -> Callable[[ModuleRequest], dict[str, Any]]:
            self._handlers[normalized] = handler
            return handler

        return decorator

    def add(self, action: str, handler: Callable[[ModuleRequest], dict[str, Any]]) -> None:
        normalized = str(action).strip().upper()
        if not normalized:
            raise ValueError("action is required")
        self._handlers[normalized] = handler

    def dispatch(self, payload: dict[str, Any]) -> dict[str, Any]:
        req = ModuleRequest(payload)
        handler = self._handlers.get(req.action)
        if handler is None:
            return error("UNSUPPORTED_ACTION", f"unsupported action: {req.action}")
        try:
            output = handler(req)
        except ValueError as exc:
            return error("VALIDATION_FAILED", str(exc))
        if not isinstance(output, dict):
            return error("BAD_OUTPUT", "handler must return a JSON object")
        return output


def make_router() -> ActionRouter:
    return ActionRouter()


def emit_log(*args: Any, **fields: Any) -> None:
    """Emit a structured runtime log line (consumed by bridge in real time)."""
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

    for key, value in fields.items():
        if value is None:
            continue
        payload[str(key)] = value
    print(json.dumps(payload, ensure_ascii=False), flush=True)


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
    except Exception as exc:  # pragma: no cover - defensive fallback
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
