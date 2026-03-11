#!/usr/bin/env python3
"""
Arkose Labs FunCaptcha solver via 2captcha.

API reference: https://2captcha.com/api-docs/arkoselabs-funcaptcha

GitHub wraps Arkose Labs under the name "OctoCaptcha".
The solved token is submitted as the `octocaptcha-token` form field.
"""

from __future__ import annotations

import json
import time
from typing import Any
from urllib import error, request


# ---------------------------------------------------------------------------
# Errors
# ---------------------------------------------------------------------------

class CaptchaError(Exception):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message


# ---------------------------------------------------------------------------
# Arkose Labs solver
# ---------------------------------------------------------------------------

class ArkoseSolver:
    """
    Solve Arkose Labs (FunCaptcha / OctoCaptcha) using 2captcha.

    Task types:
        FunCaptchaTaskProxyless — 2captcha uses its own IP (default)
        FunCaptchaTask          — caller supplies a proxy

    createTask request body:
        {
          "clientKey": "<2captcha api key>",
          "task": {
            "type":                  "FunCaptchaTaskProxyless",
            "websiteURL":            "<page url>",
            "websitePublicKey":      "<arkose public key UUID>",
            "funcaptchaApiJSSubdomain": "<custom subdomain>",  // optional
            "data":                  "<stringified JSON>",     // optional
            "userAgent":             "<ua string>",            // optional
            // proxy fields only for FunCaptchaTask:
            "proxyType":    "http" | "socks4" | "socks5",
            "proxyAddress": "<host>",
            "proxyPort":    <port>,
            "proxyLogin":   "<user>",   // optional
            "proxyPassword":"<pass>"    // optional
          }
        }

    getTaskResult response (when status == "ready"):
        {
          "errorId": 0,
          "status": "ready",
          "solution": { "token": "<funcaptcha token>" },
          "cost": "0.00145",
          "ip": "...",
          "createTime": ...,
          "endTime": ...
        }
    """

    _CREATE_TASK_URL = "https://api.2captcha.com/createTask"
    _GET_RESULT_URL  = "https://api.2captcha.com/getTaskResult"

    # GitHub's known Arkose Labs configuration (used as fallback)
    GITHUB_SIGNUP_PUBLIC_KEY = "0252249B-CF71-4112-BAF6-1827BE8A0C28"
    GITHUB_API_JS_SUBDOMAIN  = "github-api.arkoselabs.com"

    POLL_INTERVAL_S = 5    # seconds between result polls
    MAX_WAIT_S      = 120  # total solve timeout

    def __init__(self, api_key: str) -> None:
        if not api_key:
            raise CaptchaError("CAPTCHA_CONFIG_ERROR", "2captcha api_key is required")
        self._api_key = api_key

    # -- HTTP helpers ----------------------------------------------------------

    def _post_json(self, url: str, payload: dict[str, Any]) -> dict[str, Any]:
        """POST JSON, return parsed response dict."""
        data = json.dumps(payload).encode("utf-8")
        req = request.Request(url, data=data, method="POST")
        req.add_header("Content-Type", "application/json")
        req.add_header("Accept", "application/json")
        try:
            with request.urlopen(req, timeout=30) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except error.HTTPError as exc:
            raw = exc.read().decode("utf-8", errors="replace")
            raise CaptchaError(
                "CAPTCHA_HTTP_ERROR",
                f"2captcha HTTP {exc.code}: {raw[:300]}",
            ) from exc
        except error.URLError as exc:
            raise CaptchaError(
                "CAPTCHA_NETWORK_ERROR",
                f"2captcha network error: {exc.reason}",
            ) from exc

    # -- Task lifecycle --------------------------------------------------------

    def _create_task(self, task: dict[str, Any]) -> int:
        payload = {"clientKey": self._api_key, "task": task}
        resp = self._post_json(self._CREATE_TASK_URL, payload)
        if resp.get("errorId") != 0:
            desc = (
                resp.get("errorDescription")
                or resp.get("errorCode")
                or str(resp)
            )
            raise CaptchaError(
                "CAPTCHA_CREATE_FAILED",
                f"2captcha createTask error: {desc}",
            )
        task_id = resp.get("taskId")
        if not task_id:
            raise CaptchaError("CAPTCHA_CREATE_FAILED", "2captcha returned no taskId")
        return int(task_id)

    def _poll_result(self, task_id: int) -> str:
        payload = {"clientKey": self._api_key, "taskId": task_id}
        deadline = time.monotonic() + self.MAX_WAIT_S

        while time.monotonic() < deadline:
            time.sleep(self.POLL_INTERVAL_S)
            resp = self._post_json(self._GET_RESULT_URL, payload)

            if resp.get("errorId") != 0:
                desc = (
                    resp.get("errorDescription")
                    or resp.get("errorCode")
                    or str(resp)
                )
                raise CaptchaError(
                    "CAPTCHA_SOLVE_FAILED",
                    f"2captcha getTaskResult error: {desc}",
                )

            if resp.get("status") == "ready":
                solution = resp.get("solution") or {}
                token = str(solution.get("token", "")).strip()
                if not token:
                    raise CaptchaError(
                        "CAPTCHA_EMPTY_TOKEN",
                        "2captcha returned a ready status but an empty token",
                    )
                return token
            # status == "processing" — keep waiting

        raise CaptchaError(
            "CAPTCHA_TIMEOUT",
            f"2captcha did not return a solution within {self.MAX_WAIT_S}s",
        )

    # -- Public API ------------------------------------------------------------

    def solve(
        self,
        website_url: str,
        website_public_key: str,
        *,
        api_js_subdomain: str = "",
        data: str = "",
        user_agent: str = "",
        proxy_type: str = "",     # "http" | "socks4" | "socks5"
        proxy_address: str = "",
        proxy_port: int = 0,
        proxy_login: str = "",
        proxy_password: str = "",
    ) -> str:
        """
        Submit an Arkose Labs FunCaptcha task to 2captcha and return the token.

        Args:
            website_url:        Full URL of the page with the captcha.
            website_public_key: Arkose Labs public key (UUID string).
            api_js_subdomain:   Custom Arkose Labs API JS subdomain (optional).
            data:               Extra payload as a JSON-encoded string (optional).
            user_agent:         Browser User-Agent to report (optional).
            proxy_type:         "http", "socks4", or "socks5" — enables FunCaptchaTask.
            proxy_address:      Proxy hostname/IP (required when proxy_type is set).
            proxy_port:         Proxy port (required when proxy_type is set).
            proxy_login:        Proxy username (optional).
            proxy_password:     Proxy password (optional).

        Returns:
            Solved FunCaptcha token string (submit as `octocaptcha-token` on GitHub).

        Raises:
            CaptchaError on any failure.
        """
        use_proxy = bool(proxy_type and proxy_address and proxy_port)
        task_type = "FunCaptchaTask" if use_proxy else "FunCaptchaTaskProxyless"

        task: dict[str, Any] = {
            "type":             task_type,
            "websiteURL":       website_url,
            "websitePublicKey": website_public_key,
        }

        if api_js_subdomain:
            task["funcaptchaApiJSSubdomain"] = api_js_subdomain
        if data:
            task["data"] = data
        if user_agent:
            task["userAgent"] = user_agent

        if use_proxy:
            task["proxyType"]    = proxy_type
            task["proxyAddress"] = proxy_address
            task["proxyPort"]    = proxy_port
            if proxy_login:
                task["proxyLogin"] = proxy_login
            if proxy_password:
                task["proxyPassword"] = proxy_password

        task_id = self._create_task(task)
        return self._poll_result(task_id)
