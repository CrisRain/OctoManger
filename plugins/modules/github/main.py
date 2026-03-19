#!/usr/bin/env python3
from __future__ import annotations

import os
import sys
import time
from typing import Any

# Ensure the octo SDK package is importable when running main.py directly.
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))

from octo import (
    Module,
    ModuleRequest,
    ParamSpec,
    Setting,
    UIButton,
    UISection,
    UITab,
    emit_daemon_done,
    emit_daemon_event,
    emit_daemon_init_ok,
    emit_log,
    error,
    success,
)
from client import GitHubAPIError, GitHubClient, as_config
from register import CaptchaError, RegistrationError, handle_register
from utils import as_bool, as_dict, as_int, now_utc

TYPE_KEY = "github"

# ---------------------------------------------------------------------------
# Module definition
# ---------------------------------------------------------------------------

module = Module(
    key=TYPE_KEY,
    name="GitHub",
    category="generic",
    account_schema={
        "type": "object",
        "properties": {
            "username": {"type": "string", "title": "GitHub 用户名"},
            "token": {"type": "string", "title": "Access Token"},
            "api_base_url": {
                "type": "string",
                "title": "API Base URL",
                "default": "https://api.github.com",
            },
            "user_agent": {"type": "string", "title": "User-Agent"},
            "timeout_seconds": {
                "type": "integer",
                "title": "超时秒数",
                "default": 30,
                "minimum": 5,
                "maximum": 120,
            },
            "default_owner": {"type": "string", "title": "默认 Owner"},
        },
        "required": ["username", "token"],
    },
    settings=[
        Setting(
            key="twocaptcha_api_key",
            label="2captcha API Key",
            type="string",
            secret=True,
            description="用于 REGISTER 时自动解 Arkose Labs 验证码",
        ),
        Setting(
            key="proxy",
            label="HTTP 代理",
            type="string",
            description="注册时使用的代理，格式 http://host:port",
        ),
    ],
)

module.set_ui(
    tabs=[
        UITab(
            key="account",
            label="账号",
            sections=[
                UISection(
                    title="账号操作",
                    buttons=[
                        UIButton(action="VERIFY", label="验证活性", mode="sync"),
                        UIButton(action="GET_PROFILE", label="获取资料", mode="sync"),
                    ],
                ),
            ],
        ),
        UITab(
            key="repos",
            label="仓库",
            sections=[
                UISection(
                    title="仓库管理",
                    buttons=[
                        UIButton(action="LIST_REPOSITORIES", label="列出仓库", mode="sync"),
                        UIButton(
                            action="CREATE_REPOSITORY",
                            label="创建仓库",
                            mode="sync",
                            form=[
                                ParamSpec(name="name", type="string", required=True, description="仓库名"),
                                ParamSpec(name="description", type="string", description="仓库描述"),
                                ParamSpec(name="private", type="string", choices=["true", "false"], description="是否私有（默认 false）"),
                                ParamSpec(name="auto_init", type="string", choices=["true", "false"], description="是否自动初始化（默认 false）"),
                            ],
                        ),
                    ],
                ),
            ],
        ),
        UITab(
            key="register",
            label="注册",
            context="create",
            sections=[
                UISection(
                    title="自动注册",
                    buttons=[
                        UIButton(
                            action="REGISTER",
                            label="注册账号",
                            mode="job",
                            form=[
                                ParamSpec(name="username", type="string", required=True, description="GitHub 用户名"),
                                ParamSpec(name="password", type="string", required=True, description="账号密码"),
                                ParamSpec(name="email", type="string", required=True, description="注册邮箱地址"),
                                ParamSpec(name="email_account_id", type="string", required=True, description="OctoManger 邮箱账号 ID"),
                                ParamSpec(name="twocaptcha_api_key", type="string", description="2captcha Key（留空则读取模块设置）"),
                                ParamSpec(name="mode", type="string", choices=["api", "browser"], description="注册模式（默认 api）"),
                                ParamSpec(name="proxy", type="string", description="HTTP 代理（留空则读取模块设置）"),
                                ParamSpec(name="wait_seconds", type="string", description="等待验证码邮件最长秒数（默认 120）"),
                            ],
                        ),
                    ],
                ),
            ],
        ),
    ],
    list_actions=[
        UIButton(action="VERIFY", label="验证活性", mode="sync"),
    ],
)


