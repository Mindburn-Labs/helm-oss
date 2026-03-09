# Security Model

> **Canonical architecture**: see [ARCHITECTURE.md](ARCHITECTURE.md) for the
> system-level model (trust boundaries, VPL, Proof Condensation).

## Design Principle: Fail-Closed, Proof-First

HELM assumes every external input is adversarial. The kernel enforces a strict
policy enforcement point (PEP) at every execution boundary. If validation fails
at any stage, execution halts — there is no fallback path.

## Execution Pipeline

The canonical execution protocol is the Verified Planning Loop (VPL) — see
[ARCHITECTURE.md §5](ARCHITECTURE.md#5-runtime-execution--verified-planning-loop-vpl).

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

The TCB is explicitly bounded to 8 packages. For the full inventory and
expansion criteria, see [TCB_POLICY.md](TCB_POLICY.md).

| Package           | Responsibility                                                |
| ----------------- | ------------------------------------------------------------- |
| `contracts`       | Canonical data structures (Decision, Effect, Receipt, Intent) |
| `crypto`          | Ed25519 signing, verification, JCS canonicalization           |
| `guardian`        | Policy enforcement (PEP, PRG, compliance)                     |
| `executor`        | SafeExecutor — gated execution, idempotency                   |
| `proofgraph`      | Append-only DAG, AIGP anchors                                 |
| `trust`           | Event-sourced key registry, TUF, Rekor, SLSA                  |
| `runtime/sandbox` | WASI isolation: gas, time, memory caps                        |
| `receipts`        | Receipt policy enforcement (fail-closed)                      |

**TCB expansion requires**: deterministic behavior, no external I/O, no
reflection, >80% test coverage, maintainer review (see [TCB_POLICY.md](TCB_POLICY.md)).

## Sandbox Model (WASI)

Untrusted code executes in a WASI sandbox with:

- **No filesystem access** (deny-by-default)
- **No network access** (deny-by-default)
- **Gas metering** — hard budget per invocation
- **Wall-clock timeout** — configurable per tool
- **Memory cap** — WASM linear memory bounded
- **Deterministic traps** — budget exhaustion produces stable error codes

## Approval Ceremonies

High-risk operations require human approval via the approval ceremony,
which binds cryptographic hashes — see
[ARCHITECTURE.md §3](ARCHITECTURE.md#3-policy-precedence):

1. `policy_bundle_hash` — SHA-256 of active policy bundle set
2. `p0_ceiling_hash` — SHA-256 of active P0 ceiling set
3. `intent_hash` — SHA-256 of the proposed execution intent
4. `approver_signature` — Ed25519 signature from authorized approver

The ceremony supports timelock, quorum, rate limits, and emergency override.

## EvidencePack

Every session can be exported as a deterministic `.tar` containing:

- Retained receipts (signed) + Merkle inclusion proofs for condensed entries
- ProofGraph DAG state
- Trust registry snapshot
- Schema versions used
- Replay script

The archive is byte-identical for identical content (sorted paths, epoch mtime,
root uid/gid, fixed permissions). Low-risk receipts may be replaced by inclusion
proofs after checkpoint — see [ARCHITECTURE.md §5.2](ARCHITECTURE.md#52-proof-condensation).
