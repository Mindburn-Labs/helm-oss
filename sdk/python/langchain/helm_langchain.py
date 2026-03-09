"""
HELM governance adapter for LangChain.

Drop-in tool wrapper that routes LangChain tool calls through HELM's
governance plane. Supports fail-closed mode, receipt collection, and
async execution.

Usage:
    from helm_langchain import HelmToolWrapper

    wrapper = HelmToolWrapper(helm_url="http://localhost:8080")
    governed_tools = wrapper.wrap_tools(my_tools)
    agent = create_react_agent(llm, governed_tools)
"""

from __future__ import annotations

import hashlib
import json
import time
from dataclasses import dataclass, field
from typing import Any, Callable, Optional, Sequence

import httpx


@dataclass
class HelmToolWrapperConfig:
    """Configuration for the HELM tool wrapper."""

    helm_url: str = "http://localhost:8080"
    api_key: Optional[str] = None
    fail_closed: bool = True
    collect_receipts: bool = True
    timeout: float = 30.0


@dataclass
class ToolCallReceipt:
    """A receipt for a governed tool call."""

    tool_name: str
    args: dict[str, Any]
    receipt_id: str
    decision: str  # "APPROVED" | "DENIED"
    reason_code: str
    duration_ms: float
    request_hash: str
    output_hash: str


@dataclass
class ToolCallDenial:
    """Details of a denied tool call."""

    tool_name: str
    args: dict[str, Any]
    reason_code: str
    message: str


class HelmToolDenyError(Exception):
    """Raised when HELM denies a tool call."""

    def __init__(self, denial: ToolCallDenial):
        super().__init__(
            f'HELM denied tool call "{denial.tool_name}": '
            f"{denial.reason_code} — {denial.message}"
        )
        self.denial = denial


class HelmToolWrapper:
    """
    Wraps LangChain tools with HELM governance.

    Each tool call is sent to HELM for policy evaluation before the
    underlying tool is executed. If denied, a HelmToolDenyError is raised.
    """

    def __init__(self, config: Optional[HelmToolWrapperConfig] = None, **kwargs: Any):
        if config is None:
            config = HelmToolWrapperConfig(**kwargs)
        self.config = config
        self._receipts: list[ToolCallReceipt] = []
        self._on_receipt: Optional[Callable[[ToolCallReceipt], None]] = None
        self._on_deny: Optional[Callable[[ToolCallDenial], None]] = None

        headers: dict[str, str] = {"Content-Type": "application/json"}
        if config.api_key:
            headers["Authorization"] = f"Bearer {config.api_key}"
        self._client = httpx.Client(
            base_url=config.helm_url,
            headers=headers,
            timeout=config.timeout,
        )

    def on_receipt(self, callback: Callable[[ToolCallReceipt], None]) -> "HelmToolWrapper":
        """Register a callback for tool call receipts."""
        self._on_receipt = callback
        return self

    def on_deny(self, callback: Callable[[ToolCallDenial], None]) -> "HelmToolWrapper":
        """Register a callback for denied tool calls."""
        self._on_deny = callback
        return self

    @property
    def receipts(self) -> list[ToolCallReceipt]:
        """Get all collected receipts."""
        return list(self._receipts)

    def clear_receipts(self) -> None:
        """Clear collected receipts."""
        self._receipts.clear()

    def wrap_tools(self, tools: Sequence[Any]) -> list[Any]:
        """
        Wrap a list of LangChain tools with HELM governance.

        Compatible with:
        - langchain_core.tools.BaseTool
        - @tool decorated functions
        - Any object with name and _run or invoke methods
        """
        return [self.wrap_tool(tool) for tool in tools]

    def wrap_tool(self, tool: Any) -> Any:
        """Wrap a single LangChain tool with HELM governance."""
        return GovernedTool(tool, self)

    def _evaluate_intent(self, tool_name: str, args: dict[str, Any]) -> dict[str, Any]:
        """Send a tool call intent to HELM for policy evaluation."""
        intent = {
            "model": "helm-governance",
            "messages": [
                {
                    "role": "user",
                    "content": json.dumps(
                        {
                            "type": "tool_call_intent",
                            "tool": tool_name,
                            "arguments": args,
                        }
                    ),
                }
            ],
            "tools": [
                {
                    "type": "function",
                    "function": {"name": tool_name},
                }
            ],
        }
        resp = self._client.post("/v1/chat/completions", json=intent)
        if resp.status_code >= 400:
            body = resp.json()
            err = body.get("error", {})
            raise HelmToolDenyError(
                ToolCallDenial(
                    tool_name=tool_name,
                    args=args,
                    reason_code=err.get("reason_code", "ERROR_INTERNAL"),
                    message=err.get("message", resp.text),
                )
            )
        return resp.json()

    def close(self) -> None:
        """Close the HTTP client."""
        self._client.close()

    def __enter__(self) -> "HelmToolWrapper":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


