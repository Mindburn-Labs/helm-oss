"""
Tests for HELM LangChain adapter.
"""

from __future__ import annotations

import json
from typing import Any
from unittest.mock import MagicMock

import pytest

from helm_langchain import (
    GovernedTool,
    HelmToolDenyError,
    HelmToolWrapper,
    HelmToolWrapperConfig,
    ToolCallDenial,
)


class MockTool:
    """A mock LangChain-compatible tool."""

    def __init__(self, name: str = "calculator", result: Any = "42"):
        self.name = name
        self.description = f"Mock {name} tool"
        self._result = result

    def _run(self, **kwargs: Any) -> Any:
        return self._result

    def invoke(self, input: Any, config: Any = None) -> Any:
        return self._result


class MockHelmServer:
    """Mocks HELM API responses for testing."""

    @staticmethod
    def approved_response(tool_name: str = "calculator") -> dict[str, Any]:
        return {
            "id": "helm-test-123",
            "object": "chat.completion",
            "created": 1234567890,
            "model": "helm-governance",
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": None,
                        "tool_calls": [
                            {
                                "id": "call_1",
                                "type": "function",
                                "function": {
                                    "name": tool_name,
                                    "arguments": "{}",
                                },
                            }
                        ],
                    },
                    "finish_reason": "tool_calls",
                }
            ],
        }

    @staticmethod
    def denied_response(reason: str = "DENY_POLICY_VIOLATION") -> dict[str, Any]:
        return {
            "id": "helm-denied-123",
            "object": "chat.completion",
            "created": 1234567890,
            "model": "helm-governance",
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": f"Denied: {reason}",
                    },
                    "finish_reason": "stop",
                }
            ],
        }


def test_governed_tool_approved(monkeypatch: pytest.MonkeyPatch) -> None:
    """Tool call is approved and executed."""
    config = HelmToolWrapperConfig(helm_url="http://localhost:8080")
    wrapper = HelmToolWrapper(config=config)

    monkeypatch.setattr(
        wrapper,
        "_evaluate_intent",
        lambda name, args: MockHelmServer.approved_response(name),
    )

    tool = MockTool(name="calculator", result="42")
    governed = wrapper.wrap_tool(tool)

    result = governed.invoke({"expression": "6*7"})
    assert result == "42"
    assert len(wrapper.receipts) == 1
    assert wrapper.receipts[0].decision == "APPROVED"
    assert wrapper.receipts[0].tool_name == "calculator"


def test_governed_tool_denied(monkeypatch: pytest.MonkeyPatch) -> None:
    """Tool call is denied by HELM."""
    config = HelmToolWrapperConfig(helm_url="http://localhost:8080")
    wrapper = HelmToolWrapper(config=config)

    monkeypatch.setattr(
        wrapper,
        "_evaluate_intent",
        lambda name, args: MockHelmServer.denied_response(),
    )

    deny_callback = MagicMock()
    wrapper.on_deny(deny_callback)

    tool = MockTool(name="dangerous_tool")
    governed = wrapper.wrap_tool(tool)

    with pytest.raises(HelmToolDenyError) as exc_info:
        governed.invoke({"action": "rm -rf /"})

    assert "DENY_POLICY_VIOLATION" in str(exc_info.value)
    deny_callback.assert_called_once()


def test_receipt_collection(monkeypatch: pytest.MonkeyPatch) -> None:
    """Receipts are collected for approved calls."""
    config = HelmToolWrapperConfig(helm_url="http://localhost:8080")
    wrapper = HelmToolWrapper(config=config)

    monkeypatch.setattr(
        wrapper,
        "_evaluate_intent",
        lambda name, args: MockHelmServer.approved_response(name),
    )

    receipt_callback = MagicMock()
    wrapper.on_receipt(receipt_callback)

    tool = MockTool()
    governed = wrapper.wrap_tool(tool)
    governed.invoke({"x": 1})
    governed.invoke({"x": 2})

    assert len(wrapper.receipts) == 2
    assert receipt_callback.call_count == 2

    wrapper.clear_receipts()
    assert len(wrapper.receipts) == 0


def test_receipt_hashes(monkeypatch: pytest.MonkeyPatch) -> None:
    """Receipt request/output hashes are deterministic."""
    config = HelmToolWrapperConfig(helm_url="http://localhost:8080")
    wrapper = HelmToolWrapper(config=config)

    monkeypatch.setattr(
        wrapper,
        "_evaluate_intent",
        lambda name, args: MockHelmServer.approved_response(name),
    )

    tool = MockTool(result="42")
    governed = wrapper.wrap_tool(tool)
    governed.invoke({"expression": "6*7"})
    governed.invoke({"expression": "6*7"})

    r1, r2 = wrapper.receipts
    assert r1.request_hash == r2.request_hash
    assert r1.output_hash == r2.output_hash
    assert r1.request_hash.startswith("sha256:")


def test_wrap_tools(monkeypatch: pytest.MonkeyPatch) -> None:
    """wrap_tools wraps multiple tools."""
    config = HelmToolWrapperConfig(helm_url="http://localhost:8080")
    wrapper = HelmToolWrapper(config=config)

    tools = [MockTool("calc"), MockTool("search")]
    governed = wrapper.wrap_tools(tools)

    assert len(governed) == 2
    assert governed[0].name == "calc"
    assert governed[1].name == "search"


def test_governed_tool_preserves_metadata() -> None:
    """GovernedTool preserves original tool metadata."""
    config = HelmToolWrapperConfig(helm_url="http://localhost:8080")
    wrapper = HelmToolWrapper(config=config)

    tool = MockTool(name="my_tool")
    governed = wrapper.wrap_tool(tool)

    assert governed.name == "my_tool"
    assert governed.description == "Mock my_tool tool"
    assert governed._original is tool