# ---------------------------------------------------------------------------
# Error-handling wrapper
# ---------------------------------------------------------------------------

def _wrap(fn):
    """Catch GitHubAPIError / RegistrationError / CaptchaError and return proper error payloads."""
    def handler(req: ModuleRequest) -> dict:
        emit_log(
            "github action received",
            level="info",
            action=req.action,
            identifier=req.identifier,
            request_id=req.request_id,
            protocol=req.protocol,
            param_keys=sorted(req.params.keys()),
        )
        try:
            output = fn(req)
            emit_log(
                "github action completed",
                level="info",
                action=req.action,
                identifier=req.identifier,
                status=output.get("status", ""),
            )
            return output
        except ValueError:
            raise  # let ActionRouter.dispatch convert to VALIDATION_FAILED
        except GitHubAPIError as exc:
            emit_log("github api error", level="warn", action=req.action, identifier=req.identifier, code=exc.code, detail_message=exc.message)
            return error(exc.code, exc.message, exc.details or None)
        except (RegistrationError, CaptchaError) as exc:
            emit_log("github register error", level="warn", action=req.action, identifier=req.identifier, code=exc.code, detail_message=exc.message)
            details = getattr(exc, "details", None)
            return error(exc.code, exc.message, details or None)
        except Exception as exc:
            emit_log("github unexpected error", level="error", action=req.action, identifier=req.identifier, detail_message=str(exc))
            return error("UNEXPECTED_ERROR", str(exc))

    handler.__name__ = fn.__name__
    return handler


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _compact_user(user: dict[str, Any]) -> dict[str, Any]:
    return {
        "login": user.get("login"),
        "id": user.get("id"),
        "name": user.get("name"),
        "type": user.get("type"),
        "html_url": user.get("html_url"),
        "public_repos": user.get("public_repos"),
        "followers": user.get("followers"),
        "following": user.get("following"),
    }


def _compact_repo(repo: dict[str, Any]) -> dict[str, Any]:
    owner = as_dict(repo.get("owner"))
    return {
        "id": repo.get("id"),
        "name": repo.get("name"),
        "full_name": repo.get("full_name"),
        "private": repo.get("private"),
        "html_url": repo.get("html_url"),
        "description": repo.get("description"),
        "default_branch": repo.get("default_branch"),
        "visibility": repo.get("visibility"),
        "owner": owner.get("login"),
    }


def _compact_issue(issue: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": issue.get("id"),
        "number": issue.get("number"),
        "title": issue.get("title"),
        "state": issue.get("state"),
        "html_url": issue.get("html_url"),
    }


def _rate_limit(headers: dict[str, str]) -> dict[str, Any]:
    return {
        "limit": headers.get("x-ratelimit-limit"),
        "remaining": headers.get("x-ratelimit-remaining"),
        "used": headers.get("x-ratelimit-used"),
        "reset": headers.get("x-ratelimit-reset"),
    }


def _resolve_owner(identifier: str, default_owner: str, params: dict[str, Any]) -> str:
    owner = str(params.get("owner", "")).strip()
    if owner:
        return owner
    if default_owner:
        return default_owner
    return identifier


def _resolve_repo(params: dict[str, Any]) -> str:
    repo = str(params.get("repo", "")).strip() or str(params.get("name", "")).strip()
    if not repo:
        raise ValueError("repo or name is required")
    return repo


def _do_verify(req: ModuleRequest) -> dict:
    identifier = req.require_identifier()
    config = as_config(req.spec)
    client = GitHubClient(config)
    user, headers = client.get_authenticated_user()
    actual_login = str(user.get("login", "")).strip()
    return success({
        "event": "github.auth.verified",
        "identifier": identifier,
        "configured_username": config.username,
        "authenticated_login": actual_login,
        "matched": actual_login.lower() == identifier.lower(),
        "user": _compact_user(user),
        "rate_limit": _rate_limit(headers),
        "handled_at": now_utc(),
    })


# ---------------------------------------------------------------------------
# Action handlers
# ---------------------------------------------------------------------------

@module.action("VERIFY", description="验证 Token 是否可用，检测账号活性")
@_wrap
def action_verify(req: ModuleRequest) -> dict:
    return _do_verify(req)


