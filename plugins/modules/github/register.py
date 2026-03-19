#!/usr/bin/env python3
"""
GitHub account auto-registration.

Two registration modes are available:

  api  (default) — direct HTTP requests via urllib; no browser required.
                   Uses the known/fallback Arkose Labs public key.
  browser        — Playwright Chromium automation; extracts the live Arkose
                   public key from the rendered DOM.

Both modes solve the Arkose Labs (OctoCaptcha) challenge via 2captcha and
read the verification email through the OctoManger email REST API.
"""

from __future__ import annotations

import http.cookiejar
import json
import os
import random
import re
import sys
import time
from dataclasses import dataclass
from typing import Any
from urllib import parse
from urllib import request as urllib_request

from captcha import ArkoseSolver, CaptchaError
from utils import as_bool, as_int, error as build_error, now_utc, success as build_success

sys.path.append(os.path.dirname(os.path.dirname(__file__)))
import octo


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

GITHUB_BASE           = "https://github.com"
DEFAULT_EMAIL_MAILBOX = "INBOX"
DEFAULT_WAIT_SECONDS  = 120
DEFAULT_POLL_INTERVAL = 8
EMAIL_API_TIMEOUT     = 20        # seconds – OctoManger API calls
PW_TIMEOUT            = 30_000    # ms      – Playwright waits
API_HTTP_TIMEOUT      = 30        # seconds – urllib calls
MIN_WAIT_SECONDS      = 10
MAX_WAIT_SECONDS      = 600
MIN_POLL_INTERVAL     = 3
MAX_POLL_INTERVAL     = 30

_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/124.0.0.0 Safari/537.36"
)

_UUID_RE = re.compile(
    r"[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}",
    re.IGNORECASE,
)
_SUCCESS_DESTINATION_RE = re.compile(r"github\.com/(dashboard|welcome|explore)", re.IGNORECASE)
_SUPPORTED_REGISTRATION_MODES = {"api", "browser"}


# ---------------------------------------------------------------------------
# Errors
# ---------------------------------------------------------------------------

