---
title: HELM TCB Policy
---

# HELM TCB Policy

## Trust Computing Base Principles

1. **Minimal Surface**: Only code that directly handles tool execution authorization,
   cryptographic signing, and policy enforcement lives in the TCB.

2. **Fail-Closed**: All PEP (Policy Enforcement Point) boundaries default to deny.
   Unknown tools, unvalidated args, drifted outputs are all blocked.

3. **Deterministic**: All canonicalization uses RFC 8785 JCS. All hashes are SHA-256.
   All signing uses Ed25519. No randomness in the decision path.

4. **Auditable**: Every action produces a ProofGraph node with Lamport-clock ordering.
   Evidence packs can be exported and independently verified.

## TCB Boundaries

```
                    ┌──────────────────────┐
   Untrusted        │     TCB Boundary     │      Trusted
                    │                      │
   LLM Output ────► │  PEP: Tool Args      │
                    │  Validation +        │──── Guardian PRG
   Connector   ◄─── │  Output Drift        │
   Response         │  Detection           │──── SafeExecutor
                    │                      │
   Human Input ────►│  Ceremony            │──── Crypto Signer
                    │  Validation          │
                    └──────────────────────┘
```

## Adding Code to TCB

Any code addition to TCB packages requires:
1. Deterministic behavior (no goroutines, no time-dependent logic in decision path)
2. Test coverage > 80%
3. No external network calls
4. No dynamic loading (no plugins, no reflection-based dispatch)
5. Reviewed via `helm verify pack` evidence export
