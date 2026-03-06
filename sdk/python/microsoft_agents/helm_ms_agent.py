"""
HELM Microsoft Agent Framework Adapter (Python)

Intercepts tool execution boundary in MS Agent Framework RC
and routes through HELM governance plane.

Usage:
    from helm_ms_agent import HelmAgentToolWrapper

    wrapper = HelmAgentToolWrapper(helm_url="http://localhost:8080")
    result = wrapper.execute("search", {"query": "test"})
"""

import hashlib
import json
import time
import os
import threading
from dataclasses import dataclass, field
from typing import Any, Callable, Optional
import urllib.request
import urllib.error


@dataclass
class HelmReceipt:
    """Governance receipt."""
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
class GovernedResult:
    """Result of governed tool execution."""
    allowed: bool
    receipt: HelmReceipt
    output: Any = None
    error: Optional[str] = None


class HelmAgentToolWrapper:
    """
    Wraps MS Agent Framework tool execution with HELM governance.

    Intercepts the tool execution boundary and routes through HELM PEP.
    Compatible with Microsoft Agent Framework RC (successor to Semantic Kernel / AutoGen).

    Args:
        helm_url: Base URL of HELM server
        fail_closed: Deny on HELM unreachable (default: True)
        metadata: Global metadata for all receipts
    """

    def __init__(
        self,
        helm_url: str = "http://localhost:8080",
        fail_closed: bool = True,
        metadata: Optional[dict] = None,
    ):
        self.helm_url = helm_url.rstrip("/")
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
        principal: str = "ms-agent",
        **extra_metadata,
    ) -> GovernedResult:
        """Execute a tool through HELM governance."""
        with self._lock:
            self._lamport += 1
            lamport = self._lamport

        args_canonical = json.dumps(arguments, sort_keys=True, separators=(",", ":"))
        args_hash = "sha256:" + hashlib.sha256(args_canonical.encode()).hexdigest()

        verdict = "ALLOW"
        reason_code = "POLICY_PASS"

        try:
            response = self._call_helm("/v1/tools/evaluate", {
                "tool_name": tool_name,
                "arguments": arguments,
                "principal": principal,
                "args_hash": args_hash,
            })
            verdict = response.get("verdict", "ALLOW")
            reason_code = response.get("reason_code", "POLICY_PASS")
        except (urllib.error.URLError, ConnectionError):
            if self.fail_closed:
                verdict = "DENY"
                reason_code = "HELM_UNREACHABLE"

        with self._lock:
            prev_hash = self._prev_hash
            preimage = f"{tool_name}|{args_hash}|{verdict}|{reason_code}|{lamport}|{prev_hash}"
            receipt_hash = hashlib.sha256(preimage.encode()).hexdigest()
            self._prev_hash = receipt_hash

        receipt = HelmReceipt(
            receipt_id=f"rcpt-msaf-{receipt_hash[:8]}-{lamport}",
            timestamp=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            tool_name=tool_name,
            args_hash=args_hash,
            verdict=verdict,
            reason_code=reason_code,
            lamport_clock=lamport,
            prev_hash=prev_hash,
            hash=receipt_hash,
            metadata={**self.metadata, **extra_metadata, "principal": principal, "framework": "ms-agent-framework"},
        )
        with self._lock:
            self._receipts.append(receipt)

        if verdict != "ALLOW":
            return GovernedResult(allowed=False, receipt=receipt, error=reason_code)

        return GovernedResult(allowed=True, receipt=receipt)

    def wrap_tool(self, tool_fn: Callable) -> Callable:
        """Decorator to wrap a tool function with governance."""
        def wrapper(*args, **kwargs):
            tool_name = getattr(tool_fn, "__name__", str(tool_fn))
            result = self.execute(tool_name, kwargs or {"args": args})
            if not result.allowed:
                raise RuntimeError(f"HELM denied: {result.error}")
            return tool_fn(*args, **kwargs)
        wrapper.__name__ = getattr(tool_fn, "__name__", "governed_tool")
        return wrapper

    @property
    def receipts(self) -> list[HelmReceipt]:
        return list(self._receipts)

    def export_evidence_pack(self, path: str) -> str:
        """Export receipts as deterministic .tar EvidencePack."""
        import tarfile
        import io as _io

        with tarfile.open(path, "w") as tar:
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
                info.mtime = 0
                info.uid = 0
                info.gid = 0
                tar.addfile(info, _io.BytesIO(data))

            manifest = json.dumps({
                "session_id": f"ms-agent-{int(time.time())}",
                "receipt_count": len(self._receipts),
                "final_hash": self._prev_hash,
                "lamport": self._lamport,
                "framework": "microsoft-agent-framework",
            }, indent=2).encode()
            info = tarfile.TarInfo(name="manifest.json")
            info.size = len(manifest)
            info.mtime = 0
            info.uid = 0
            info.gid = 0
            tar.addfile(info, _io.BytesIO(manifest))

        with open(path, "rb") as f:
            return hashlib.sha256(f.read()).hexdigest()

    def _call_helm(self, endpoint: str, payload: dict) -> dict:
        url = f"{self.helm_url}{endpoint}"
        data = json.dumps(payload).encode()
        req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
