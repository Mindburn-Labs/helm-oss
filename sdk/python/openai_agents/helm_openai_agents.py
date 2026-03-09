"""
HELM OpenAI Agents SDK Adapter (Python)

Routes tool execution through the HELM governance plane.
Supports allow/deny decisions, receipt collection, metadata preservation,
and EvidencePack generation.

Usage:
    from helm_openai_agents import HelmToolExecutor

    executor = HelmToolExecutor(helm_url="http://localhost:8080")
    result = executor.execute("search_web", {"query": "test"})
    print(result.receipt_id)
    executor.export_evidence_pack("evidence.tar")
"""

import hashlib
import json
import time
import os
import threading
from dataclasses import dataclass, field
from typing import Any, Optional
import urllib.request
import urllib.error


@dataclass
class HelmReceipt:
    """A governance receipt from HELM."""
    receipt_id: str
    timestamp: str
    tool_name: str
    args_hash: str
    verdict: str
    reason_code: str
    lamport_clock: int
    prev_hash: str
    hash: str
    metadata: dict = field(default_factory=dict)


@dataclass
class ExecutionResult:
    """Result of a governed tool execution."""
    allowed: bool
    receipt: HelmReceipt
    output: Any = None
    error: Optional[str] = None


class HelmGovernanceError(Exception):
    """Raised when HELM denies a tool execution."""
    def __init__(self, reason_code: str, receipt: HelmReceipt):
        self.reason_code = reason_code
        self.receipt = receipt
        super().__init__(f"HELM denied execution: {reason_code}")


