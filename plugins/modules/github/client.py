#!/usr/bin/env python3
from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any
from urllib import error, parse, request


DEFAULT_API_BASE_URL = "https://api.github.com"
DEFAULT_USER_AGENT = "OctoManger-GitHub-Module/1.0"
DEFAULT_TIMEOUT_SECONDS = 30
DEFAULT_API_VERSION = "2022-11-28"


class GitHubAPIError(Exception):
    def __init__(self, status_code: int, code: str, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.code = code
        self.message = message
        self.details = details or {}


@dataclass
class GitHubConfig:
    username: str
    token: str
    api_base_url: str = DEFAULT_API_BASE_URL
    user_agent: str = DEFAULT_USER_AGENT
    timeout_seconds: int = DEFAULT_TIMEOUT_SECONDS
    default_owner: str = ""


def as_int(value: Any, default: int, field_name: str) -> int:
    if value is None or value == "":
        return default
    try:
        return int(value)
    except Exception as exc:
        raise GitHubAPIError(400, "VALIDATION_FAILED", f"{field_name} must be an integer") from exc


def _as_bool(value: Any, default: bool) -> bool:
    if isinstance(value, bool):
        return value
    if value is None:
        return default

    normalized = str(value).strip().lower()
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off"}:
        return False
    return default


class GitHubClient:
    def __init__(self, config: GitHubConfig) -> None:
        self.config = config

    def get_authenticated_user(self) -> tuple[dict[str, Any], dict[str, str]]:
        return self.request_json("GET", "/user")

    def list_repositories(self, params: dict[str, Any]) -> tuple[list[dict[str, Any]], dict[str, str]]:
        organization = str(params.get("organization", "")).strip()
        query = {
            "visibility": str(params.get("visibility", "all")).strip() or "all",
            "affiliation": str(params.get("affiliation", "owner,collaborator,organization_member")).strip(),
            "sort": str(params.get("sort", "full_name")).strip() or "full_name",
            "direction": str(params.get("direction", "asc")).strip() or "asc",
            "per_page": as_int(params.get("per_page", 30), 30, "per_page"),
            "page": as_int(params.get("page", 1), 1, "page"),
        }
        if query["per_page"] < 1 or query["per_page"] > 100:
            raise GitHubAPIError(400, "VALIDATION_FAILED", "per_page must be between 1 and 100")
        if query["page"] < 1:
            raise GitHubAPIError(400, "VALIDATION_FAILED", "page must be >= 1")

        if organization:
            path = f"/orgs/{parse.quote(organization)}/repos"
            query.pop("affiliation", None)
        else:
            path = "/user/repos"

        payload, headers = self.request_json("GET", path, query=query)
        if not isinstance(payload, list):
            raise GitHubAPIError(502, "INVALID_RESPONSE", "GitHub API returned unexpected repository list payload")
        return payload, headers

    def get_repository(self, owner: str, repo: str) -> tuple[dict[str, Any], dict[str, str]]:
        path = f"/repos/{parse.quote(owner)}/{parse.quote(repo)}"
        payload, headers = self.request_json("GET", path)
        return self.expect_object(payload, "repository"), headers

    def create_repository(self, params: dict[str, Any]) -> tuple[dict[str, Any], dict[str, str]]:
        name = str(params.get("name", "")).strip()
        if not name:
            raise GitHubAPIError(400, "VALIDATION_FAILED", "name is required")

        organization = str(params.get("organization", "")).strip()
        body = {
            "name": name,
            "description": str(params.get("description", "")).strip(),
            "homepage": str(params.get("homepage", "")).strip(),
            "private": _as_bool(params.get("private"), False),
            "has_issues": _as_bool(params.get("has_issues"), True),
            "has_projects": _as_bool(params.get("has_projects"), True),
            "has_wiki": _as_bool(params.get("has_wiki"), True),
            "auto_init": _as_bool(params.get("auto_init"), False),
            "gitignore_template": str(params.get("gitignore_template", "")).strip() or None,
            "license_template": str(params.get("license_template", "")).strip() or None,
        }
        body = {key: value for key, value in body.items() if value is not None and value != ""}

        if organization:
            path = f"/orgs/{parse.quote(organization)}/repos"
        else:
            path = "/user/repos"

        payload, headers = self.request_json("POST", path, body=body)
        return self.expect_object(payload, "repository"), headers

    def create_issue(self, owner: str, repo: str, params: dict[str, Any]) -> tuple[dict[str, Any], dict[str, str]]:
        title = str(params.get("title", "")).strip()
        if not title:
            raise GitHubAPIError(400, "VALIDATION_FAILED", "title is required")

        labels = params.get("labels", [])
        assignees = params.get("assignees", [])
        if labels is None:
            labels = []
        if assignees is None:
            assignees = []
        if not isinstance(labels, list):
            raise GitHubAPIError(400, "VALIDATION_FAILED", "labels must be an array")
        if not isinstance(assignees, list):
            raise GitHubAPIError(400, "VALIDATION_FAILED", "assignees must be an array")

        body = {
            "title": title,
            "body": str(params.get("body", "")),
            "labels": [str(item) for item in labels],
            "assignees": [str(item) for item in assignees],
        }
        payload, headers = self.request_json(
            "POST",
            f"/repos/{parse.quote(owner)}/{parse.quote(repo)}/issues",
            body=body,
        )
        return self.expect_object(payload, "issue"), headers

    def request_json(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | None = None,
    ) -> tuple[Any, dict[str, str]]:
        base = self.config.api_base_url.rstrip("/")
        if not path.startswith("/"):
            path = "/" + path

        url = base + path
        if query:
            encoded_query = parse.urlencode({k: v for k, v in query.items() if v is not None and v != ""})
            if encoded_query:
                url = f"{url}?{encoded_query}"

        raw_body = None
        if body is not None:
            raw_body = json.dumps(body).encode("utf-8")

        req = request.Request(
            url,
            data=raw_body,
            method=method,
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {self.config.token}",
                "User-Agent": self.config.user_agent,
                "X-GitHub-Api-Version": DEFAULT_API_VERSION,
                "Content-Type": "application/json; charset=utf-8",
            },
        )

        try:
            with request.urlopen(req, timeout=self.config.timeout_seconds) as response:
                raw = response.read().decode("utf-8")
                payload = json.loads(raw) if raw else {}
                headers = {key.lower(): value for key, value in response.headers.items()}
                return payload, headers
        except error.HTTPError as exc:
            raw = exc.read().decode("utf-8", errors="replace")
            payload = {}
            if raw:
                try:
                    payload = json.loads(raw)
                except Exception:
                    payload = {"message": raw}
            raise GitHubAPIError(
                exc.code,
                self.normalize_error_code(payload.get("message", "github api error")),
                str(payload.get("message", f"GitHub API request failed with status {exc.code}")),
                {
                    "status_code": exc.code,
                    "documentation_url": payload.get("documentation_url"),
                    "errors": payload.get("errors"),
                    "response": payload,
                },
            ) from exc
        except error.URLError as exc:
            raise GitHubAPIError(0, "NETWORK_ERROR", f"request failed: {exc.reason}") from exc

    @staticmethod
    def expect_object(payload: Any, name: str) -> dict[str, Any]:
        if not isinstance(payload, dict):
            raise GitHubAPIError(502, "INVALID_RESPONSE", f"GitHub API returned unexpected {name} payload")
        return payload

    @staticmethod
    def normalize_error_code(message: str) -> str:
        text = str(message).strip().upper().replace(" ", "_").replace("-", "_")
        return text[:64] or "GITHUB_API_ERROR"


