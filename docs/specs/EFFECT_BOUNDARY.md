---
title: EFFECT_BOUNDARY
---

# EffectBoundary Specification

> **Status**: normative v1  
> **Canonical IDL**: `protocols/proto/helm/kernel/v1/helm.proto`

## 1. Overview

The EffectBoundary is the **single enforcement point** through which all governed
effects pass. It is the Policy Enforcement Point (PEP) in HELM's architecture.

Every side effect — tool calls, file operations, API requests, deployments,
data mutations — MUST transit the EffectBoundary before execution.

## 2. Wire Protocol

### 2.1 gRPC Service

```protobuf
service EffectBoundaryService {
  rpc Submit(EffectRequest)     returns (EffectResponse);
  rpc Complete(ExecutionResult)  returns (CompletionReceipt);
}
```

### 2.2 HTTP/JSON Equivalent

For clients that cannot use gRPC:

```
POST /v1/effects/submit
POST /v1/effects/complete
```

Request/response bodies are JSON representations of the protobuf messages.

## 3. Submit Flow

```
Client → EffectRequest → EffectBoundary → PDP → Verdict
                                              ↓
                                    ALLOW → AuthorizedExecutionIntent
                                    DENY  → Receipt (with reason_code)
                                    ESCALATE → Receipt (pending approval)
```

### 3.1 EffectRequest

| Field       | Type   | Required | Description                                    |
| ----------- | ------ | -------- | ---------------------------------------------- |
| `effect`    | Effect | yes      | The effect to govern                           |
| `principal` | string | yes      | Identity of the requesting agent               |
| `context`   | map    | no       | Additional context (jurisdiction, environment) |

### 3.2 EffectResponse

| Field         | Type                      | Description                                 |
| ------------- | ------------------------- | ------------------------------------------- |
| `verdict`     | Verdict                   | ALLOW, DENY, or ESCALATE                    |
| `reason_code` | ReasonCode                | Machine-readable reason (for DENY/ESCALATE) |
| `reason`      | string                    | Human-readable explanation                  |
| `receipt`     | Receipt                   | Governance receipt for this decision        |
| `intent`      | AuthorizedExecutionIntent | Execution authorization (ALLOW only)        |

## 4. Complete Flow

After executing an authorized effect, clients MUST report completion:

```
Client → ExecutionResult → EffectBoundary → CompletionReceipt
```

The completion receipt:

- Closes the governance loop
- Records the execution result in the ProofGraph
- Enables auditability of the full lifecycle

## 5. Fail-Closed Guarantee

If the EffectBoundary is unreachable or returns an error:

- The client MUST NOT execute the effect
- The client SHOULD log the failure
- This is the **kernel invariant**: no ungovern execution

## 6. Portable Embedding

Frameworks embed the EffectBoundary by implementing a function equivalent to:

```
helm.Effect(effect_type, params, principal) → (verdict, receipt, intent?)
```

This function MUST:

1. Construct an EffectRequest
2. Submit to the EffectBoundary (local or remote)
3. Return the verdict and receipt
4. If ALLOW, return the authorized intent for execution

See: `docs/specs/PORTABLE_EFFECT_MODEL.md`

## 7. SDK Integration Points

| SDK        | Function                  | Transport  |
| ---------- | ------------------------- | ---------- |
| Go         | `guardian.SignDecision()` | In-process |
| Python     | `helm.submit_effect()`    | gRPC/HTTP  |
| TypeScript | `helm.submitEffect()`     | gRPC/HTTP  |
| Java       | `Helm.submitEffect()`     | gRPC/HTTP  |
| Rust       | `helm::submit_effect()`   | gRPC/HTTP  |
