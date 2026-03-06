---
title: Threat Model
---

# Threat Model

## Trust Boundaries

```
┌─────────────────────────────────────────────────────┐
│                    UNTRUSTED                        │
│  LLM Provider · User Prompts · Connector Outputs    │
└───────────────────────┬─────────────────────────────┘
                        │
                  ┌─────▼─────┐
                  │ HELM      │  ← PEP boundary (schema + hash)
                  │ Kernel    │  ← Guardian (policy engine)
                  │           │  ← SafeExecutor (signed receipts)
                  └─────┬─────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│                    TRUSTED                          │
│  Signed Receipt Store · ProofGraph DAG · Trust Reg  │
└─────────────────────────────────────────────────────┘
```

## Threat Categories

### T1: Unauthorized Tool Execution

**Attack:** Model generates a tool call not sanctioned by the current policy.

**Defense:** Guardian policy engine maintains an explicit allowlist. Undeclared tools are blocked before reaching the executor. Default-deny.

**Residual risk:** None — this is a hard block.

### T2: Argument Tampering

**Attack:** Malicious input crafts tool arguments that bypass validation or alter semantics.

**Defense:**
1. Schema validation against pinned JSON Schema (fail-closed)
2. JCS canonicalization (RFC 8785) eliminates encoding ambiguity
3. SHA-256 hash of canonical args (`ArgsHash`) bound into signed receipt

**Residual risk:** Schema must be correct. HELM enforces the schema, not its semantic correctness.

### T3: Output Spoofing

**Attack:** Malicious connector returns data that doesn't match the declared output schema.

**Defense:** Output validation against pinned schema. Contract drift produces `ERR_CONNECTOR_CONTRACT_DRIFT` and halts execution.

**Residual risk:** Connector could return semantically wrong but schema-valid data.

### T4: Resource Exhaustion (WASI)

**Attack:** Uploaded WASM module consumes unbounded CPU, memory, or time.

**Defense:**
- Gas metering: hard budget per invocation
- Wall-clock timeout: configurable per-tool
- Memory cap: WASM linear memory bounded
- Deterministic trap codes on budget exhaustion

**Residual risk:** None for compute resources. Side-channels at the host OS level are out of scope.

### T5: Receipt Forgery

**Attack:** Attacker creates fake receipts to claim executions that didn't happen.

**Defense:** Ed25519 signatures on canonical payloads. Verification requires the signer's public key.

**Residual risk:** Key compromise. Mitigated by Trust Registry key rotation.

### T6: Replay Attacks

**Attack:** Attacker replays a valid receipt to re-execute an effect.

**Defense:**
- Lamport clock monotonicity per session
- Causal `PrevHash` chain (each receipt signs over previous receipt's signature)
- Idempotency cache in executor

**Residual risk:** None within a single session. Cross-session replay mitigated by session scoping.

### T7: Approval Bypass

**Attack:** Model or operator bypasses human approval for high-risk operations.

**Defense:**
- Timelock: approval window must elapse before execution
- Deliberate confirmation: approver must produce a hash derived from the original intent
- Domain separation: approval keys are distinct from execution keys
- Challenge/response ceremony for disputes

**Residual risk:** Social engineering of the human approver is out of scope.

### T8: Trust Registry Manipulation

**Attack:** Attacker adds a rogue key or revokes a legitimate one.

**Defense:** Event-sourced trust registry. Every key lifecycle event (add/revoke/rotate) is a signed, immutable event with Lamport ordering. Registry state is replayable from genesis.

**Residual risk:** Compromise of the registry admin key. Mitigated by ceremony-based key management.

## Out of Scope

- Content safety / prompt injection within the text domain
- Vulnerabilities in upstream LLM providers
- Host OS / hardware side channels
- Network-level attacks (TLS is assumed)
- Social engineering of human approvers
