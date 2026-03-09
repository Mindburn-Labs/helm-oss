"""
HELM governance adapter for Google Agent Development Kit (ADK).

Wraps Google ADK tool functions with HELM governance using a decorator
pattern that integrates naturally with ADK's tool definition style.

Usage:
    from helm_google_adk import HelmADKGovernor

    governor = HelmADKGovernor(helm_url="http://localhost:8080")
    governed_tools = governor.govern_tools(my_tools)
"""

from __future__ import annotations

import hashlib
import json
import time
from dataclasses import dataclass
from typing import Any, Callable, Optional, Sequence

import httpx


@dataclass
class HelmADKConfig:
    helm_url: str = "http://localhost:8080"
    api_key: Optional[str] = None
    fail_closed: bool = True
    collect_receipts: bool = True
    timeout: float = 30.0


@dataclass
class ToolCallReceipt:
    tool_name: str
    args: dict[str, Any]
    receipt_id: str
    decision: str
    reason_code: str
    duration_ms: float
    request_hash: str
    output_hash: str


@dataclass
class ToolCallDenial:
    tool_name: str
    args: dict[str, Any]
    reason_code: str
    message: str


class HelmToolDenyError(Exception):
    def __init__(self, denial: ToolCallDenial):
        super().__init__(
            f'HELM denied tool "{denial.tool_name}": '
            f"{denial.reason_code} — {denial.message}"
        )
        self.denial = denial


class HelmADKGovernor:
    """Governs Google ADK tool calls through HELM."""

    def __init__(self, config: Optional[HelmADKConfig] = None, **kwargs: Any):
        if config is None:
            config = HelmADKConfig(**kwargs)
        self.config = config
        self._receipts: list[ToolCallReceipt] = []
        self._on_receipt: Optional[Callable[[ToolCallReceipt], None]] = None
        self._on_deny: Optional[Callable[[ToolCallDenial], None]] = None

        headers: dict[str, str] = {"Content-Type": "application/json"}
        if config.api_key:
            headers["Authorization"] = f"Bearer {config.api_key}"
        self._client = httpx.Client(
            base_url=config.helm_url, headers=headers, timeout=config.timeout,
        )

    def on_receipt(self, cb: Callable[[ToolCallReceipt], None]) -> "HelmADKGovernor":
        self._on_receipt = cb
        return self

    def on_deny(self, cb: Callable[[ToolCallDenial], None]) -> "HelmADKGovernor":
        self._on_deny = cb
        return self

    @property
    def receipts(self) -> list[ToolCallReceipt]:
        return list(self._receipts)

    def clear_receipts(self) -> None:
        self._receipts.clear()

    def govern_tools(self, tools: Sequence[Any]) -> list[Any]:
        return [self.govern_tool(t) for t in tools]

    def govern_tool(self, tool: Any) -> Any:
        return GovernedADKTool(tool, self)

    def _evaluate_intent(self, tool_name: str, args: dict[str, Any]) -> dict[str, Any]:
        intent = {
            "model": "helm-governance",
            "messages": [{"role": "user", "content": json.dumps({
                "type": "tool_call_intent", "tool": tool_name, "arguments": args,
            })}],
            "tools": [{"type": "function", "function": {"name": tool_name}}],
        }
        resp = self._client.post("/v1/chat/completions", json=intent)
        if resp.status_code >= 400:
            body = resp.json()
            err = body.get("error", {})
            raise HelmToolDenyError(ToolCallDenial(
                tool_name=tool_name, args=args,
                reason_code=err.get("reason_code", "ERROR_INTERNAL"),
                message=err.get("message", resp.text),
            ))
        return resp.json()

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> "HelmADKGovernor":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


class GovernedADKTool:
    """Google ADK-compatible governed tool wrapper."""

    def __init__(self, original: Any, governor: HelmADKGovernor):
        self._original = original
        self._governor = governor
        self.name: str = getattr(original, "name", getattr(original, "__name__", str(original)))
        self.description: str = getattr(original, "description", "")

    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        tool_input = kwargs if kwargs else {"input": args[0] if args else ""}
        return self._governed_execute(tool_input)

    def _governed_execute(self, args: dict[str, Any]) -> Any:
        start_ms = time.monotonic() * 1000
        try:
            response = self._governor._evaluate_intent(self.name, args)
            choices = response.get("choices", [])
            if not choices or (
                choices[0].get("finish_reason") == "stop"
                and not choices[0].get("message", {}).get("tool_calls")
            ):
                denial = ToolCallDenial(
                    tool_name=self.name, args=args,
                    reason_code="DENY_POLICY_VIOLATION",
                    message="Denied by HELM governance",
                )
                if self._governor._on_deny:
                    self._governor._on_deny(denial)
                raise HelmToolDenyError(denial)

            result = self._original(**args) if callable(self._original) else self._original.run(**args)

            duration_ms = time.monotonic() * 1000 - start_ms
            receipt = ToolCallReceipt(
                tool_name=self.name, args=args,
                receipt_id=response.get("id", ""),
                decision="APPROVED", reason_code="ALLOW",
                duration_ms=duration_ms,
                request_hash="sha256:" + hashlib.sha256(json.dumps(args, sort_keys=True).encode()).hexdigest(),
                output_hash="sha256:" + hashlib.sha256(str(result).encode()).hexdigest(),
            )
            if self._governor.config.collect_receipts:
                self._governor._receipts.append(receipt)
            if self._governor._on_receipt:
                self._governor._on_receipt(receipt)
            return result

        except HelmToolDenyError:
            raise
        except httpx.HTTPError as e:
            if self._governor.config.fail_closed:
                raise HelmToolDenyError(ToolCallDenial(
                    tool_name=self.name, args=args,
                    reason_code="ERROR_INTERNAL", message=str(e),
                )) from e
            return self._original(**args) if callable(self._original) else self._original.run(**args)