@module.action("HEALTH_CHECK", description="VERIFY 的别名，检测账号活性")
@_wrap
def action_health_check(req: ModuleRequest) -> dict:
    return _do_verify(req)


@module.action("GET_PROFILE", description="获取当前认证用户资料")
@_wrap
def action_get_profile(req: ModuleRequest) -> dict:
    config = as_config(req.spec)
    client = GitHubClient(config)
    user, headers = client.get_authenticated_user()
    return success({
        "event": "github.profile.loaded",
        "user": _compact_user(user),
        "raw": user,
        "rate_limit": _rate_limit(headers),
        "handled_at": now_utc(),
    })


@module.action(
    "LIST_REPOSITORIES",
    description="列出当前账号的仓库",
    params=[
        ParamSpec(name="organization", type="string", description="组织名（留空则列个人仓库）"),
        ParamSpec(name="visibility", type="string", choices=["all", "public", "private"], description="仓库可见性（默认 all）"),
        ParamSpec(name="per_page", type="string", description="每页数量（1-100，默认 30）"),
        ParamSpec(name="page", type="string", description="页码（默认 1）"),
    ],
)
@_wrap
def action_list_repositories(req: ModuleRequest) -> dict:
    identifier = req.require_identifier()
    config = as_config(req.spec)
    client = GitHubClient(config)
    repos, headers = client.list_repositories(req.params)
    items = [_compact_repo(repo) for repo in repos]
    return success({
        "event": "github.repositories.listed",
        "identifier": identifier,
        "count": len(items),
        "items": items,
        "rate_limit": _rate_limit(headers),
        "handled_at": now_utc(),
    })


@module.action(
    "GET_REPOSITORY",
    description="获取单个仓库详情",
    params=[
        ParamSpec(name="repo", type="string", required=True, description="仓库名"),
        ParamSpec(name="owner", type="string", description="Owner（留空则用 default_owner 或 identifier）"),
    ],
)
@_wrap
def action_get_repository(req: ModuleRequest) -> dict:
    identifier = req.require_identifier()
    config = as_config(req.spec)
    client = GitHubClient(config)
    owner = _resolve_owner(identifier, config.default_owner, req.params)
    repo_name = _resolve_repo(req.params)
    repo, headers = client.get_repository(owner, repo_name)
    return success({
        "event": "github.repository.loaded",
        "identifier": identifier,
        "owner": owner,
        "repo": _compact_repo(repo),
        "raw": repo,
        "rate_limit": _rate_limit(headers),
        "handled_at": now_utc(),
    })


@module.action(
    "CREATE_REPOSITORY",
    description="创建仓库",
    params=[
        ParamSpec(name="name", type="string", required=True, description="仓库名"),
        ParamSpec(name="description", type="string", description="仓库描述"),
        ParamSpec(name="private", type="string", choices=["true", "false"], description="是否私有（默认 false）"),
        ParamSpec(name="auto_init", type="string", choices=["true", "false"], description="是否自动初始化（默认 false）"),
        ParamSpec(name="organization", type="string", description="在指定组织下创建（留空则在个人账号下）"),
    ],
)
@_wrap
def action_create_repository(req: ModuleRequest) -> dict:
    identifier = req.require_identifier()
    config = as_config(req.spec)
    client = GitHubClient(config)
    params = dict(req.params)
    params["private"] = as_bool(params.get("private"), False)
    params["auto_init"] = as_bool(params.get("auto_init"), False)
    params["has_issues"] = as_bool(params.get("has_issues"), True)
    params["has_projects"] = as_bool(params.get("has_projects"), True)
    params["has_wiki"] = as_bool(params.get("has_wiki"), True)
    repo, headers = client.create_repository(params)
    return success({
        "event": "github.repository.created",
        "identifier": identifier,
        "repo": _compact_repo(repo),
        "raw": repo,
        "rate_limit": _rate_limit(headers),
        "handled_at": now_utc(),
    })


