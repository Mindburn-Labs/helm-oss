---
title: TCB_POLICY
---

# HELM TCB Policy

> **Canonical architecture**: see [ARCHITECTURE.md §2](ARCHITECTURE.md#2-trust-boundaries)
> for the normative trust boundary model.

## Trusted Computing Base Principles

1. **Minimal Surface**: Only code that directly handles tool execution authorization,
   cryptographic signing, and policy enforcement lives in the TCB.

2. **Fail-Closed**: All PEP (Policy Enforcement Point) boundaries default to deny.
   Unknown tools, unvalidated args, drifted outputs are all blocked.

3. **Deterministic**: All canonicalization uses RFC 8785 JCS. All hashes are SHA-256.
   All signing uses Ed25519. No randomness in the decision path.

4. **Auditable**: Every action produces a ProofGraph node with Lamport-clock ordering.
   Evidence packs can be exported and independently verified.

## TCB Package Inventory

The TCB is bounded to the following 8 packages. CI enforces forbidden-import
gates — no code outside the TCB may be imported into a TCB package.

| Package           | Responsibility                                                             |
| :---------------- | :------------------------------------------------------------------------- |
| `contracts`       | Canonical data structures: Decision, Effect, Receipt, Intent, Verdict      |
| `crypto`          | Ed25519 signing, JCS canonicalization, HSM bridge, audit log               |
| `guardian`        | Policy Enforcement Point (PEP), temporal constraints, compliance pre-check |
| `executor`        | SafeExecutor — gated execution, idempotency, output drift detection        |
| `proofgraph`      | Append-only DAG, AIGP anchors                                              |
| `trust`           | Event-sourced key registry, TUF, Rekor, SLSA verification                  |
| `runtime/sandbox` | WASI isolation: gas, time, memory caps, deterministic traps                |
| `receipts`        | Receipt policy enforcement (fail-closed)                                   |

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
5. No reflection
6. Reviewed via `helm verify pack` evidence export
7. Maintainer review with explicit TCB-expansion justification
