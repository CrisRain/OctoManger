#!/usr/bin/env python3
from __future__ import annotations

import base64
import hashlib
import json
import secrets
import sys
import time
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Any
from urllib import parse

import requests

import octo


SUPPORTED_DOMAINS = {"outlook.com", "hotmail.com", "live.com", "msn.com"}
DEFAULT_SIGNUP_URL = "https://signup.live.com/signup?lic=1"
DEFAULT_SCOPE = [
    "offline_access",
    "openid",
    "profile",
    "email",
    "https://graph.microsoft.com/Mail.Read",
]
DEFAULT_MAILBOX = "INBOX"
DEFAULT_GRAPH_BASE_URL = "https://graph.microsoft.com/v1.0"


class RegistrationError(RuntimeError):
    def __init__(self, code: str, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.details = details or {}


def normalize_playwright_error(exc: Exception) -> RegistrationError:
    message = str(exc)
    lowered = message.lower()
    if "executable doesn't exist" in lowered or "please run the following command to download new browsers" in lowered:
        return RegistrationError(
            "PLAYWRIGHT_BROWSER_MISSING",
            "playwright browser executable is missing; run `python -m playwright install chromium` in this module venv",
        )
    if "no module named 'playwright'" in lowered:
        return RegistrationError(
            "PLAYWRIGHT_NOT_INSTALLED",
            "playwright is not installed in this module venv; install requirements first",
        )
    if "error while loading shared libraries" in lowered or "cannot open shared object file" in lowered:
        return RegistrationError(
            "PLAYWRIGHT_SYSTEM_DEPS_MISSING",
            "playwright browser system libraries are missing in the Linux runtime image; rebuild the app image with the required shared libraries",
        )
    return RegistrationError("BROWSER_ERROR", f"unexpected Playwright error: {type(exc).__name__}: {exc}")


@dataclass
class OAuthConfig:
    client_id: str
    client_secret: str
    tenant: str
    redirect_uri: str
    scope: list[str]
    mailbox: str
    graph_base_url: str
    token_url: str


@dataclass
class BatchConfig:
    provider: str
    count: int
    prefix: str
    domain: str
    start_index: int
    status: int
    password: str
    first_name: str
    last_name: str
    birth_month: str
    birth_day: int
    birth_year: int
    country: str
    headless: bool
    challenge_timeout_seconds: int
    oauth_timeout_seconds: int
    page_timeout_seconds: int
    api_timeout_seconds: int
    slow_mo_ms: int
    proxy: str
    oauth: OAuthConfig


def as_dict(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def as_str(value: Any, default: str = "") -> str:
    if value is None:
        return default
    return str(value).strip() or default


def as_int(value: Any, default: int = 0) -> int:
    if isinstance(value, bool):
        return default
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    text = as_str(value)
    if not text:
        return default
    try:
        return int(text)
    except ValueError:
        return default


def as_bool(value: Any, default: bool = False) -> bool:
    if isinstance(value, bool):
        return value
    if value is None:
        return default
    text = str(value).strip().lower()
    if text in {"1", "true", "yes", "on"}:
        return True
    if text in {"0", "false", "no", "off"}:
        return False
    return default


def normalize_scope(raw_scope: Any) -> list[str]:
    if isinstance(raw_scope, list):
        items = [as_str(item) for item in raw_scope]
    else:
        items = as_str(raw_scope).replace(",", " ").split()
    seen: set[str] = set()
    scopes: list[str] = []
    for item in items:
        if not item or item in seen:
            continue
        scopes.append(item)
        seen.add(item)
    if scopes:
        return scopes
    return list(DEFAULT_SCOPE)


def now_utc() -> str:
    return datetime.now(timezone.utc).isoformat()


def normalize_provider(raw: str) -> str:
    provider = as_str(raw).lower()
    if provider in {"", "outlook", "hotmail", "live", "msn"}:
        return "outlook"
    return provider


def normalize_domain(raw: str) -> str:
    domain = as_str(raw, "outlook.com").lower()
    if domain not in SUPPORTED_DOMAINS:
        raise RegistrationError(
            "UNSUPPORTED_DOMAIN",
            f"unsupported outlook domain: {domain}",
            {"supported_domains": sorted(SUPPORTED_DOMAINS)},
        )
    return domain


def normalize_month(raw: str) -> str:
    month = as_str(raw, "January")
    normalized = month[:1].upper() + month[1:].lower()
    valid = {
        "January",
        "February",
        "March",
        "April",
        "May",
        "June",
        "July",
        "August",
        "September",
        "October",
        "November",
        "December",
    }
    if normalized not in valid:
        raise RegistrationError("INVALID_BIRTH_MONTH", f"invalid birth_month: {month}")
    return normalized


def build_token_url(tenant: str) -> str:
    return f"https://login.microsoftonline.com/{parse.quote(tenant, safe='')}/oauth2/v2.0/token"


def parse_oauth_config(options: dict[str, Any]) -> OAuthConfig:
    oauth = as_dict(options.get("oauth"))
    client_id = as_str(oauth.get("client_id"))
    redirect_uri = as_str(oauth.get("redirect_uri"))
    if not client_id:
        raise RegistrationError("OAUTH_CONFIG_INVALID", "options.oauth.client_id is required")
    if not redirect_uri:
        raise RegistrationError("OAUTH_CONFIG_INVALID", "options.oauth.redirect_uri is required")

    tenant = as_str(oauth.get("tenant"), "consumers")
    return OAuthConfig(
        client_id=client_id,
        client_secret=as_str(oauth.get("client_secret")),
        tenant=tenant,
        redirect_uri=redirect_uri,
        scope=normalize_scope(oauth.get("scope")),
        mailbox=as_str(oauth.get("mailbox"), DEFAULT_MAILBOX),
        graph_base_url=as_str(oauth.get("graph_base_url"), DEFAULT_GRAPH_BASE_URL),
        token_url=as_str(oauth.get("token_url"), build_token_url(tenant)),
    )


def parse_batch_config(request_payload: dict[str, Any]) -> BatchConfig:
    params = as_dict(request_payload.get("params"))
    options = as_dict(params.get("options"))
    count = as_int(params.get("count"), 0)
    if count <= 0:
        raise RegistrationError("INVALID_COUNT", "count must be greater than 0")

    provider = normalize_provider(as_str(params.get("provider")))
    if provider != "outlook":
        raise RegistrationError("UNSUPPORTED_PROVIDER", f"provider {provider!r} is not supported yet")

    password = as_str(options.get("password"))
    if not password:
        raise RegistrationError("PASSWORD_REQUIRED", "options.password is required")
    if len(password) < 8:
        raise RegistrationError("PASSWORD_TOO_SHORT", "options.password must be at least 8 characters")

    headless = as_bool(options.get("headless"), False)
    return BatchConfig(
        provider=provider,
        count=count,
        prefix=as_str(params.get("prefix"), "mail"),
        domain=normalize_domain(as_str(params.get("domain"), "outlook.com")),
        start_index=max(1, as_int(params.get("start_index"), 1)),
        status=as_int(params.get("status"), 0),
        password=password,
        first_name=as_str(options.get("first_name"), "Octo"),
        last_name=as_str(options.get("last_name"), "Manager"),
        birth_month=normalize_month(as_str(options.get("birth_month"), "January")),
        birth_day=max(1, min(31, as_int(options.get("birth_day"), 1))),
        birth_year=max(1900, min(datetime.now(timezone.utc).year-1, as_int(options.get("birth_year"), 1991))),
        country=as_str(options.get("country"), "United States"),
        headless=headless,
        challenge_timeout_seconds=max(30, as_int(options.get("challenge_timeout_seconds"), 300)),
        oauth_timeout_seconds=max(60, as_int(options.get("oauth_timeout_seconds"), 300)),
        page_timeout_seconds=max(30, as_int(options.get("page_timeout_seconds"), 90)),
        api_timeout_seconds=max(15, as_int(options.get("api_timeout_seconds"), 60)),
        slow_mo_ms=max(0, as_int(options.get("slow_mo_ms"), 0)),
        proxy=as_str(options.get("proxy")),
        oauth=parse_oauth_config(options),
    )


def password_for_account(template: str, address: str, index: int) -> str:
    return (
        template
        .replace("{email}", address)
        .replace("{local}", address.split("@", 1)[0])
        .replace("{index}", str(index))
    )


def page_text(page: Any) -> str:
    try:
        return page.locator("body").inner_text(timeout=2_000)
    except Exception:
        return ""


def button_visible(page: Any, label: str) -> bool:
    locator = page.get_by_role("button", name=label)
    try:
        return locator.count() > 0 and locator.first.is_visible()
    except Exception:
        return False


def maybe_click_button(page: Any, label: str) -> bool:
    locator = page.get_by_role("button", name=label)
    try:
        if locator.count() == 0 or not locator.first.is_visible():
            return False
        locator.first.click()
        return True
    except Exception:
        return False


def wait_for_visible(page: Any, selector: str, code: str, message: str, timeout_ms: int) -> None:
    try:
        page.locator(selector).first.wait_for(state="visible", timeout=timeout_ms)
    except Exception as exc:
        raise RegistrationError(code, message, {"body": page_text(page)[:800], "url": page.url}) from exc


def apply_page_timeouts(page: Any, action_timeout_seconds: int, navigation_timeout_seconds: int | None = None) -> None:
    action_timeout_ms = max(5, action_timeout_seconds) * 1000
    navigation_timeout_ms = max(action_timeout_ms, (navigation_timeout_seconds or action_timeout_seconds) * 1000)
    page.set_default_timeout(action_timeout_ms)
    page.set_default_navigation_timeout(navigation_timeout_ms)


def click_visible_option(page: Any, option_label: str, timeout_ms: int, exact: bool = False) -> bool:
    candidates = [
        page.get_by_role("option", name=option_label, exact=exact).first,
        page.locator('[role="option"]').filter(has_text=option_label).first,
    ]
    if exact:
        candidates.append(page.get_by_role("option", name=option_label, exact=False).first)

    for locator in candidates:
        try:
            locator.wait_for(state="visible", timeout=min(timeout_ms, 5_000))
            locator.click(timeout=min(timeout_ms, 5_000))
            return True
        except Exception:
            continue
    return False


def select_fluent_option(page: Any, trigger_selector: str, option_label: str, timeout_ms: int, exact: bool = False) -> None:
    deadline = time.time() + max(1, timeout_ms) / 1000.0
    last_error: Exception | None = None

    while time.time() < deadline:
        try:
            trigger = page.locator(trigger_selector).first
            trigger.wait_for(state="visible", timeout=timeout_ms)
            trigger.click(force=True, timeout=min(timeout_ms, 5_000))
            if click_visible_option(page, option_label, timeout_ms, exact=exact):
                page.wait_for_timeout(150)
                return
        except Exception as exc:
            last_error = exc
        try:
            page.keyboard.press("Escape")
        except Exception:
            pass
        page.wait_for_timeout(300)

    raise RegistrationError(
        "SIGNUP_DROPDOWN_SELECT_FAILED",
        f"failed to select dropdown option: {option_label}",
        {"selector": trigger_selector, "option": option_label, "url": page.url, "body": page_text(page)[:800]},
    ) from last_error


def complete_signup(page: Any, cfg: BatchConfig, address: str, password: str, index: int) -> None:
    timeout_ms = cfg.page_timeout_seconds * 1000

    page.goto(DEFAULT_SIGNUP_URL, wait_until="domcontentloaded", timeout=timeout_ms)
    wait_for_visible(page, "input[name=Email]", "SIGNUP_LOAD_FAILED", "failed to load outlook signup page", timeout_ms)
    page.fill("input[name=Email]", address)
    maybe_click_button(page, "Next")

    wait_for_visible(page, "input[type=password]", "SIGNUP_EMAIL_REJECTED", "outlook signup did not accept the target email address", timeout_ms)
    page.fill("input[type=password]", password)
    maybe_click_button(page, "Next")

    wait_for_visible(page, "#BirthMonthDropdown", "SIGNUP_PASSWORD_REJECTED", "outlook signup did not accept the password", timeout_ms)
    if cfg.country and cfg.country != "United States":
        select_fluent_option(page, "#countryDropdownId", cfg.country, timeout_ms)
    select_fluent_option(page, "#BirthMonthDropdown", cfg.birth_month, timeout_ms)
    select_fluent_option(page, "#BirthDayDropdown", str(cfg.birth_day), timeout_ms, exact=True)
    page.fill("input[name=BirthYear]", str(cfg.birth_year))
    maybe_click_button(page, "Next")

    wait_for_visible(page, "#firstNameInput", "SIGNUP_DETAILS_REJECTED", "outlook signup did not accept the birth details", timeout_ms)
    page.fill("#firstNameInput", cfg.first_name)
    page.fill("#lastNameInput", cfg.last_name)
    octo.emit_log(
        "outlook signup submitted base profile",
        level="info",
        address=address,
        index=index,
        headless=cfg.headless,
    )
    maybe_click_button(page, "Next")

    wait_for_human_verification(page, address, cfg.challenge_timeout_seconds)


def wait_for_human_verification(page: Any, address: str, timeout_seconds: int) -> None:
    deadline = time.time() + timeout_seconds
    logged_waiting = False

    while time.time() < deadline:
        body = page_text(page)
        if "Let's prove you're human" in body or "Press and hold" in body:
            if not logged_waiting:
                octo.emit_log(
                    "outlook signup waiting for human verification",
                    level="info",
                    address=address,
                    timeout_seconds=timeout_seconds,
                )
                logged_waiting = True
            page.wait_for_timeout(1_000)
            continue

        if body.strip():
            octo.emit_log("outlook signup human verification completed", level="info", address=address)
            return
        page.wait_for_timeout(500)

    raise RegistrationError(
        "HUMAN_VERIFICATION_TIMEOUT",
        "manual human verification did not complete before timeout",
        {"address": address, "timeout_seconds": timeout_seconds},
    )


def create_pkce_pair() -> tuple[str, str]:
    verifier = secrets.token_urlsafe(64)
    digest = hashlib.sha256(verifier.encode("utf-8")).digest()
    challenge = base64.urlsafe_b64encode(digest).decode("ascii").rstrip("=")
    return verifier, challenge


def build_authorize_url(oauth: OAuthConfig, address: str, code_challenge: str, state: str) -> str:
    query = parse.urlencode(
        {
            "client_id": oauth.client_id,
            "response_type": "code",
            "redirect_uri": oauth.redirect_uri,
            "response_mode": "query",
            "scope": " ".join(oauth.scope),
            "state": state,
            "login_hint": address,
            "code_challenge": code_challenge,
            "code_challenge_method": "S256",
        }
    )
    return f"https://login.microsoftonline.com/{parse.quote(oauth.tenant, safe='')}/oauth2/v2.0/authorize?{query}"


def maybe_fill(page: Any, selector: str, value: str) -> bool:
    locator = page.locator(selector)
    try:
        if locator.count() == 0 or not locator.first.is_visible():
            return False
        locator.first.fill(value)
        return True
    except Exception:
        return False


def maybe_click_selector(page: Any, selector: str) -> bool:
    locator = page.locator(selector)
    try:
        if locator.count() == 0 or not locator.first.is_visible():
            return False
        locator.first.click()
        return True
    except Exception:
        return False


def authorize_with_existing_session(
    page: Any,
    oauth: OAuthConfig,
    address: str,
    password: str,
    timeout_seconds: int,
    api_timeout_seconds: int,
) -> dict[str, Any]:
    verifier, challenge = create_pkce_pair()
    state = secrets.token_urlsafe(18)
    authorize_url = build_authorize_url(oauth, address, challenge, state)
    deadline = time.time() + timeout_seconds
    login_filled = False
    password_filled = False
    consent_logged = False

    page.goto(authorize_url, wait_until="domcontentloaded", timeout=timeout_seconds * 1000)
    while time.time() < deadline:
        current_url = page.url
        if current_url.startswith(oauth.redirect_uri):
            parsed = parse.urlparse(current_url)
            query = parse.parse_qs(parsed.query)
            error_value = as_str((query.get("error") or [""])[0])
            if error_value:
                description = as_str((query.get("error_description") or [""])[0], error_value)
                raise RegistrationError("OAUTH_AUTHORIZE_FAILED", description)
            code = as_str((query.get("code") or [""])[0])
            state_value = as_str((query.get("state") or [""])[0])
            if not code:
                raise RegistrationError("OAUTH_CODE_MISSING", "oauth redirect did not include code")
            if state_value and state_value != state:
                raise RegistrationError("OAUTH_STATE_MISMATCH", "oauth state mismatch")
            return exchange_code(oauth, code, verifier, api_timeout_seconds)

        if maybe_fill(page, "input[name=loginfmt]", address) or maybe_fill(page, "input[type=email]", address):
            maybe_click_selector(page, "#idSIButton9")
            maybe_click_button(page, "Next")
            login_filled = True
            page.wait_for_timeout(1_200)
            continue

        if maybe_fill(page, "input[name=passwd]", password) or maybe_fill(page, "input[type=password]", password):
            maybe_click_selector(page, "#idSIButton9")
            maybe_click_button(page, "Sign in")
            password_filled = True
            page.wait_for_timeout(1_500)
            continue

        if maybe_click_selector(page, "#acceptButton") or maybe_click_button(page, "Accept") or maybe_click_button(page, "Continue") or maybe_click_button(page, "Yes"):
            if not consent_logged:
                octo.emit_log("outlook oauth consent submitted", level="info", address=address)
                consent_logged = True
            page.wait_for_timeout(1_500)
            continue

        if maybe_click_selector(page, "#idBtn_Back") or maybe_click_button(page, "No"):
            page.wait_for_timeout(1_200)
            continue

        body = page_text(page)
        if "Stay signed in?" in body:
            if maybe_click_selector(page, "#idBtn_Back") or maybe_click_button(page, "No") or maybe_click_selector(page, "#idSIButton9"):
                page.wait_for_timeout(1_200)
                continue
        if "Pick an account" in body:
            if page.get_by_text(address, exact=False).count() > 0:
                page.get_by_text(address, exact=False).first.click()
                page.wait_for_timeout(1_200)
                continue

        page.wait_for_timeout(800)

    raise RegistrationError(
        "OAUTH_TIMEOUT",
        "failed to complete outlook oauth flow before timeout",
        {
            "login_prompt_seen": login_filled,
            "password_prompt_seen": password_filled,
            "url": page.url,
            "body": page_text(page)[:800],
        },
    )


def exchange_code(oauth: OAuthConfig, code: str, verifier: str, timeout_seconds: int) -> dict[str, Any]:
    form = {
        "client_id": oauth.client_id,
        "grant_type": "authorization_code",
        "code": code,
        "redirect_uri": oauth.redirect_uri,
        "scope": " ".join(oauth.scope),
        "code_verifier": verifier,
    }
    if oauth.client_secret:
        form["client_secret"] = oauth.client_secret

    response = requests.post(
        oauth.token_url,
        data=form,
        timeout=max(15, timeout_seconds),
    )
    try:
        payload = response.json()
    except Exception as exc:
        raise RegistrationError("TOKEN_RESPONSE_INVALID", f"failed to parse token response: {exc}") from exc

    if response.status_code < 200 or response.status_code >= 300:
        error_value = as_str(payload.get("error"), f"http_{response.status_code}")
        description = as_str(payload.get("error_description"), error_value)
        raise RegistrationError("TOKEN_EXCHANGE_FAILED", description, {"error": error_value})

    access_token = as_str(payload.get("access_token"))
    refresh_token = as_str(payload.get("refresh_token"))
    if not access_token:
        raise RegistrationError("ACCESS_TOKEN_MISSING", "token response did not include access_token")
    if not refresh_token:
        raise RegistrationError("REFRESH_TOKEN_MISSING", "token response did not include refresh_token; ensure offline_access is enabled")

    expires_in = as_int(payload.get("expires_in"), 3600)
    expires_at = (datetime.now(timezone.utc) + timedelta(seconds=expires_in)).isoformat()
    return {
        "token_type": as_str(payload.get("token_type"), "Bearer"),
        "scope": as_str(payload.get("scope"), " ".join(oauth.scope)),
        "expires_in": expires_in,
        "expires_at": expires_at,
        "token_url": oauth.token_url,
        "access_token": access_token,
        "refresh_token": refresh_token,
    }


def build_graph_config(oauth: OAuthConfig, address: str, token: dict[str, Any]) -> dict[str, Any]:
    config: dict[str, Any] = {
        "auth_method": "graph_oauth2",
        "username": address,
        "client_id": oauth.client_id,
        "refresh_token": token["refresh_token"],
        "tenant": oauth.tenant,
        "scope": normalize_scope(token.get("scope")),
        "token_url": as_str(token.get("token_url"), oauth.token_url),
        "graph_base_url": oauth.graph_base_url,
        "mailbox": oauth.mailbox,
        "access_token": token["access_token"],
        "token_expires_at": token["expires_at"],
    }
    if oauth.client_secret:
        config["client_secret"] = oauth.client_secret
    return config


def register_one(browser, cfg: BatchConfig, index: int) -> dict[str, Any]:
    address = f"{cfg.prefix}{cfg.start_index + index}@{cfg.domain}"
    password = password_for_account(cfg.password, address, cfg.start_index + index)
    context = browser.new_context(locale="en-US", viewport={"width": 1440, "height": 960})
    page = context.new_page()
    try:
        apply_page_timeouts(page, cfg.page_timeout_seconds, max(cfg.page_timeout_seconds, cfg.oauth_timeout_seconds))
        complete_signup(page, cfg, address, password, index)
        token = authorize_with_existing_session(
            page,
            cfg.oauth,
            address,
            password,
            cfg.oauth_timeout_seconds,
            cfg.api_timeout_seconds,
        )
        octo.emit_log("outlook account registered", level="info", address=address, index=index)
        return {
            "index": index,
            "address": address,
            "provider": "outlook",
            "status": cfg.status,
            "graph_config": build_graph_config(cfg.oauth, address, token),
        }
    finally:
        context.close()


def handle_batch_register(request_payload: dict[str, Any]) -> dict[str, Any]:
    cfg = parse_batch_config(request_payload)
    failures: list[dict[str, Any]] = []
    generated: list[dict[str, Any]] = []
    octo.emit_log(
        "outlook batch register configuration loaded",
        level="info",
        requested=cfg.count,
        page_timeout_seconds=cfg.page_timeout_seconds,
        oauth_timeout_seconds=cfg.oauth_timeout_seconds,
        api_timeout_seconds=cfg.api_timeout_seconds,
        challenge_timeout_seconds=cfg.challenge_timeout_seconds,
        headless=cfg.headless,
    )

    try:
        from playwright.sync_api import sync_playwright  # noqa: PLC0415
    except Exception as exc:
        raise normalize_playwright_error(exc) from exc

    launch_options: dict[str, Any] = {
        "headless": cfg.headless,
    }
    if cfg.slow_mo_ms > 0:
        launch_options["slow_mo"] = cfg.slow_mo_ms
    if cfg.proxy:
        launch_options["proxy"] = {"server": cfg.proxy}

    try:
        with sync_playwright() as playwright:
            browser = playwright.chromium.launch(**launch_options)
            try:
                for index in range(cfg.count):
                    try:
                        generated.append(register_one(browser, cfg, index))
                    except RegistrationError as exc:
                        address = f"{cfg.prefix}{cfg.start_index + index}@{cfg.domain}"
                        octo.emit_log(
                            "outlook account registration failed",
                            level="warn",
                            address=address,
                            index=index,
                            code=exc.code,
                            detail_message=exc.message,
                        )
                        failures.append(
                            {
                                "index": index,
                                "address": address,
                                "code": exc.code,
                                "message": exc.message,
                            }
                        )
            finally:
                browser.close()
    except RegistrationError:
        raise
    except Exception as exc:
        raise normalize_playwright_error(exc) from exc

    return octo.success(
        {
            "provider": cfg.provider,
            "requested": cfg.count,
            "generated": generated,
            "failures": failures,
            "completed_at": now_utc(),
        }
    )


def execute(request_payload: dict[str, Any]) -> dict[str, Any]:
    action = as_str(request_payload.get("action")).upper()
    if action == "WATCH":
        account = as_dict(request_payload.get("account"))
        identifier = as_str(account.get("identifier"), "batch-operator")
        params = as_dict(request_payload.get("params"))
        interval_seconds = max(1, as_int(params.get("interval_seconds"), 60))
        stop_after_seconds = max(0, as_int(params.get("stop_after_seconds"), 0))
        emit_heartbeat = as_bool(params.get("emit_heartbeat"), False)

        octo.emit_daemon_init_ok(
            "outlook batch daemon started",
            identifier=identifier,
            interval_seconds=interval_seconds,
        )

        started_at = time.time()
        heartbeat_count = 0
        while True:
            if stop_after_seconds and time.time() - started_at >= stop_after_seconds:
                octo.emit_daemon_done(
                    "outlook batch daemon stopping",
                    identifier=identifier,
                    heartbeat_count=heartbeat_count,
                )
                return {"status": "done"}

            if emit_heartbeat:
                heartbeat_count += 1
                octo.emit_daemon_event(
                    {
                        "event": "outlook.batch.daemon.heartbeat",
                        "identifier": identifier,
                        "count": heartbeat_count,
                        "handled_at": now_utc(),
                    }
                )

            time.sleep(interval_seconds)

    if action != "BATCH_REGISTER_EMAIL":
        return octo.error("UNSUPPORTED_ACTION", f"unsupported action: {action}")

    try:
        return handle_batch_register(request_payload)
    except RegistrationError as exc:
        octo.emit_log("outlook batch register error", level="warn", code=exc.code, detail_message=exc.message)
        return octo.error(exc.code, exc.message, exc.details)
    except Exception as exc:
        octo.emit_log("outlook batch register unexpected error", level="error", detail_message=str(exc))
        return octo.error("UNEXPECTED_ERROR", str(exc))


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8")
    return octo.run_module(execute)


if __name__ == "__main__":
    raise SystemExit(main())
