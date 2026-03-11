#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import sys
import time
from datetime import datetime, timezone
from typing import Any

from client import GitHubAPIError, GitHubClient, as_config
from register import RegistrationError, handle_register

sys.path.append(os.path.dirname(os.path.dirname(__file__)))
import octo


TYPE_KEY = "github"


from utils import as_bool, as_dict, as_int, error, now_utc, success


def compact_user(user: dict[str, Any]) -> dict[str, Any]:
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


def compact_repo(repo: dict[str, Any]) -> dict[str, Any]:
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


def compact_issue(issue: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": issue.get("id"),
        "number": issue.get("number"),
        "title": issue.get("title"),
        "state": issue.get("state"),
        "html_url": issue.get("html_url"),
    }


def rate_limit(headers: dict[str, str]) -> dict[str, Any]:
    return {
        "limit": headers.get("x-ratelimit-limit"),
        "remaining": headers.get("x-ratelimit-remaining"),
        "used": headers.get("x-ratelimit-used"),
        "reset": headers.get("x-ratelimit-reset"),
    }


def resolve_owner(identifier: str, config_default_owner: str, params: dict[str, Any]) -> str:
    owner = str(params.get("owner", "")).strip()
    if owner:
        return owner
    if config_default_owner:
        return config_default_owner
    return identifier


def resolve_repo_name(params: dict[str, Any]) -> str:
    repo = str(params.get("repo", "")).strip() or str(params.get("name", "")).strip()
    if not repo:
        raise GitHubAPIError(400, "VALIDATION_FAILED", "repo or name is required")
    return repo


