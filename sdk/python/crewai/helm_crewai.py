"""
HELM governance adapter for CrewAI.

Drop-in governance wrapper that intercepts CrewAI agent tool calls through
HELM's policy plane. Supports fail-closed mode, receipt collection, and
multi-agent crew governance.

Usage:
    from helm_crewai import HelmCrewGovernor

    governor = HelmCrewGovernor(helm_url="http://localhost:8080")
    governed_agents = governor.govern_crew(my_crew)
"""

from __future__ import annotations

import hashlib
import json
import time
from dataclasses import dataclass, field
from typing import Any, Callable, Optional, Sequence

import httpx


@dataclass
class HelmCrewConfig:
    """Configuration for the HELM CrewAI governance."""

    helm_url: str = "http://localhost:8080"
    api_key: Optional[str] = None
    fail_closed: bool = True
    collect_receipts: bool = True
    timeout: float = 30.0


@dataclass
class ToolCallReceipt:
    """A receipt for a governed tool call."""

    tool_name: str
    agent_role: str
    args: dict[str, Any]
    receipt_id: str
    decision: str
    reason_code: str
    duration_ms: float
    request_hash: str
    output_hash: str


@dataclass
class ToolCallDenial:
    """Details of a denied tool call."""

    tool_name: str
    agent_role: str
    args: dict[str, Any]
    reason_code: str
    message: str


class HelmToolDenyError(Exception):
    """Raised when HELM denies a tool call."""

    def __init__(self, denial: ToolCallDenial):
        super().__init__(
            f'HELM denied tool "{denial.tool_name}" for agent '
            f'"{denial.agent_role}": {denial.reason_code} — {denial.message}'
        )
        self.denial = denial


class HelmCrewGovernor:
    """
    Governs CrewAI agent tool calls through HELM.

    Every tool invocation is sent to HELM for policy evaluation before
    the underlying tool is executed. Supports per-agent and per-crew
    governance policies.
    """

    def __init__(self, config: Optional[HelmCrewConfig] = None, **kwargs: Any):
        if config is None:
            config = HelmCrewConfig(**kwargs)
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

    def on_receipt(self, callback: Callable[[ToolCallReceipt], None]) -> "HelmCrewGovernor":
        """Register a callback for tool call receipts."""
        self._on_receipt = callback
        return self

    def on_deny(self, callback: Callable[[ToolCallDenial], None]) -> "HelmCrewGovernor":
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

    def govern_tool(self, tool: Any, agent_role: str = "unknown") -> Any:
        """Wrap a single CrewAI tool with HELM governance."""
        return GovernedCrewTool(tool, self, agent_role)

    def govern_tools(self, tools: Sequence[Any], agent_role: str = "unknown") -> list[Any]:
        """Wrap a list of CrewAI tools with HELM governance."""
        return [self.govern_tool(t, agent_role) for t in tools]

    def _evaluate_intent(self, tool_name: str, agent_role: str, args: dict[str, Any]) -> dict[str, Any]:
        """Send a tool call intent to HELM for policy evaluation."""
        intent = {
            "model": "helm-governance",
            "messages": [
                {
                    "role": "user",
                    "content": json.dumps({
                        "type": "tool_call_intent",
                        "tool": tool_name,
                        "arguments": args,
                        "principal": f"agent:{agent_role}",
                    }),
                }
            ],
            "tools": [{"type": "function", "function": {"name": tool_name}}],
        }
        resp = self._client.post("/v1/chat/completions", json=intent)
        if resp.status_code >= 400:
            body = resp.json()
            err = body.get("error", {})
            raise HelmToolDenyError(
                ToolCallDenial(
                    tool_name=tool_name,
                    agent_role=agent_role,
                    args=args,
                    reason_code=err.get("reason_code", "ERROR_INTERNAL"),
                    message=err.get("message", resp.text),
                )
            )
        return resp.json()

    def close(self) -> None:
        """Close the HTTP client."""
        self._client.close()

    def __enter__(self) -> "HelmCrewGovernor":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


class GovernedCrewTool:
    """CrewAI-compatible tool wrapper for HELM governance."""

    def __init__(self, original: Any, governor: HelmCrewGovernor, agent_role: str):
        self._original = original
        self._governor = governor
        self._agent_role = agent_role
        self.name: str = getattr(original, "name", str(original))
        self.description: str = getattr(original, "description", "")

    def _run(self, *args: Any, **kwargs: Any) -> Any:
        """Execute with HELM governance."""
        tool_input = kwargs if kwargs else {"input": args}
        return self._governed_execute(tool_input)

    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        return self._run(*args, **kwargs)

    def _governed_execute(self, args: dict[str, Any]) -> Any:
        """Execute the tool through HELM governance."""
        start_ms = time.monotonic() * 1000
        tool_name = self.name

        try:
            response = self._governor._evaluate_intent(tool_name, self._agent_role, args)
            choices = response.get("choices", [])
            if not choices or (
                choices[0].get("finish_reason") == "stop"
                and not choices[0].get("message", {}).get("tool_calls")
            ):
                denial = ToolCallDenial(
                    tool_name=tool_name,
                    agent_role=self._agent_role,
                    args=args,
                    reason_code="DENY_POLICY_VIOLATION",
                    message="Denied by HELM governance",
                )
                if self._governor._on_deny:
                    self._governor._on_deny(denial)
                raise HelmToolDenyError(denial)

            # Execute underlying tool.
            if hasattr(self._original, "_run"):
                result = self._original._run(**args)
            else:
                result = self._original(args)

            duration_ms = time.monotonic() * 1000 - start_ms
            receipt = ToolCallReceipt(
                tool_name=tool_name,
                agent_role=self._agent_role,
                args=args,
                receipt_id=response.get("id", ""),
                decision="APPROVED",
                reason_code="ALLOW",
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
                raise HelmToolDenyError(
                    ToolCallDenial(
                        tool_name=tool_name,
                        agent_role=self._agent_role,
                        args=args,
                        reason_code="ERROR_INTERNAL",
                        message=str(e),
                    )
                ) from e
            if hasattr(self._original, "_run"):
                return self._original._run(**args)
            return self._original(args)
