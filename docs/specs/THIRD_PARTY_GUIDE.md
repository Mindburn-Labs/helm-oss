# HELM Third-Party Implementation Guide

> How to build a HELM-conformant implementation from scratch.

## 1. Architecture Overview

A conformant HELM implementation requires these components:

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│  Your Runtime   │────▶│  Effect      │────▶│  Policy      │
│  (Agent/App)    │     │  Boundary    │     │  Decision    │
│                 │◀────│              │◀────│  Point       │
└─────────────────┘     └──────────────┘     └──────────────┘
                              │
                        ┌─────▼──────┐
                        │  Receipt   │
                        │  Store     │
                        └────────────┘
```

## 2. Step-by-Step

### Step 1: Implement EffectBoundary

Implement the `EffectBoundaryService` interface from either:

- **gRPC**: `protocols/proto/helm/kernel/v1/helm.proto`
- **REST**: `protocols/specs/effects/openapi.yaml`

Required operations:

1. `Submit(effect) → verdict + receipt + intent`
2. `Complete(result) → completion receipt`

### Step 2: Implement Policy Decision Point

Evaluate policy rules against submitted effects. Options:

- Use the HELM PDP library (Go)
- Implement `PolicyDecisionPointService` from the proto
- Write a custom PDP that returns `PDPResponse`

### Step 3: Generate Receipts

Every governance decision MUST produce a Receipt per
`protocols/specs/rfc/receipt-format-v1.md`:

- Content-addressed `receipt_id` (SHA-256)
- Monotonic `lamport` clock
- Ed25519 or ECDSA `signature`

### Step 4: Run Conformance Vectors

```bash
helm conform run \
  --vectors protocols/conformance/v1/test-vectors.json \
  --endpoint http://your-server:4001 \
  --level 4
```

### Step 5: Submit to Compatibility Registry

Open a PR adding your implementation to
`protocols/conformance/v1/compatibility-registry.json`.

## 3. Reference Materials

| Document           | Path                                            |
| ------------------ | ----------------------------------------------- |
| Receipt Format RFC | `protocols/specs/rfc/receipt-format-v1.md`      |
| Reason Codes RFC   | `protocols/specs/rfc/reason-codes-v1.md`        |
| Identifier URN RFC | `protocols/specs/rfc/identifier-urn-v1.md`      |
| Versioning RFC     | `protocols/specs/rfc/artifact-versioning-v1.md` |
| OpenAPI Spec       | `protocols/specs/effects/openapi.yaml`          |
| Proto IDL          | `protocols/proto/helm/kernel/v1/helm.proto`     |
| Conformance Guide  | `protocols/conformance/v1/CONFORMANCE_GUIDE.md` |

## 4. Certification

See [CONFORMANCE.md](../../docs/CONFORMANCE.md) for gate levels and profile verification.
