"""
HELM EffectBoundary Substrate Example — Python

Demonstrates implementing the EffectBoundary contract in Python
using the OpenAPI-defined REST surface.
"""

import json
import hashlib
import time
from dataclasses import dataclass, field, asdict
from enum import Enum
from typing import Optional
from http.server import HTTPServer, BaseHTTPRequestHandler


class Verdict(Enum):
    ALLOW = "ALLOW"
    DENY = "DENY"
    ESCALATE = "ESCALATE"


@dataclass
class EffectRequest:
    effect_type: str
    principal_id: str
    params: dict = field(default_factory=dict)
    context: dict = field(default_factory=dict)
    idempotency_key: Optional[str] = None


@dataclass
class Receipt:
    receipt_id: str
    verdict: str
    reason_code: str
    reason: str
    timestamp: str
    lamport: int
    principal_id: str

    @staticmethod
    def create(verdict: Verdict, reason_code: str, reason: str,
               principal_id: str, lamport: int) -> "Receipt":
        ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        content = f"{verdict.value}:{reason_code}:{ts}:{lamport}"
        receipt_id = f"urn:helm:receipt:{hashlib.sha256(content.encode()).hexdigest()[:16]}"
        return Receipt(
            receipt_id=receipt_id,
            verdict=verdict.value,
            reason_code=reason_code,
            reason=reason,
            timestamp=ts,
            lamport=lamport,
            principal_id=principal_id,
        )


class EffectBoundary:
    """Minimal EffectBoundary implementation for substrate demonstration."""

    def __init__(self):
        self._lamport = 0
        self._receipts: list[Receipt] = []

    def submit(self, req: EffectRequest) -> dict:
        """Submit an effect for governance evaluation."""
        self._lamport += 1

        # --- PDP evaluation (simplified) ---
        verdict, reason_code, reason = self._evaluate(req)

        receipt = Receipt.create(
            verdict=verdict,
            reason_code=reason_code,
            reason=reason,
            principal_id=req.principal_id,
            lamport=self._lamport,
        )
        self._receipts.append(receipt)

        return {
            "verdict": verdict.value,
            "receipt": asdict(receipt),
            "intent": {
                "effect_type": req.effect_type,
                "allowed": verdict == Verdict.ALLOW,
            },
        }

    def complete(self, receipt_id: str, result: dict) -> dict:
        """Record completion of a previously allowed effect."""
        self._lamport += 1
        completion = Receipt.create(
            verdict=Verdict.ALLOW,
            reason_code="EFFECT_COMPLETED",
            reason=f"Effect {receipt_id} completed",
            principal_id="system",
            lamport=self._lamport,
        )
        self._receipts.append(completion)
        return asdict(completion)

    def _evaluate(self, req: EffectRequest) -> tuple[Verdict, str, str]:
        """Simplified PDP: deny risky effects, allow everything else."""
        if req.effect_type == "data_export" and req.params.get("data_class") == "PII":
            return Verdict.DENY, "POLICY_VIOLATION", "PII export denied by policy"
        if req.effect_type == "financial_transfer":
            amount = req.params.get("amount_cents", 0)
            if amount > 1_000_000:
                return Verdict.ESCALATE, "TEMPORAL_INTERVENTION", "High value transfer requires approval"
        return Verdict.ALLOW, "POLICY_SATISFIED", "Effect allowed"


class SubstrateHandler(BaseHTTPRequestHandler):
    """HTTP handler implementing the HELM EffectBoundary REST surface."""

    boundary = EffectBoundary()

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))

        if self.path == "/v1/effects":
            req = EffectRequest(**body)
            result = self.boundary.submit(req)
            self._json_response(200, result)

        elif self.path.startswith("/v1/effects/") and self.path.endswith("/complete"):
            receipt_id = self.path.split("/")[3]
            result = self.boundary.complete(receipt_id, body)
            self._json_response(200, result)

        else:
            self._json_response(404, {"error": "Not found"})

    def _json_response(self, code: int, data: dict):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data, indent=2).encode())


if __name__ == "__main__":
    print("HELM EffectBoundary substrate running on :4001")
    HTTPServer(("", 4001), SubstrateHandler).serve_forever()
