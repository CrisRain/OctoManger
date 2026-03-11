package octomodule

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EnsureScriptFile(path string, typeKey string) (bool, error) {
	scriptPath := strings.TrimSpace(path)
	if scriptPath == "" {
		return false, fmt.Errorf("script path is empty")
	}

	if info, err := os.Stat(scriptPath); err == nil {
		if info.IsDir() {
			return false, fmt.Errorf("script path is a directory: %s", scriptPath)
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return false, err
	}

	content := buildTemplate(typeKey)
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func buildTemplate(typeKey string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(typeKey), "'", "_")
	if normalized == "" {
		normalized = "unknown"
	}

	return `#!/usr/bin/env python3
from __future__ import annotations

import os
import sys
from datetime import datetime, timezone
from typing import Any

sys.path.append(os.path.dirname(os.path.dirname(__file__)))
import octo

TYPE_KEY = '` + normalized + `'


def now_utc() -> str:
    return datetime.now(timezone.utc).isoformat()


def as_dict(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def success(result: dict[str, Any], session: dict[str, Any] | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {"status": "success", "result": result}
    if session is not None:
        payload["session"] = session
    return payload


def error(code: str, message: str, details: dict[str, Any] | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "status": "error",
        "error_code": code,
        "error_message": message,
    }
    if details:
        payload["result"] = {"details": details}
    return payload


# ── Action handlers ───────────────────────────────────────────────────────────
# Each handler receives (identifier, spec, params) and returns success() or error().

def handle_register(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    # TODO: implement
    return success({"event": "registered", "identifier": identifier, "handled_at": now_utc()})


def handle_verify(identifier: str, spec: dict[str, Any], params: dict[str, Any]) -> dict[str, Any]:
    # TODO: implement
    return success({"event": "verified", "identifier": identifier, "handled_at": now_utc()})


# ── Dispatch table ────────────────────────────────────────────────────────────
# To add a new action: define handle_<action>() above, then register it here.

ACTIONS = {
    "REGISTER": handle_register,
    "VERIFY":   handle_verify,
}


def execute(request: dict[str, Any]) -> dict[str, Any]:
    action = str(request.get("action", "")).strip().upper()
    account = as_dict(request.get("account"))
    identifier = str(account.get("identifier", "")).strip()
    if not identifier:
        return error("VALIDATION_FAILED", "account.identifier is required")

    spec = as_dict(account.get("spec"))
    params = as_dict(request.get("params"))

    handler = ACTIONS.get(action)
    if handler is None:
        return error("UNSUPPORTED_ACTION", f"unsupported action: {action}")

    return handler(identifier, spec, params)


def main() -> int:
    # Serve mode only: read NDJSON requests from stdin continuously.
    return octo.run_module(execute)


if __name__ == "__main__":
    raise SystemExit(main())
`
}