class HelmToolExecutor:
    """
    Wraps tool execution with HELM governance.

    Routes every tool call through the HELM PEP boundary.
    Collects receipts and can export EvidencePacks.

    Args:
        helm_url: Base URL of HELM server (default: http://localhost:8080)
        api_key: Optional API key for HELM authentication
        fail_closed: If True (default), raise on governance errors
        metadata: Additional metadata to include in every receipt
    """

    def __init__(
        self,
        helm_url: str = "http://localhost:8080",
        api_key: str = "",
        fail_closed: bool = True,
        metadata: Optional[dict] = None,
    ):
        self.helm_url = helm_url.rstrip("/")
        self.api_key = api_key or os.environ.get("HELM_API_KEY", "")
        self.fail_closed = fail_closed
        self.metadata = metadata or {}
        self._receipts: list[HelmReceipt] = []
        self._prev_hash = "GENESIS"
        self._lamport = 0
        self._lock = threading.Lock()

    def execute(
        self,
        tool_name: str,
        arguments: dict,
        principal: str = "openai-agent",
        **extra_metadata,
    ) -> ExecutionResult:
        """
        Execute a tool through HELM governance.

        Args:
            tool_name: Name of the tool to execute
            arguments: Tool arguments dict
            principal: Identity of the calling agent
            **extra_metadata: Additional metadata for the receipt

        Returns:
            ExecutionResult with receipt and output

        Raises:
            HelmGovernanceError: If execution is denied and fail_closed=True
        """
        with self._lock:
            self._lamport += 1
            lamport = self._lamport

        # Canonicalize arguments (sorted keys, no whitespace)
        args_canonical = json.dumps(arguments, sort_keys=True, separators=(",", ":"))
        args_hash = "sha256:" + hashlib.sha256(args_canonical.encode()).hexdigest()

        # Build governance request
        gov_request = {
            "tool_name": tool_name,
            "arguments": arguments,
            "principal": principal,
            "args_hash": args_hash,
            "metadata": {**self.metadata, **extra_metadata},
        }

        # Evaluate governance
        verdict = "ALLOW"
        reason_code = "POLICY_PASS"
        output = None
        error = None

        try:
            response = self._call_helm("/v1/tools/evaluate", gov_request)
            verdict = response.get("verdict", "ALLOW")
            reason_code = response.get("reason_code", "POLICY_PASS")
        except (urllib.error.URLError, ConnectionError):
            if self.fail_closed:
                verdict = "DENY"
                reason_code = "HELM_UNREACHABLE"
            # If not fail_closed, allow through

        # Build receipt
        with self._lock:
            prev_hash = self._prev_hash
            preimage = f"{tool_name}|{args_hash}|{verdict}|{reason_code}|{lamport}|{prev_hash}"
            receipt_hash = hashlib.sha256(preimage.encode()).hexdigest()
            self._prev_hash = receipt_hash

        receipt = HelmReceipt(
            receipt_id=f"rcpt-oai-{receipt_hash[:8]}-{lamport}",
            timestamp=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            tool_name=tool_name,
            args_hash=args_hash,
            verdict=verdict,
            reason_code=reason_code,
            lamport_clock=lamport,
            prev_hash=prev_hash,
            hash=receipt_hash,
            metadata={**self.metadata, **extra_metadata, "principal": principal},
        )
        with self._lock:
            self._receipts.append(receipt)

        if verdict != "ALLOW":
            if self.fail_closed:
                raise HelmGovernanceError(reason_code, receipt)
            return ExecutionResult(allowed=False, receipt=receipt, error=reason_code)

        return ExecutionResult(allowed=True, receipt=receipt, output=output)

    @property
    def receipts(self) -> list[HelmReceipt]:
        """All collected receipts."""
        return list(self._receipts)

    def export_evidence_pack(self, path: str) -> str:
        """
        Export receipts as an EvidencePack (.tar).

        Args:
            path: Output file path

        Returns:
            SHA-256 hash of the exported pack
        """
        import tarfile
        import io as _io

        with tarfile.open(path, "w") as tar:
            # Sort receipts by lamport clock for determinism
            for i, receipt in enumerate(sorted(self._receipts, key=lambda r: r.lamport_clock)):
                data = json.dumps({
                    "receipt_id": receipt.receipt_id,
                    "timestamp": receipt.timestamp,
                    "tool_name": receipt.tool_name,
                    "args_hash": receipt.args_hash,
                    "verdict": receipt.verdict,
                    "reason_code": receipt.reason_code,
                    "lamport_clock": receipt.lamport_clock,
                    "prev_hash": receipt.prev_hash,
                    "hash": receipt.hash,
                    "metadata": receipt.metadata,
                }, indent=2).encode()

                info = tarfile.TarInfo(name=f"{i:03d}_{receipt.receipt_id}.json")
                info.size = len(data)
                info.mtime = 0  # epoch for determinism
                info.uid = 0
                info.gid = 0
                tar.addfile(info, _io.BytesIO(data))

            # Manifest
            manifest = json.dumps({
                "session_id": f"oai-agents-{int(time.time())}",
                "receipt_count": len(self._receipts),
                "final_hash": self._prev_hash,
                "lamport": self._lamport,
            }, indent=2).encode()
            info = tarfile.TarInfo(name="manifest.json")
            info.size = len(manifest)
            info.mtime = 0
            info.uid = 0
            info.gid = 0
            tar.addfile(info, _io.BytesIO(manifest))

        # Return pack hash
        with open(path, "rb") as f:
            return hashlib.sha256(f.read()).hexdigest()

    def _call_helm(self, endpoint: str, payload: dict) -> dict:
        """Call a HELM API endpoint."""
        url = f"{self.helm_url}{endpoint}"
        data = json.dumps(payload).encode()
        req = urllib.request.Request(
            url,
            data=data,
            headers={
                "Content-Type": "application/json",
                **({"Authorization": f"Bearer {self.api_key}"} if self.api_key else {}),
            },
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())


def wrap_openai_tool(executor: HelmToolExecutor, tool_fn):
    """
    Decorator to wrap an OpenAI Agents SDK tool function with HELM governance.

    Usage:
        @wrap_openai_tool(executor, search_tool)
        def governed_search(query: str) -> str:
            return search_tool(query)
    """
    def wrapper(*args, **kwargs):
        tool_name = getattr(tool_fn, "__name__", str(tool_fn))
        result = executor.execute(tool_name, kwargs or {"args": args})
        if not result.allowed:
            raise HelmGovernanceError(result.receipt.reason_code, result.receipt)
        return tool_fn(*args, **kwargs)
    wrapper.__name__ = getattr(tool_fn, "__name__", "governed_tool")
    wrapper.__wrapped__ = tool_fn
    return wrapper
