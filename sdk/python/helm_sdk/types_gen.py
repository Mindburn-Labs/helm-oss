# AUTO-GENERATED from api/openapi/helm.openapi.yaml — DO NOT EDIT
# Regenerate: bash scripts/sdk/gen.sh

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


# ── Reason Codes ──────────────────────────────────────
REASON_CODES = [
    "ALLOW",
    "DENY_TOOL_NOT_FOUND",
    "DENY_SCHEMA_MISMATCH",
    "DENY_OUTPUT_DRIFT",
    "DENY_BUDGET_EXCEEDED",
    "DENY_APPROVAL_REQUIRED",
    "DENY_APPROVAL_TIMEOUT",
    "DENY_SANDBOX_TRAP",
    "DENY_GAS_EXHAUSTION",
    "DENY_TIME_LIMIT",
    "DENY_MEMORY_LIMIT",
    "DENY_POLICY_VIOLATION",
    "DENY_TRUST_KEY_REVOKED",
    "DENY_IDEMPOTENCY_DUPLICATE",
    "ERROR_INTERNAL",
]


@dataclass
class HelmErrorDetail:
    message: str = ""
    type: str = ""
    code: str = ""
    reason_code: str = ""
    details: Optional[Dict[str, Any]] = None


@dataclass
class ChatMessage:
    role: str = ""
    content: str = ""
    tool_call_id: Optional[str] = None


@dataclass
class ToolFunction:
    name: str = ""
    description: str = ""
    parameters: Optional[Dict[str, Any]] = None


@dataclass
class Tool:
    type: str = "function"
    function: Optional[ToolFunction] = None


@dataclass
class ChatCompletionRequest:
    model: str = ""
    messages: List[ChatMessage] = field(default_factory=list)
    tools: Optional[List[Tool]] = None
    temperature: Optional[float] = None
    max_tokens: Optional[int] = None
    stream: bool = False


@dataclass
class ToolCall:
    id: str = ""
    type: str = ""
    function: Optional[Dict[str, str]] = None


@dataclass
class ChoiceMessage:
    role: str = ""
    content: Optional[str] = None
    tool_calls: Optional[List[ToolCall]] = None


@dataclass
class Choice:
    index: int = 0
    message: Optional[ChoiceMessage] = None
    finish_reason: str = ""


@dataclass
class Usage:
    prompt_tokens: int = 0
    completion_tokens: int = 0
    total_tokens: int = 0


@dataclass
class ChatCompletionResponse:
    id: str = ""
    object: str = "chat.completion"
    created: int = 0
    model: str = ""
    choices: List[Choice] = field(default_factory=list)
    usage: Optional[Usage] = None


@dataclass
class ApprovalRequest:
    intent_hash: str = ""
    signature_b64: str = ""
    public_key_b64: str = ""
    challenge_response: Optional[str] = None


@dataclass
class Receipt:
    receipt_id: str = ""
    decision_id: str = ""
    effect_id: str = ""
    status: str = ""
    reason_code: str = ""
    output_hash: str = ""
    blob_hash: str = ""
    prev_hash: str = ""
    lamport_clock: int = 0
    signature: str = ""
    timestamp: str = ""
    principal: str = ""


@dataclass
class Session:
    session_id: str = ""
    created_at: str = ""
    receipt_count: int = 0
    last_lamport_clock: int = 0


@dataclass
class ExportRequest:
    session_id: Optional[str] = None
    format: str = "tar.gz"


@dataclass
class VerificationResult:
    verdict: str = ""
    checks: Optional[Dict[str, str]] = None
    errors: List[str] = field(default_factory=list)


@dataclass
class ConformanceRequest:
    level: str = "L1"
    profile: str = "full"


@dataclass
class ConformanceResult:
    report_id: str = ""
    level: str = ""
    verdict: str = ""
    gates: int = 0
    failed: int = 0
    details: Optional[Dict[str, str]] = None


@dataclass
class VersionInfo:
    version: str = ""
    commit: str = ""
    build_time: str = ""
    go_version: str = ""
