from __future__ import annotations

from collections.abc import Callable

from ._request import ModuleRequest
from ._response import error


class ActionRouter:
    def __init__(self) -> None:
        self._handlers: dict[str, Callable[[ModuleRequest], dict[str, object]]] = {}

    def action(
        self,
        action: str,
    ) -> Callable[[Callable[[ModuleRequest], dict[str, object]]], Callable[[ModuleRequest], dict[str, object]]]:
        normalized = str(action).strip().upper()
        if not normalized:
            raise ValueError("action is required")

        def decorator(handler: Callable[[ModuleRequest], dict[str, object]]) -> Callable[[ModuleRequest], dict[str, object]]:
            self._handlers[normalized] = handler
            return handler

        return decorator

    def add(self, action: str, handler: Callable[[ModuleRequest], dict[str, object]]) -> None:
        normalized = str(action).strip().upper()
        if not normalized:
            raise ValueError("action is required")
        self._handlers[normalized] = handler

    def dispatch(self, payload: dict[str, object]) -> dict[str, object]:
        req = ModuleRequest(payload)
        handler = self._handlers.get(req.action)
        if handler is None:
            return error("UNSUPPORTED_ACTION", f"unsupported action: {req.action}")
        try:
            output = handler(req)
        except ValueError as exc:
            return error("VALIDATION_FAILED", str(exc))
        if not isinstance(output, dict):
            return error("BAD_OUTPUT", "handler must return a JSON object")
        return output


def make_router() -> ActionRouter:
    return ActionRouter()