class RegistrationError(Exception):
    def __init__(self, code: str, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.code    = code
        self.message = message
        self.details = details or {}


@dataclass(frozen=True)
class RegistrationRequest:
    mode: str
    username: str
    password: str
    email: str
    email_account_id: int
    twocaptcha_api_key: str
    email_api_url: str
    email_api_key: str
    email_mailbox: str
    wait_seconds: int
    poll_interval: int
    proxy: str = ""
    headless: bool = True
    arkose_public_key: str = ""
    arkose_subdomain: str = ""
    internal_client: Any = None

    @property
    def uses_internal_email_client(self) -> bool:
        return self.internal_client is not None


# ---------------------------------------------------------------------------
# OctoManger email client  (stdlib urllib — no browser needed)
# ---------------------------------------------------------------------------

def _describe_lookup_error(scope: str, exc: Exception) -> str:
    return f"{scope}: {type(exc).__name__}: {exc}"


class OctoMangerEmailClient:
    """Calls the OctoManger email REST API to fetch verification emails."""

    def __init__(self, api_url: str, api_key: str, account_id: int, internal_client: Any = None) -> None:
        self.api_url    = api_url.rstrip("/")
        self.api_key    = api_key
        self.account_id = account_id
        self.internal_client = internal_client
        self.last_error = ""

    def _can_use_http_fallback(self) -> bool:
        return bool(self.api_url and self.api_key)

    def get_latest_message(self, mailbox: str = "INBOX") -> dict[str, Any] | None:
        """Return the latest email dict, or None on empty inbox / error."""
        self.last_error = ""
        if self.internal_client is not None:
            try:
                item = self.internal_client.get_latest_email(self.account_id, mailbox)
                if isinstance(item, dict):
                    return item
            except Exception as exc:
                self.last_error = _describe_lookup_error("internal email lookup failed", exc)

        if not self._can_use_http_fallback():
            return None

        path = f"/api/v1/email/accounts/{self.account_id}/messages/latest"
        url  = self.api_url + path
        if mailbox:
            url += "?" + parse.urlencode({"mailbox": mailbox})

        req = urllib_request.Request(
            url,
            headers={
                "X-Api-Key":  self.api_key,
                "Accept":     "application/json",
                "User-Agent": "OctoManger-GitHub-Module/1.0",
            },
        )
        try:
            with urllib_request.urlopen(req, timeout=EMAIL_API_TIMEOUT) as resp:
                payload = json.loads(resp.read().decode("utf-8"))
                data    = payload.get("data", {})
                if not isinstance(data, dict) or not data.get("found"):
                    return None
                return data.get("item")
        except Exception as exc:
            self.last_error = _describe_lookup_error("email api request failed", exc)
            return None


# ---------------------------------------------------------------------------
# Email content helpers
# ---------------------------------------------------------------------------

def _extract_verification_code(text: str) -> str:
    """Find a 6–8 digit verification code in email body text."""
    patterns = [
        r"(?:verification code|your code|enter this code|launch code|code is)[:\s]+(\d{6,8})",
        r"(?<!\d)(\d{6})(?!\d)",
    ]
    for pattern in patterns:
        m = re.search(pattern, text, re.IGNORECASE)
        if m:
            return m.group(1)
    return ""


def _is_github_verification_email(item: dict[str, Any]) -> bool:
    subject   = str(item.get("subject", "")).lower()
    from_addr = str(item.get("from",    "")).lower()
    return (
        "github" in from_addr
        or "noreply@github.com" in from_addr
        or any(
            kw in subject
            for kw in ("verify", "verification", "confirm", "launch", "welcome to github")
        )
    )


def poll_for_verification_email(
    client: OctoMangerEmailClient,
    mailbox: str,
    wait_seconds: int,
    poll_interval: int,
    known_message_id: str = "",
) -> dict[str, Any] | None:
    """
    Poll OctoManger until a new GitHub verification email arrives.

    Skips `known_message_id` so we only act on email that arrived after
    registration was submitted.
    """
    deadline = time.monotonic() + wait_seconds
    seen_id  = known_message_id
    successful_lookups = 0
    last_lookup_error = ""

    while time.monotonic() < deadline:
        item = client.get_latest_message(mailbox)
        if client.last_error:
            last_lookup_error = client.last_error
        else:
            successful_lookups += 1

        if item and isinstance(item, dict):
            msg_id = str(item.get("id", ""))
            if msg_id and msg_id != seen_id and _is_github_verification_email(item):
                return item
            if msg_id:
                seen_id = msg_id

        remaining = deadline - time.monotonic()
        if remaining <= 0:
            break
        time.sleep(min(poll_interval, max(remaining, 1)))

    if successful_lookups == 0 and last_lookup_error:
        raise RegistrationError(
            "EMAIL_API_ERROR",
            "Could not fetch the verification email from OctoManger",
            {"mailbox": mailbox, "reason": last_lookup_error},
        )

    return None


# ---------------------------------------------------------------------------
# HTTP helpers (shared by the API registration mode)
# ---------------------------------------------------------------------------

def _build_http_opener(proxy: str) -> urllib_request.OpenerDirector:
    """Build an urllib opener with cookies, a browser UA, and optional proxy."""
    jar: http.cookiejar.CookieJar = http.cookiejar.CookieJar()
    handlers: list = [urllib_request.HTTPCookieProcessor(jar)]
    if proxy:
        handlers.append(
            urllib_request.ProxyHandler({"http": proxy, "https": proxy})
        )
    opener = urllib_request.build_opener(*handlers)
    opener.addheaders = [
        ("User-Agent",      _USER_AGENT),
        ("Accept",          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
        ("Accept-Language", "en-US,en;q=0.9"),
    ]
    return opener


def _http_get(opener: urllib_request.OpenerDirector, url: str) -> tuple[str, str]:
    """GET a URL; return (final_url, html_body)."""
    try:
        with opener.open(url, timeout=API_HTTP_TIMEOUT) as resp:
            return resp.geturl(), resp.read().decode("utf-8", errors="replace")
    except urllib_request.HTTPError as exc:
        body = b""
        try:
            body = exc.read()
        except Exception:
            pass
        return exc.geturl() or url, body.decode("utf-8", errors="replace")
    except Exception as exc:
        raise RegistrationError(
            "NETWORK_ERROR", f"HTTP GET {url} failed: {exc}"
        ) from exc


def _http_post(
    opener: urllib_request.OpenerDirector,
    url: str,
    fields: dict[str, str],
    referer: str = "",
) -> tuple[str, str]:
    """POST URL-encoded form data; return (final_url, html_body)."""
    data = parse.urlencode(fields).encode("utf-8")
    req  = urllib_request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/x-www-form-urlencoded")
    req.add_header("Origin", GITHUB_BASE)
    if referer:
        req.add_header("Referer", referer)
    try:
        with opener.open(req, timeout=API_HTTP_TIMEOUT) as resp:
            return resp.geturl(), resp.read().decode("utf-8", errors="replace")
    except urllib_request.HTTPError as exc:
        body = b""
        try:
            body = exc.read()
        except Exception:
            pass
        return exc.geturl() or url, body.decode("utf-8", errors="replace")
    except Exception as exc:
        raise RegistrationError(
            "NETWORK_ERROR", f"HTTP POST {url} failed: {exc}"
        ) from exc


def _parse_hidden_fields(html: str) -> dict[str, str]:
    """Extract all hidden <input> fields from an HTML page."""
    fields: dict[str, str] = {}
    for tag_m in re.finditer(r"<input[^>]+>", html, re.IGNORECASE):
        tag = tag_m.group(0)
        if 'type="hidden"' not in tag.lower() and "type='hidden'" not in tag.lower():
            continue
        name_m  = re.search(r'name=["\']([^"\']+)["\']', tag)
        value_m = re.search(r'value=["\']([^"\']*)["\']', tag)
        if name_m:
            fields[name_m.group(1)] = value_m.group(1) if value_m else ""
    return fields


def _parse_form_action(html: str, fallback: str) -> str:
    """Return the first form's action URL (absolute)."""
    m = re.search(r'<form[^>]+action=["\']([^"\']+)["\']', html, re.IGNORECASE)
    if m:
        action = m.group(1)
        return action if action.startswith("http") else GITHUB_BASE + action
    return fallback


def _resolve_arkose_settings(
    *,
    override_public_key: str,
    override_subdomain: str,
    detected_public_key: str = "",
    detected_subdomain: str = "",
) -> tuple[str, str]:
    public_key = override_public_key or detected_public_key or ArkoseSolver.GITHUB_SIGNUP_PUBLIC_KEY
    subdomain = override_subdomain or detected_subdomain or ArkoseSolver.GITHUB_API_JS_SUBDOMAIN
    return public_key, subdomain


def _is_signup_complete(url: str) -> bool:
    return bool(_SUCCESS_DESTINATION_RE.search(url))


def _message_id(item: dict[str, Any] | None) -> str:
    if not isinstance(item, dict):
        return ""
    return str(item.get("id", "")).strip()


def _email_search_text(item: dict[str, Any]) -> str:
    text_body = str(item.get("text_body", ""))
    if text_body:
        return text_body

    html_body = str(item.get("html_body", ""))
    return re.sub(r"<[^>]+>", " ", html_body)


def _wait_for_verification_code(
    *,
    email_client: OctoMangerEmailClient,
    mailbox: str,
    wait_seconds: int,
    poll_interval: int,
    known_message_id: str,
    email: str,
) -> tuple[str, dict[str, Any]]:
    email_item = poll_for_verification_email(
        client=email_client,
        mailbox=mailbox,
        wait_seconds=wait_seconds,
        poll_interval=poll_interval,
        known_message_id=known_message_id,
    )
    if not email_item:
        raise RegistrationError(
            "EMAIL_NOT_RECEIVED",
            f"No GitHub verification email arrived within {wait_seconds}s",
            {"email": email, "mailbox": mailbox},
        )

    search_text = _email_search_text(email_item)
    verification_code = _extract_verification_code(search_text)
    if not verification_code:
        raise RegistrationError(
            "CODE_NOT_FOUND",
            "Received GitHub email but could not extract a verification code",
            {"subject": email_item.get("subject", ""), "preview": search_text[:300]},
        )
    return verification_code, email_item


# ---------------------------------------------------------------------------
# API registration flow (no browser required)
# ---------------------------------------------------------------------------

def _api_register(
    registration: RegistrationRequest,
    email_client: OctoMangerEmailClient,
) -> dict[str, Any]:
    """
    Register a GitHub account using direct HTTP requests — no browser needed.

    Steps
    -----
    1.  GET github.com/signup to get session cookies and CSRF token.
    2.  Determine Arkose Labs public key (override → fallback constant).
    3.  Solve the FunCaptcha challenge via 2captcha.
    4.  POST the signup form with all required fields + captcha token.
    5.  Detect errors (username taken, captcha rejected, etc.).
    6.  Poll OctoManger email API for the verification code.
    7.  POST the verification code to complete registration.

    Returns a result dict consumed by handle_register().
    Raises RegistrationError or CaptchaError on failure.
    """
    opener     = _build_http_opener(registration.proxy)
    signup_url = GITHUB_BASE + "/signup"

    # ── 1. Fetch signup page ───────────────────────────────────────────────
    _, signup_html = _http_get(opener, signup_url)
    if not signup_html:
        raise RegistrationError("SIGNUP_PAGE_ERROR", "GitHub signup page returned an empty response")

    hidden = _parse_hidden_fields(signup_html)
    csrf   = hidden.get("authenticity_token", "")
    if not csrf:
        raise RegistrationError(
            "SIGNUP_PAGE_ERROR",
            "Could not extract CSRF token from GitHub signup page — "
            "GitHub may have changed their page structure.",
        )

    # ── 2. Arkose Labs parameters ──────────────────────────────────────────
    public_key, subdomain = _resolve_arkose_settings(
        override_public_key=registration.arkose_public_key,
        override_subdomain=registration.arkose_subdomain,
    )

    # ── 3. Solve FunCaptcha via 2captcha ──────────────────────────────────
    solver        = ArkoseSolver(registration.twocaptcha_api_key)
    captcha_token = solver.solve(
        website_url        = signup_url,
        website_public_key = public_key,
        api_js_subdomain   = subdomain,
        user_agent         = _USER_AGENT,
    )

    # ── 4. Snapshot inbox before submitting the form ───────────────────────
    pre_existing = email_client.get_latest_message(registration.email_mailbox)
    known_id     = _message_id(pre_existing)

    # ── 5. POST signup form ────────────────────────────────────────────────
    # Start from the full set of hidden fields (preserves honeypot / timestamps),
    # then add / override the user-facing fields.
    post_fields: dict[str, str] = dict(hidden)
    post_fields.update({
        "authenticity_token":       csrf,
        "user[login]":              registration.username,
        "user[email]":              registration.email,
        "user[password]":           registration.password,
        "octocaptcha-token":        captcha_token,
        "user[receive_newsletter]": "0",
        "source":                   "login",
        "return_to":                "",
        "payload":                  "",
    })

    form_action                 = _parse_form_action(signup_html, signup_url)
    response_url, response_html = _http_post(opener, form_action, post_fields, referer=signup_url)

    # ── 6. Detect common errors ────────────────────────────────────────────
    lower = response_html.lower()
    if "username" in lower and any(
        kw in lower for kw in ("not available", "already taken", "unavailable")
    ):
        raise RegistrationError(
            "USERNAME_UNAVAILABLE",
            f"Username {registration.username!r} is already taken or unavailable",
        )
    if "octocaptcha" in response_url.lower() or (
        "captcha" in response_url.lower()
    ):
        raise RegistrationError(
            "CAPTCHA_REJECTED",
            "GitHub returned a captcha wall after submitting the solved token — "
            "the public key may be outdated. Set arkose_public_key in params.",
            {"public_key_used": public_key, "subdomain_used": subdomain},
        )

    # ── 7. Already on dashboard? ───────────────────────────────────────────
    if _is_signup_complete(response_url):
        return {
            "email_verified":    True,
            "verification_code": None,
            "captcha_public_key": public_key,
            "email_subject":     "",
        }

    # ── 8. Poll OctoManger email API for verification code ─────────────────
    code, email_item = _wait_for_verification_code(
        email_client=email_client,
        mailbox=registration.email_mailbox,
        wait_seconds=registration.wait_seconds,
        poll_interval=registration.poll_interval,
        known_message_id=known_id,
        email=registration.email,
    )

    # ── 9. Submit verification code ───────────────────────────────────────
    # Prefer the form action from the response page; fall back to known path.
    verify_url = GITHUB_BASE + "/users/email_verification"
    action_m   = re.search(
        r'<form[^>]+action=["\']([^"\']*(?:verif|confirm|pin)[^"\']*)["\']',
        response_html, re.IGNORECASE,
    )
    if action_m:
        action     = action_m.group(1)
        verify_url = action if action.startswith("http") else GITHUB_BASE + action

    # Refresh CSRF token from the response page if available.
    verify_csrf = csrf
    new_csrf_m  = re.search(
        r'<input[^>]+name=["\']authenticity_token["\'][^>]+value=["\']([^"\']+)["\']',
        response_html,
    )
    if new_csrf_m:
        verify_csrf = new_csrf_m.group(1)

    final_url, _ = _http_post(
        opener,
        verify_url,
        {"authenticity_token": verify_csrf, "user_email_pin": code},
        referer=response_url,
    )
    verified = _is_signup_complete(final_url)

    return {
        "email_verified":     verified,
        "verification_code":  code,
        "captcha_public_key": public_key,
        "email_subject":      email_item.get("subject", ""),
    }


# ---------------------------------------------------------------------------
# Arkose Labs param extraction from the live rendered DOM  (browser mode only)
# ---------------------------------------------------------------------------

def _extract_arkose_key_from_dom(page: Any) -> tuple[str, str]:
    """
    Query the live Playwright page for the Arkose Labs public key and
    API JS subdomain.

    Strategy (in order):
      1. DOM attribute query on known Arkose / GitHub-captcha elements.
      2. JavaScript evaluation of window-level config objects.
      3. Regex scan of the raw page HTML source.

    Returns (public_key, api_js_subdomain); either may be empty string.
    """
    public_key = ""
    subdomain  = ""

    # 1. DOM attribute queries
    for selector, attrs in [
        ("github-captcha",       ("data-pkey", "data-public-key", "public-key")),
        ("[data-pkey]",          ("data-pkey",)),
        ("[data-public-key]",    ("data-public-key",)),
        ("fc-token",             ("data-pkey", "data-public-key")),
    ]:
        try:
            el = page.query_selector(selector)
            if not el:
                continue
            for attr in attrs:
                val = el.get_attribute(attr) or ""
                if val and _UUID_RE.match(val.strip()):
                    public_key = val.strip().upper()
                    break
            if public_key:
                break
        except Exception:
            continue

    # 2. JavaScript window-object probe
    if not public_key:
        try:
            pk = page.evaluate(
                """() => {
                    const candidates = [
                        window.__octocaptchaConfig?.publicKey,
                        window.octocaptcha?.publicKey,
                        document.querySelector('[data-pkey]')?.getAttribute('data-pkey'),
                    ];
                    return candidates.find(v => v && v.length > 10) || '';
                }"""
            )
            if pk and _UUID_RE.match(str(pk).strip()):
                public_key = str(pk).strip().upper()
        except Exception:
            pass

    # 3. Regex fallback on full page HTML
    html = ""
    if not public_key or not subdomain:
        try:
            html = page.content()
        except Exception:
            pass

    if not public_key and html:
        for attr in ("data-pkey", "data-public-key"):
            m = re.search(
                rf'{re.escape(attr)}=["\']([^"\']+)["\']', html, re.IGNORECASE
            )
            if m and _UUID_RE.match(m.group(1)):
                public_key = m.group(1).upper()
                break

    if not subdomain and html:
        for pattern in (
            r'funcaptchaApiJSSubdomain\s*[=:]\s*["\']([^"\']+)["\']',
            r'api_js_subdomain\s*[=:]\s*["\']([^"\']+)["\']',
        ):
            m = re.search(pattern, html, re.IGNORECASE)
            if m:
                subdomain = m.group(1).strip()
                break

    return public_key, subdomain


# ---------------------------------------------------------------------------
# Playwright registration flow  (browser mode)
# ---------------------------------------------------------------------------

def _sleep(lo: float = 0.3, hi: float = 0.9) -> None:
    time.sleep(random.uniform(lo, hi))


def _playwright_register(
    registration: RegistrationRequest,
    email_client: OctoMangerEmailClient,
) -> dict[str, Any]:
    """
    Run the full GitHub signup flow inside a Playwright Chromium browser.

    Steps
    -----
    1.  Open github.com/signup
    2.  Multi-step form: email → password → username → preferences
    3.  Extract Arkose Labs public key from the rendered DOM
    4.  Solve the FunCaptcha challenge via 2captcha
    5.  Inject the solved token into the page, click "Create account"
    6.  Poll OctoManger email API for the verification code
    7.  Enter the verification code in the browser
    8.  Confirm successful registration

    Returns a dict consumed by handle_register().
    Raises RegistrationError or CaptchaError on failure.
    """
    # Lazy import so Playwright is not required for API mode.
    try:
        from playwright.sync_api import (  # noqa: PLC0415
            BrowserContext,
            Page,
            TimeoutError as PlaywrightTimeout,
            sync_playwright,
        )
    except ImportError as exc:
        raise RegistrationError(
            "PLAYWRIGHT_NOT_INSTALLED",
            "playwright is not installed in this module venv; install requirements first",
        ) from exc

    proxy_config = {"server": registration.proxy} if registration.proxy else None

    try:
        pw_ctx = sync_playwright()
        pw = pw_ctx.__enter__()
    except Exception as exc:
        raise RegistrationError(
            "BROWSER_ERROR",
            f"failed to initialize Playwright: {type(exc).__name__}: {exc}",
        ) from exc

    try:
        try:
            browser = pw.chromium.launch(
                headless=registration.headless,
                proxy=proxy_config,
                args=[
                    "--disable-blink-features=AutomationControlled",
                    "--no-sandbox",
                ],
            )
        except Exception as exc:
            message = str(exc)
            lowered = message.lower()
            if "executable doesn't exist" in lowered or "please run the following command to download new browsers" in lowered:
                raise RegistrationError(
                    "PLAYWRIGHT_BROWSER_MISSING",
                    "playwright browser executable is missing; run `python -m playwright install chromium` in this module venv",
                ) from exc
            if "error while loading shared libraries" in lowered or "cannot open shared object file" in lowered:
                raise RegistrationError(
                    "PLAYWRIGHT_SYSTEM_DEPS_MISSING",
                    "playwright browser system libraries are missing in the Linux runtime image; rebuild the app image with the required shared libraries",
                ) from exc
            raise RegistrationError(
                "BROWSER_ERROR",
                f"failed to launch Playwright browser: {type(exc).__name__}: {exc}",
            ) from exc
        context: BrowserContext = browser.new_context(
            user_agent=_USER_AGENT,
            viewport={"width": 1280, "height": 800},
            locale="en-US",
            timezone_id="America/New_York",
        )
        # Hide webdriver fingerprint
        context.add_init_script(
            "Object.defineProperty(navigator, 'webdriver', {get: () => undefined})"
        )
        page: Page = context.new_page()

        try:
            # ── 1. Open signup page ────────────────────────────────────────────
            page.goto(GITHUB_BASE + "/signup", wait_until="networkidle", timeout=PW_TIMEOUT)
            _sleep(1, 2)

            # ── 2a. Email ──────────────────────────────────────────────────────
            page.fill("#email", registration.email)
            _sleep()
            page.click("button[data-continue-to='password-container']")
            page.wait_for_selector("#password", timeout=10_000)
            _sleep()

            # ── 2b. Password ───────────────────────────────────────────────────
            page.fill("#password", registration.password)
            _sleep()
            page.click("button[data-continue-to='username-container']")
            page.wait_for_selector("#login", timeout=10_000)
            _sleep()

            # ── 2c. Username ───────────────────────────────────────────────────
            page.fill("#login", registration.username)
            _sleep(1, 2)
            # Wait for async availability check
            page.wait_for_timeout(1500)
            try:
                err = page.query_selector(
                    "#login-error, [data-gh-component='InlineMessage']:visible"
                )
                if err and err.is_visible():
                    msg = err.inner_text().strip()
                    if msg and "available" not in msg.lower() and "✓" not in msg:
                        raise RegistrationError(
                            "USERNAME_UNAVAILABLE", f"Username not available: {msg}"
                        )
            except PlaywrightTimeout:
                pass

            try:
                page.click("button[data-continue-to='opt-in-container']", timeout=5_000)
                _sleep()
            except PlaywrightTimeout:
                pass

            # ── 2d. Email preferences → skip to captcha ────────────────────────
            try:
                page.click(
                    "button[data-continue-to='captcha-and-terms-container']",
                    timeout=5_000,
                )
                _sleep()
            except PlaywrightTimeout:
                pass

            # ── 3. Extract Arkose Labs public key from the rendered DOM ────────
            page.wait_for_timeout(2000)   # let captcha widget initialise
            page_public_key, page_subdomain = _extract_arkose_key_from_dom(page)

            public_key, subdomain = _resolve_arkose_settings(
                override_public_key=registration.arkose_public_key,
                override_subdomain=registration.arkose_subdomain,
                detected_public_key=page_public_key,
                detected_subdomain=page_subdomain,
            )

            # ── 4. Solve FunCaptcha via 2captcha ──────────────────────────────
            solver = ArkoseSolver(registration.twocaptcha_api_key)
            captcha_token = solver.solve(
                website_url        = GITHUB_BASE + "/signup",
                website_public_key = public_key,
                api_js_subdomain   = subdomain,
                user_agent         = _USER_AGENT,
            )

            # ── 5. Inject token and submit ────────────────────────────────────
            page.evaluate(
                """(token) => {
                    // Hidden field submitted with the form
                    const field = document.querySelector(
                        'input[name="octocaptcha-token"], #octocaptcha-token'
                    );
                    if (field) {
                        field.value = token;
                        field.dispatchEvent(new Event('input',  { bubbles: true }));
                        field.dispatchEvent(new Event('change', { bubbles: true }));
                    }
                    // GitHub's Web Component setter (if available)
                    const cap = document.querySelector('github-captcha');
                    if (cap && typeof cap.setToken === 'function') cap.setToken(token);
                }""",
                captcha_token,
            )
            _sleep(0.5, 1.0)

            page.click(
                "button[type='submit'], button:has-text('Create account')",
                timeout=PW_TIMEOUT,
            )
            _sleep(2, 4)

            # Detect captcha rejection (token was not accepted)
            if "octocaptcha" in page.url or page.query_selector("github-captcha:visible"):
                raise RegistrationError(
                    "CAPTCHA_REJECTED",
                    "GitHub returned a captcha wall after submitting the solved token — "
                    "the public key may be outdated. Set arkose_public_key in params.",
                    {"public_key_used": public_key, "subdomain_used": subdomain},
                )

            # ── 6. Snapshot inbox before email arrives ────────────────────────
            pre_existing = email_client.get_latest_message(registration.email_mailbox)
            known_id = _message_id(pre_existing)

            # ── 7. Wait for verification code input to appear ─────────────────
            code_input_sel = (
                "input[name='user_email_pin'], "
                "#user-email-pinned-field, "
                "input[autocomplete='one-time-code']"
            )
            try:
                page.wait_for_selector(code_input_sel, timeout=15_000)
            except PlaywrightTimeout:
                if _is_signup_complete(page.url):
                    return {
                        "email_verified":    True,
                        "verification_code": None,
                        "captcha_public_key": public_key,
                        "email_subject":     "",
                    }
                raise RegistrationError(
                    "VERIFICATION_PAGE_NOT_FOUND",
                    f"Verification code input not found after submit. URL: {page.url}",
                )

            # ── 8. Poll OctoManger email API for the code ─────────────────────
            code, email_item = _wait_for_verification_code(
                email_client=email_client,
                mailbox=registration.email_mailbox,
                wait_seconds=registration.wait_seconds,
                poll_interval=registration.poll_interval,
                known_message_id=known_id,
                email=registration.email,
            )

            # ── 9. Type code into browser and submit ──────────────────────────
            page.fill(code_input_sel, code)
            _sleep(0.5, 1.0)
            try:
                page.click("button[type='submit']:visible", timeout=5_000)
            except PlaywrightTimeout:
                page.keyboard.press("Enter")
            _sleep(2, 4)

            # ── 10. Confirm registration is complete ──────────────────────────
            try:
                page.wait_for_url(
                    _SUCCESS_DESTINATION_RE,
                    timeout=20_000,
                )
                verified = True
            except PlaywrightTimeout:
                verified = _is_signup_complete(page.url)

            return {
                "email_verified":     verified,
                "verification_code":  code,
                "captcha_public_key": public_key,
                "email_subject":      email_item.get("subject", ""),
            }

        except (CaptchaError, RegistrationError):
            raise
        except Exception as exc:
            raise RegistrationError(
                "BROWSER_ERROR",
                f"Unexpected Playwright error: {type(exc).__name__}: {exc}",
            ) from exc
        finally:
            context.close()
            browser.close()
    finally:
        pw_ctx.__exit__(None, None, None)


# ---------------------------------------------------------------------------
# Registration request parsing
# ---------------------------------------------------------------------------

def _string_option(params: dict[str, Any], spec: dict[str, Any], key: str, default: str = "") -> str:
    return str(params.get(key, spec.get(key, default))).strip()


def _email_account_id(params: dict[str, Any], spec: dict[str, Any]) -> int:
    raw_value = params.get("email_account_id", spec.get("email_account_id"))
    if raw_value is None or str(raw_value).strip() == "":
        raise ValueError("email_account_id is required")

    try:
        return int(raw_value)
    except (TypeError, ValueError) as exc:
        raise ValueError("email_account_id must be an integer") from exc


def _parse_registration_request(
    spec: dict[str, Any],
    params: dict[str, Any],
    context: dict[str, Any] | None,
) -> RegistrationRequest:
    internal_client = octo.from_context(context if isinstance(context, dict) else {})
    mode = str(params.get("mode", "api")).strip().lower() or "api"
    if mode not in _SUPPORTED_REGISTRATION_MODES:
        raise ValueError("mode must be 'api' or 'browser'")

    request = RegistrationRequest(
        mode=mode,
        username=str(params.get("username", "")).strip(),
        password=str(params.get("password", "")).strip(),
        email=str(params.get("email", "")).strip(),
        email_account_id=_email_account_id(params, spec),
        twocaptcha_api_key=_string_option(params, spec, "twocaptcha_api_key"),
        email_api_url=_string_option(params, spec, "email_api_url"),
        email_api_key=_string_option(params, spec, "email_api_key"),
        email_mailbox=str(params.get("email_mailbox", DEFAULT_EMAIL_MAILBOX)).strip() or DEFAULT_EMAIL_MAILBOX,
        wait_seconds=max(
            MIN_WAIT_SECONDS,
            min(as_int(params.get("wait_seconds", DEFAULT_WAIT_SECONDS), DEFAULT_WAIT_SECONDS), MAX_WAIT_SECONDS),
        ),
        poll_interval=max(
            MIN_POLL_INTERVAL,
            min(as_int(params.get("poll_interval", DEFAULT_POLL_INTERVAL), DEFAULT_POLL_INTERVAL), MAX_POLL_INTERVAL),
        ),
        proxy=_string_option(params, spec, "proxy"),
        headless=as_bool(params.get("headless", spec.get("headless", True)), True),
        arkose_public_key=_string_option(params, spec, "arkose_public_key"),
        arkose_subdomain=_string_option(params, spec, "arkose_subdomain"),
        internal_client=internal_client,
    )

    missing_fields = [
        field_name
        for field_name, value in (
            ("username", request.username),
            ("password", request.password),
            ("email", request.email),
            ("twocaptcha_api_key", request.twocaptcha_api_key),
        )
        if not value
    ]
    if not request.uses_internal_email_client:
        if not request.email_api_url:
            missing_fields.append("email_api_url")
        if not request.email_api_key:
            missing_fields.append("email_api_key")

    if missing_fields:
        raise ValueError(f"missing required params: {', '.join(missing_fields)}")
    return request


def _execute_registration(
    registration: RegistrationRequest,
    email_client: OctoMangerEmailClient,
) -> dict[str, Any]:
    if registration.mode == "browser":
        return _playwright_register(registration, email_client)
    return _api_register(registration, email_client)


# ---------------------------------------------------------------------------
# Main handler (called from main.py dispatch)
# ---------------------------------------------------------------------------

def handle_register(
    identifier: str, spec: dict[str, Any], params: dict[str, Any], context: dict[str, Any] | None = None
) -> dict[str, Any]:
    """REGISTER action — create a new GitHub account via API or Playwright."""
    try:
        registration = _parse_registration_request(spec, params, context)
    except ValueError as exc:
        return build_error("VALIDATION_FAILED", str(exc))

    email_client = OctoMangerEmailClient(
        registration.email_api_url,
        registration.email_api_key,
        registration.email_account_id,
        registration.internal_client,
    )
    octo.emit_log(
        "github register started",
        level="info",
        identifier=identifier,
        mode=registration.mode,
        email=registration.email,
        mailbox=registration.email_mailbox,
        wait_seconds=registration.wait_seconds,
        internal_api=registration.uses_internal_email_client,
    )

    try:
        result = _execute_registration(registration, email_client)
    except CaptchaError as exc:
        octo.emit_log("github register captcha error", level="warn", identifier=identifier, code=exc.code, detail_message=exc.message)
        return build_error(exc.code, exc.message)
    except RegistrationError as exc:
        octo.emit_log("github register failed", level="warn", identifier=identifier, code=exc.code, detail_message=exc.message)
        return build_error(exc.code, exc.message, exc.details)

    octo.emit_log(
        "github register completed",
        level="info",
        identifier=identifier,
        mode=registration.mode,
        email_verified=result["email_verified"],
    )
    return build_success({
        "event":              "github.account.registered",
        "identifier":         identifier,
        "username":           registration.username,
        "email":              registration.email,
        "mode":               registration.mode,
        "email_verified":     result["email_verified"],
        "verification_code":  result.get("verification_code"),
        "email_subject":      result.get("email_subject", ""),
        "captcha_public_key": result.get("captcha_public_key", ""),
        "note": (
            "Registration complete. "
            "Visit https://github.com/settings/tokens to create a Personal Access Token."
        ),
        "handled_at": now_utc(),
    })
