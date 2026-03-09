"""
HELM governance adapter for PydanticAI.

Wraps PydanticAI tool functions with HELM governance. Tools are intercepted
at the function call boundary and evaluated against HELM policy.

Usage:
    from helm_pydanticai import helm_tool

    @helm_tool(helm_url="http://localhost:8080")
    async def search(query: str) -> str:
        return await do_search(query)
"""

from __future__ import annotations

import functools
import hashlib
import json
import time
from dataclasses import dataclass
from typing import Any, Callable, Optional, TypeVar

import httpx

F = TypeVar("F", bound=Callable[..., Any])


@dataclass
class HelmPydanticAIConfig:
    helm_url: str = "http://localhost:8080"
    api_key: Optional[str] = None
    fail_closed: bool = True
    timeout: float = 30.0


@dataclass
class ToolCallReceipt:
    tool_name: str
    args: dict[str, Any]
    receipt_id: str
    decision: str
    reason_code: str
    duration_ms: float


class HelmToolDenyError(Exception):
    def __init__(self, tool_name: str, reason_code: str, message: str):
        super().__init__(f'HELM denied "{tool_name}": {reason_code} — {message}')
        self.tool_name = tool_name
        self.reason_code = reason_code


def helm_tool(
    helm_url: str = "http://localhost:8080",
    api_key: Optional[str] = None,
    fail_closed: bool = True,
    timeout: float = 30.0,
) -> Callable[[F], F]:
    """
    Decorator that wraps a PydanticAI tool function with HELM governance.

    Usage:
        @helm_tool(helm_url="http://localhost:8080")
        def my_tool(ctx: RunContext, query: str) -> str:
            ...
    """
    config = HelmPydanticAIConfig(
        helm_url=helm_url, api_key=api_key,
        fail_closed=fail_closed, timeout=timeout,
    )

    def decorator(fn: F) -> F:
        tool_name = fn.__name__

        @functools.wraps(fn)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            headers: dict[str, str] = {"Content-Type": "application/json"}
            if config.api_key:
                headers["Authorization"] = f"Bearer {config.api_key}"

            start_ms = time.monotonic() * 1000
            intent = {
                "model": "helm-governance",
                "messages": [{"role": "user", "content": json.dumps({
                    "type": "tool_call_intent",
                    "tool": tool_name,
                    "arguments": {k: str(v) for k, v in kwargs.items()},
                })}],
                "tools": [{"type": "function", "function": {"name": tool_name}}],
            }

            try:
                with httpx.Client(base_url=config.helm_url, headers=headers, timeout=config.timeout) as client:
                    resp = client.post("/v1/chat/completions", json=intent)
                    if resp.status_code >= 400:
                        body = resp.json()
                        err = body.get("error", {})
                        raise HelmToolDenyError(
                            tool_name, err.get("reason_code", "ERROR_INTERNAL"),
                            err.get("message", resp.text),
                        )
                    data = resp.json()
                    choices = data.get("choices", [])
                    if not choices or (
                        choices[0].get("finish_reason") == "stop"
                        and not choices[0].get("message", {}).get("tool_calls")
                    ):
                        raise HelmToolDenyError(
                            tool_name, "DENY_POLICY_VIOLATION", "Denied by HELM",
                        )
            except httpx.HTTPError as e:
                if config.fail_closed:
                    raise HelmToolDenyError(tool_name, "ERROR_INTERNAL", str(e)) from e
                # Fall through to direct execution.

            return fn(*args, **kwargs)

        return wrapper  # type: ignore[return-value]

    return decorator