def handle_verify(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    del params
    config = as_config(spec)
    client = GitHubClient(config)
    user, headers = client.get_authenticated_user()
    actual_login = str(user.get("login", "")).strip()

    result = {
        "event": "github.auth.verified",
        "identifier": identifier,
        "configured_username": config.username,
        "authenticated_login": actual_login,
        "matched": actual_login.lower() == identifier.lower(),
        "user": compact_user(user),
        "rate_limit": rate_limit(headers),
        "handled_at": now_utc(),
    }
    return success(result)


def handle_get_profile(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    del identifier, params
    config = as_config(spec)
    client = GitHubClient(config)
    user, headers = client.get_authenticated_user()
    return success(
        {
            "event": "github.profile.loaded",
            "user": compact_user(user),
            "raw": user,
            "rate_limit": rate_limit(headers),
            "handled_at": now_utc(),
        }
    )


def handle_list_repositories(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    config = as_config(spec)
    client = GitHubClient(config)
    repos, headers = client.list_repositories(params)
    items = [compact_repo(repo) for repo in repos]
    return success(
        {
            "event": "github.repositories.listed",
            "identifier": identifier,
            "count": len(items),
            "items": items,
            "rate_limit": rate_limit(headers),
            "handled_at": now_utc(),
        }
    )


def handle_get_repository(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    config = as_config(spec)
    client = GitHubClient(config)
    owner = resolve_owner(identifier, config.default_owner, params)
    repo_name = resolve_repo_name(params)
    repo, headers = client.get_repository(owner, repo_name)
    return success(
        {
            "event": "github.repository.loaded",
            "identifier": identifier,
            "owner": owner,
            "repo": compact_repo(repo),
            "raw": repo,
            "rate_limit": rate_limit(headers),
            "handled_at": now_utc(),
        }
    )


def handle_create_repository(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    config = as_config(spec)
    client = GitHubClient(config)

    normalized_params = dict(params)
    normalized_params["private"] = as_bool(params.get("private"), False)
    normalized_params["auto_init"] = as_bool(params.get("auto_init"), False)
    normalized_params["has_issues"] = as_bool(params.get("has_issues"), True)
    normalized_params["has_projects"] = as_bool(params.get("has_projects"), True)
    normalized_params["has_wiki"] = as_bool(params.get("has_wiki"), True)

    repo, headers = client.create_repository(normalized_params)
    return success(
        {
            "event": "github.repository.created",
            "identifier": identifier,
            "repo": compact_repo(repo),
            "raw": repo,
            "rate_limit": rate_limit(headers),
            "handled_at": now_utc(),
        }
    )


def handle_create_issue(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    config = as_config(spec)
    client = GitHubClient(config)
    owner = resolve_owner(identifier, config.default_owner, params)
    repo_name = resolve_repo_name(params)
    issue, headers = client.create_issue(owner, repo_name, params)
    return success(
        {
            "event": "github.issue.created",
            "identifier": identifier,
            "owner": owner,
            "repo": repo_name,
            "issue": compact_issue(issue),
            "raw": issue,
            "rate_limit": rate_limit(headers),
            "handled_at": now_utc(),
        }
    )


def handle_watch(identifier: str, spec: dict[str, Any], params: dict[str, Any], context: dict[str, Any]) -> dict[str, Any]:
    del spec, context
    interval_seconds = max(1, as_int(params.get("interval_seconds"), 60))
    stop_after_seconds = max(0, as_int(params.get("stop_after_seconds"), 0))
    emit_heartbeat = as_bool(params.get("emit_heartbeat"), False)

    octo.emit_daemon_init_ok("github daemon started", identifier=identifier, interval_seconds=interval_seconds)

    started_at = time.time()
    heartbeat_count = 0
    while True:
        if stop_after_seconds and time.time() - started_at >= stop_after_seconds:
            octo.emit_daemon_done("github daemon stopping", identifier=identifier, heartbeat_count=heartbeat_count)
            return {"status": "done"}

        if emit_heartbeat:
            heartbeat_count += 1
            octo.emit_daemon_event(
                {
                    "event": "github.daemon.heartbeat",
                    "identifier": identifier,
                    "count": heartbeat_count,
                    "handled_at": now_utc(),
                }
            )

        time.sleep(interval_seconds)


ACTIONS = {
    "VERIFY": handle_verify,
    "GET_PROFILE": handle_get_profile,
    "LIST_REPOSITORIES": handle_list_repositories,
    "GET_REPOSITORY": handle_get_repository,
    "CREATE_REPOSITORY": handle_create_repository,
    "CREATE_ISSUE": handle_create_issue,
    "HEALTH_CHECK": handle_verify,
    "REGISTER": handle_register,
}


def execute(request_payload: dict[str, Any]) -> dict[str, Any]:
    action = str(request_payload.get("action", "")).strip().upper()
    account = as_dict(request_payload.get("account"))
    identifier = str(account.get("identifier", "")).strip()
    context = as_dict(request_payload.get("context"))
    if not identifier:
        return error("VALIDATION_FAILED", "account.identifier is required")

    spec = as_dict(account.get("spec"))
    params = as_dict(request_payload.get("params"))
    octo.emit_log(
        "github action received",
        level="info",
        action=action,
        identifier=identifier,
        request_id=str(context.get("request_id", "")).strip(),
        protocol=str(context.get("protocol", "")).strip(),
        param_keys=sorted(list(params.keys())),
    )

    if action == "WATCH":
        return handle_watch(identifier, spec, params, context)

    handler = ACTIONS.get(action)
    if handler is None:
        return error("UNSUPPORTED_ACTION", f"unsupported action: {action}")

    try:
        if action == "REGISTER":
            output = handle_register(identifier, spec, params, context=context)
        else:
            output = handler(identifier, spec, params)
        octo.emit_log("github action completed", level="info", action=action, identifier=identifier, status=output.get("status", ""))
        return output
    except GitHubAPIError as exc:
        octo.emit_log("github api error", level="warn", action=action, identifier=identifier, code=exc.code, detail_message=exc.message)
        return error(exc.code, exc.message, exc.details)
    except RegistrationError as exc:
        octo.emit_log("github register error", level="warn", action=action, identifier=identifier, code=exc.code, detail_message=exc.message)
        return error(exc.code, exc.message, exc.details)
    except Exception as exc:
        octo.emit_log("github unexpected error", level="error", action=action, identifier=identifier, detail_message=str(exc))
        return error("UNEXPECTED_ERROR", str(exc))


def main() -> int:
    return octo.run_module(execute)


if __name__ == "__main__":
    raise SystemExit(main())