def as_config(spec: dict[str, Any]) -> GitHubConfig:
    username = str(spec.get("username", "")).strip()
    token = str(spec.get("token", "")).strip()
    if not username:
        raise GitHubAPIError(400, "VALIDATION_FAILED", "spec.username is required")
    if not token:
        raise GitHubAPIError(400, "VALIDATION_FAILED", "spec.token is required")

    api_base_url = str(spec.get("api_base_url", DEFAULT_API_BASE_URL)).strip() or DEFAULT_API_BASE_URL
    user_agent = str(spec.get("user_agent", DEFAULT_USER_AGENT)).strip() or DEFAULT_USER_AGENT

    timeout_seconds = as_int(spec.get("timeout_seconds", DEFAULT_TIMEOUT_SECONDS), DEFAULT_TIMEOUT_SECONDS, "spec.timeout_seconds")
    if timeout_seconds < 5 or timeout_seconds > 120:
        raise GitHubAPIError(400, "VALIDATION_FAILED", "spec.timeout_seconds must be between 5 and 120")

    default_owner = str(spec.get("default_owner", username)).strip() or username

    return GitHubConfig(
        username=username,
        token=token,
        api_base_url=api_base_url,
        user_agent=user_agent,
        timeout_seconds=timeout_seconds,
        default_owner=default_owner,
    )
