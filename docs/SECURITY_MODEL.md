# Security Model

## Design Principle: Fail-Closed, Proof-First

HELM assumes every external input is adversarial. The kernel enforces a strict policy enforcement point (PEP) at every execution boundary. If validation fails at any stage, execution halts — there is no fallback path.

## Execution Pipeline

```
Request → Guardian (policy) → PEP (schema + hash) → SafeExecutor → Driver
                                                         │
                                              ┌──────────▼──────────┐
                                              │  Output Validation  │
                                              │  (pinned schema)    │
                                              └──────────┬──────────┘
                                                         │
                                              ┌──────────▼──────────┐
                                              │  Sign Receipt       │
                                              │  (Ed25519)          │
                                              └──────────┬──────────┘
                                                         │
                                              ┌──────────▼──────────┐
                                              │  ProofGraph DAG     │
                                              │  (append-only)      │
                                              └─────────────────────┘
```

## Cryptographic Chain

Every execution produces the following cryptographic artifacts:

1. **ArgsHash** — SHA-256 of JCS-canonicalized tool arguments
2. **OutputHash** — SHA-256 of validated connector output
3. **Receipt** — signed record binding: `ReceiptID:DecisionID:EffectID:Status:OutputHash:PrevHash:LamportClock:ArgsHash`
4. **PrevHash** — signature of the previous receipt (causal link)

This chain is append-only and verifiable offline.

## Trusted Computing Base (TCB)

The TCB is explicitly bounded to 8 packages:

| Package           | Responsibility                                      |
| ----------------- | --------------------------------------------------- |
| `crypto`          | Ed25519 signing, verification, canonicalization     |
| `executor`        | SafeExecutor — the single execution boundary        |
| `guardian`        | Policy enforcement (allowlist, deny rules)          |
| `manifest`        | Schema validation (input + output)                  |
| `proofgraph`      | DAG construction and verification                   |
| `trust`           | Event-sourced key registry                          |
| `runtime/sandbox` | WASI isolation                                      |
| `contracts`       | Data structures (Decision, Effect, Receipt, Intent) |

**TCB expansion requires**: justification, new CI gates, and maintainer review (see `docs/TCB_POLICY.md`).

## Sandbox Model (WASI)

Untrusted code executes in a WASI sandbox with:

- **No filesystem access** (deny-by-default)
- **No network access** (deny-by-default)
- **Gas metering** — hard budget per invocation
- **Wall-clock timeout** — configurable per tool
- **Memory cap** — WASM linear memory bounded
- **Deterministic traps** — budget exhaustion produces stable error codes

## Approval Ceremonies

High-risk operations require human approval with:

1. **Timelock** — minimum deliberation window
2. **Deliberate confirmation** — approver produces `SHA-256(intent_id + "CONFIRM")`
3. **Domain separation** — approval keys ≠ execution keys
4. **Challenge/response** — for disputed executions

## EvidencePack

Every session can be exported as a deterministic `.tar` containing:

- All receipts (signed)
- ProofGraph DAG state
- Trust registry snapshot
- Schema versions used
- Replay script

The archive is byte-identical for identical content (sorted paths, epoch mtime, root uid/gid, fixed permissions).
