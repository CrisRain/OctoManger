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
        raw_account_id = self.account.get("id")
        self.account_id: int | None = int(raw_account_id) if isinstance(raw_account_id, int) else None

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


def request(payload: dict[str, object]) -> ModuleRequest:
    return ModuleRequest(payload)