@module.action(
    "CREATE_ISSUE",
    description="创建 Issue",
    params=[
        ParamSpec(name="repo", type="string", required=True, description="仓库名"),
        ParamSpec(name="title", type="string", required=True, description="Issue 标题"),
        ParamSpec(name="body", type="string", description="Issue 内容"),
        ParamSpec(name="owner", type="string", description="Owner（留空则用 default_owner）"),
    ],
)
@_wrap
def action_create_issue(req: ModuleRequest) -> dict:
    identifier = req.require_identifier()
    config = as_config(req.spec)
    client = GitHubClient(config)
    owner = _resolve_owner(identifier, config.default_owner, req.params)
    repo_name = _resolve_repo(req.params)
    issue, headers = client.create_issue(owner, repo_name, req.params)
    return success({
        "event": "github.issue.created",
        "identifier": identifier,
        "owner": owner,
        "repo": repo_name,
        "issue": _compact_issue(issue),
        "raw": issue,
        "rate_limit": _rate_limit(headers),
        "handled_at": now_utc(),
    })


@module.action(
    "REGISTER",
    description="自动注册 GitHub 账号（通过 2captcha 解验证码，轮询邮箱获取验证码）",
    params=[
        ParamSpec(name="username", type="string", required=True, description="目标 GitHub 用户名"),
        ParamSpec(name="password", type="string", required=True, description="账号密码"),
        ParamSpec(name="email", type="string", required=True, description="注册邮箱"),
        ParamSpec(name="email_account_id", type="string", required=True, description="OctoManger 邮箱账号 ID"),
        ParamSpec(name="twocaptcha_api_key", type="string", description="2captcha Key（留空则读取模块设置或 spec）"),
        ParamSpec(name="mode", type="string", choices=["api", "browser"], description="注册模式：api 或 browser（默认 api）"),
        ParamSpec(name="email_api_url", type="string", description="OctoManger 后端地址（留空则从 context 自动读取）"),
        ParamSpec(name="email_api_key", type="string", description="OctoManger API Key（留空则从 context 自动读取）"),
        ParamSpec(name="proxy", type="string", description="HTTP 代理（留空则读取模块设置或 spec）"),
        ParamSpec(name="wait_seconds", type="string", description="等待验证码邮件最长秒数（默认 120）"),
        ParamSpec(name="poll_interval", type="string", description="轮询间隔秒数（默认 8）"),
        ParamSpec(name="arkose_public_key", type="string", description="手动指定 Arkose 公钥 UUID"),
        ParamSpec(name="arkose_subdomain", type="string", description="手动指定 Arkose API JS 子域"),
    ],
)
def action_register(req: ModuleRequest) -> dict:
    # handle_register handles its own logging and catches CaptchaError / RegistrationError internally.
    identifier = req.require_identifier()
    return handle_register(identifier, req.spec, req.params, context=req.context)


@module.action(
    "WATCH",
    description="保活守护进程（心跳模式）",
    params=[
        ParamSpec(name="interval_seconds", type="string", description="心跳间隔秒数（默认 60）"),
        ParamSpec(name="stop_after_seconds", type="string", description="运行多少秒后自动停止（0 表示不停止）"),
        ParamSpec(name="emit_heartbeat", type="string", choices=["true", "false"], description="是否输出心跳事件（默认 false）"),
    ],
)
def action_watch(req: ModuleRequest) -> dict:
    identifier = req.require_identifier()
    interval_seconds = max(1, as_int(req.params.get("interval_seconds"), 60))
    stop_after_seconds = max(0, as_int(req.params.get("stop_after_seconds"), 0))
    emit_heartbeat_flag = as_bool(req.params.get("emit_heartbeat"), False)

    emit_daemon_init_ok(
        "github daemon started",
        identifier=identifier,
        interval_seconds=interval_seconds,
    )

    started_at = time.monotonic()
    heartbeat_count = 0
    while True:
        if stop_after_seconds and time.monotonic() - started_at >= stop_after_seconds:
            emit_daemon_done("github daemon stopping", identifier=identifier, heartbeat_count=heartbeat_count)
            return {"status": "done"}

        if emit_heartbeat_flag:
            heartbeat_count += 1
            emit_daemon_event({
                "event": "github.daemon.heartbeat",
                "identifier": identifier,
                "count": heartbeat_count,
                "handled_at": now_utc(),
            })

        time.sleep(interval_seconds)


if __name__ == "__main__":
    raise SystemExit(module.run())