class GovernedTool:
    """
    A LangChain-compatible tool wrapper that routes execution through HELM.

    Implements the same interface as LangChain's BaseTool:
    - name, description properties
    - _run / invoke / __call__ methods
    """

    def __init__(self, original: Any, wrapper: HelmToolWrapper):
        self._original = original
        self._wrapper = wrapper
        # Preserve LangChain tool metadata.
        self.name: str = getattr(original, "name", str(original))
        self.description: str = getattr(original, "description", "")
        self.args_schema = getattr(original, "args_schema", None)

    def _run(self, *args: Any, **kwargs: Any) -> Any:
        """Execute with HELM governance (synchronous)."""
        return self._governed_execute(kwargs if kwargs else {"input": args})

    def invoke(self, input: Any, config: Any = None, **kwargs: Any) -> Any:
        """LangChain invoke interface."""
        tool_input = input if isinstance(input, dict) else {"input": input}
        return self._governed_execute(tool_input)

    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        return self._run(*args, **kwargs)

    def _governed_execute(self, args: dict[str, Any]) -> Any:
        """Execute the tool through HELM governance."""
        start_ms = time.monotonic() * 1000
        tool_name = self.name

        try:
            # Step 1: Evaluate intent through HELM.
            response = self._wrapper._evaluate_intent(tool_name, args)

            # Step 2: Check if approved.
            choices = response.get("choices", [])
            if not choices:
                denial = ToolCallDenial(
                    tool_name=tool_name,
                    args=args,
                    reason_code="DENY_POLICY_VIOLATION",
                    message="No response from HELM governance",
                )
                if self._wrapper._on_deny:
                    self._wrapper._on_deny(denial)
                raise HelmToolDenyError(denial)

            choice = choices[0]
            if choice.get("finish_reason") == "stop" and not choice.get("message", {}).get(
                "tool_calls"
            ):
                denial = ToolCallDenial(
                    tool_name=tool_name,
                    args=args,
                    reason_code="DENY_POLICY_VIOLATION",
                    message=choice.get("message", {}).get("content", "Denied by HELM"),
                )
                if self._wrapper._on_deny:
                    self._wrapper._on_deny(denial)
                raise HelmToolDenyError(denial)

            # Step 3: Execute the actual tool.
            if hasattr(self._original, "invoke"):
                result = self._original.invoke(args)
            elif hasattr(self._original, "_run"):
                result = self._original._run(**args)
            else:
                result = self._original(args)

            # Step 4: Build receipt.
            duration_ms = time.monotonic() * 1000 - start_ms
            request_hash = "sha256:" + hashlib.sha256(
                json.dumps(args, sort_keys=True).encode()
            ).hexdigest()
            output_hash = "sha256:" + hashlib.sha256(
                str(result).encode()
            ).hexdigest()

            receipt = ToolCallReceipt(
                tool_name=tool_name,
                args=args,
                receipt_id=response.get("id", ""),
                decision="APPROVED",
                reason_code="ALLOW",
                duration_ms=duration_ms,
                request_hash=request_hash,
                output_hash=output_hash,
            )

            if self._wrapper.config.collect_receipts:
                self._wrapper._receipts.append(receipt)
            if self._wrapper._on_receipt:
                self._wrapper._on_receipt(receipt)

            return result

        except HelmToolDenyError:
            raise
        except httpx.HTTPError as e:
            if self._wrapper.config.fail_closed:
                denial = ToolCallDenial(
                    tool_name=tool_name,
                    args=args,
                    reason_code="ERROR_INTERNAL",
                    message=str(e),
                )
                raise HelmToolDenyError(denial) from e
            # Fall through to direct execution.
            if hasattr(self._original, "invoke"):
                return self._original.invoke(args)
            elif hasattr(self._original, "_run"):
                return self._original._run(**args)
            return self._original(args)
