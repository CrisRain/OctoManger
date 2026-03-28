from __future__ import annotations

from ._client import OctoClient, as_dict, from_context


class ModuleRequest:
    def __init__(self, payload: dict[str, object]) -> None:
        request = as_dict(payload)
        self.raw: dict[str, object] = request
        self.action: str = str(request.get("action", "")).strip().upper()
        # Support both formats:
        #   - Legacy/direct: account and params at the top level
        #   - Go job/execute: account and params nested inside "input"
        raw_input = as_dict(request.get("input"))
        self.account: dict[str, object] = as_dict(request.get("account") or raw_input.get("account"))
        self.spec: dict[str, object] = as_dict(self.account.get("spec"))
        self.params: dict[str, object] = as_dict(request.get("params") or raw_input.get("params"))
        self.context: dict[str, object] = as_dict(request.get("context"))
        self.identifier: str = str(self.account.get("identifier", "")).strip()
        self.account_id: int | None = _parse_account_id(self.account.get("id"))
        self._loaded_account: dict[str, object] | None = None
        self._account_loaded: bool = False
        self._account_lookup_error: str = ""

    @property
    def request_id(self) -> str:
        return str(self.context.get("request_id", "")).strip()

    @property
    def protocol(self) -> str:
        return str(self.context.get("protocol", "")).strip()

    def client(self) -> OctoClient | None:
        return from_context(self.context)

    def load_account(self, *, type_key: str | None = None) -> dict[str, object] | None:
        if self._account_loaded:
            if self._loaded_account is not None:
                return dict(self._loaded_account)
            return None

        octo_client = self.client()
        resolved_type_key = str(type_key or self.context.get("plugin_key", "")).strip()

        # Persisted accounts must always be resolved from internal API.
        if self.account_id is not None:
            if octo_client is None:
                self._account_lookup_error = "internal api client is unavailable (missing api_url/api_token)"
                self._account_loaded = True
                return None

            lookup_errors: list[str] = []
            try:
                account = octo_client.get_account_or_raise(self.account_id)
                loaded = as_dict(account)
                if loaded:
                    self._loaded_account = loaded
                    self._account_lookup_error = ""
                    self._account_loaded = True
                    return dict(loaded)
                lookup_errors.append("query account by id returned empty payload")
            except Exception as exc:
                lookup_errors.append(f"query account by id failed: {exc}")

            if self.identifier and resolved_type_key:
                try:
                    account = octo_client.get_account_by_identifier_or_raise(resolved_type_key, self.identifier)
                    loaded = as_dict(account)
                    if loaded:
                        self._loaded_account = loaded
                        self._account_lookup_error = ""
                        self._account_loaded = True
                        return dict(loaded)
                    lookup_errors.append("query account by identifier returned empty payload")
                except Exception as exc:
                    lookup_errors.append(f"query account by identifier failed: {exc}")

            self._account_lookup_error = "; ".join(lookup_errors) if lookup_errors else "account not found"
            self._account_loaded = True
            return None

        if octo_client is not None and self.identifier and resolved_type_key:
            account = octo_client.get_account_by_identifier(resolved_type_key, self.identifier)
            if account:
                loaded = as_dict(account)
                self._loaded_account = loaded
                self._account_lookup_error = ""
                self._account_loaded = True
                return dict(loaded)

        fallback = as_dict(self.account)
        if fallback:
            self._loaded_account = fallback
            self._account_lookup_error = ""
            self._account_loaded = True
            return dict(fallback)
        self._account_loaded = True
        return None

    def load_spec(self, *, type_key: str | None = None) -> dict[str, object]:
        account = self.load_account(type_key=type_key)
        if account:
            loaded = as_dict(account.get("spec"))
            if loaded:
                return loaded

        if self.account_id is not None:
            return {}

        if self.spec:
            return dict(self.spec)
        return {}

    def account_lookup_error(self) -> str:
        return self._account_lookup_error

    def has_loaded_account(self) -> bool:
        return self._account_loaded and self._loaded_account is not None

    def require_identifier(self) -> str:
        if self.identifier:
            return self.identifier
        raise ValueError("account.identifier is required")


def request(payload: dict[str, object]) -> ModuleRequest:
    return ModuleRequest(payload)


def _parse_account_id(value: object) -> int | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        if value.is_integer():
            return int(value)
        return None
    if isinstance(value, str):
        text = value.strip()
        if not text:
            return None
        try:
            parsed = int(text)
        except ValueError:
            return None
        return parsed
    return None
