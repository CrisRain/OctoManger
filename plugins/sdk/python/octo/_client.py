from __future__ import annotations

import json
import os
from typing import Any
from urllib import error as urllib_error
from urllib import parse, request as urllib_request

_DEFAULT_TIMEOUT = 60


def as_dict(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


class OctoAPIError(RuntimeError):
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
        try:
            return self.get_account_or_raise(account_id)
        except Exception:
            return None

    def get_account_by_identifier(self, type_key: str, identifier: str) -> dict[str, Any] | None:
        try:
            return self.get_account_by_identifier_or_raise(type_key, identifier)
        except Exception:
            return None

    def patch_account_spec(self, account_id: int, spec: dict[str, Any]) -> dict[str, Any] | None:
        try:
            data = self.patch(
                f"/api/v1/octo-modules/internal/accounts/{account_id}/spec",
                body={"spec": spec},
            )
            return data if isinstance(data, dict) else None
        except Exception:
            return None

    def get_account_or_raise(self, account_id: int) -> dict[str, Any]:
        data = self.get(f"/api/v1/octo-modules/internal/accounts/{account_id}")
        if not isinstance(data, dict):
            raise OctoAPIError("internal account api returned invalid payload")
        return data

    def get_account_by_identifier_or_raise(self, type_key: str, identifier: str) -> dict[str, Any]:
        data = self.get(
            "/api/v1/octo-modules/internal/accounts/by-identifier",
            query={"type_key": type_key, "identifier": identifier},
        )
        if not isinstance(data, dict):
            raise OctoAPIError("internal account api returned invalid payload")
        return data


def from_context(ctx: dict[str, Any] | None) -> OctoClient | None:
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
