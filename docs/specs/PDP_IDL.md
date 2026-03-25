---
title: PDP_IDL
---

# PDP IDL Specification

> **Status**: normative v1  
> **Canonical IDL**: `protocols/proto/helm/kernel/v1/helm.proto`

## 1. Overview

The Policy Decision Point (PDP) is a pluggable evaluator that determines whether
an effect should be allowed, denied, or escalated. HELM supports both in-process
and external PDP implementations.

## 2. Wire Protocol

### 2.1 gRPC Service

```protobuf
service PolicyDecisionPointService {
  rpc Evaluate(PDPRequest) returns (PDPResponse);
}
```

### 2.2 HTTP/JSON Equivalent

```
POST /v1/pdp/evaluate
```

## 3. PDPRequest

| Field     | Type              | Required | Description            |
| --------- | ----------------- | -------- | ---------------------- |
| `effect`  | Effect            | yes      | Effect being evaluated |
| `subject` | SubjectDescriptor | yes      | Who is requesting      |
| `context` | ContextDescriptor | no       | Evaluation context     |

### 3.1 SubjectDescriptor

| Field       | Type     | Description            |
| ----------- | -------- | ---------------------- |
| `principal` | string   | Agent or user identity |
| `tenant`    | string   | Tenant identifier      |
| `roles`     | string[] | Active roles           |

### 3.2 ContextDescriptor

| Field               | Type      | Description                    |
| ------------------- | --------- | ------------------------------ |
| `jurisdiction`      | string    | Active jurisdiction (ISO 3166) |
| `environment`       | string    | prod / staging / dev           |
| `time_window_start` | Timestamp | Evaluation window start        |
| `time_window_end`   | Timestamp | Evaluation window end          |

## 4. PDPResponse

| Field           | Type         | Description                              |
| --------------- | ------------ | ---------------------------------------- |
| `allow`         | bool         | Whether the effect is permitted          |
| `reason_code`   | ReasonCode   | Reason for deny (if !allow)              |
| `policy_ref`    | string       | Reference to the policy that applies     |
| `decision_hash` | string       | SHA-256 of the decision for auditability |
| `obligations`   | Obligation[] | Post-execution obligations (if allow)    |

## 5. PDP Implementations

### 5.1 In-Process (Go)

The default PDP uses the Policy Requirement Graph (PRG) with CEL expressions.
No network call required.

### 5.2 External (gRPC/HTTP)

```yaml
pdp:
  type: external
  endpoint: grpc://pdp.example.com:443
  timeout: 5s
  tls:
    cert: /path/to/cert.pem
```

## 6. Fail-Closed Behavior

If the PDP is unreachable or returns an error:

- The EffectBoundary MUST deny the effect
- Reason code: `PDP_ERROR`
- This is non-negotiable

## 7. Conformance

A PDP implementation is conformant if:

1. It accepts `PDPRequest` and returns `PDPResponse`
2. It supports the `effect` and `subject` fields at minimum
3. It populates `reason_code` on deny
4. It returns a deterministic `decision_hash`
5. It fails closed on internal error
