# Third-Party HELM Implementation Guide

> How to build a non-Go HELM implementation that is conformant.

## 1. What You Need to Implement

A conformant HELM implementation MUST:

| Component          | Required | Description                                   |
| ------------------ | -------- | --------------------------------------------- |
| EffectBoundary     | ✅       | Single enforcement point for governed effects |
| PDP                | ✅       | Policy decision point (pluggable)             |
| Receipt Generation | ✅       | Cryptographically signed decision receipts    |
| ProofGraph         | ✅       | Hash-chained decision log with Lamport clock  |
| Bundle Loader      | ✅       | Load and verify signed policy bundles         |
| Verdict Vocabulary | ✅       | ALLOW/DENY/ESCALATE (canonical types)         |
| Reason Codes       | ✅       | 17 core codes from registry                   |

A conformant implementation MAY additionally:

- Anchor to transparency logs (Rekor)
- Support Cedar/OPA as PDP backends
- Implement WASI sandboxing
- Provide framework-specific adapters

## 2. Canonical IDL

Start from the proto IDL:

```
protocols/proto/helm/kernel/v1/helm.proto
```

Generate your types from this. Do not manually transcribe types.

## 3. Implementation Checklist

### 3.1 Core Types

```
☐ Verdict enum: ALLOW, DENY, ESCALATE
☐ ReasonCode enum: all 17 core codes
☐ Receipt struct with all required fields
☐ DecisionRecord struct
☐ Effect struct
☐ AuthorizedExecutionIntent struct
```

### 3.2 EffectBoundary

```
☐ Submit(EffectRequest) → EffectResponse
☐ Complete(ExecutionResult) → CompletionReceipt
☐ Fail-closed on PDP unreachable (MUST return DENY + PDP_ERROR)
☐ Receipt generated for every decision (including denials)
```

### 3.3 Receipt Properties (Non-Negotiable)

```
☐ receipt_id is unique (UUID v4 or equivalent)
☐ verdict matches the decision
☐ timestamp is monotonically increasing
☐ signature is valid Ed25519 or ECDSA
☐ payload_hash is SHA-256 of JCS-canonical payload
☐ reason_code is from the registered registry
☐ lamport clock is strictly increasing in ProofGraph
```

### 3.4 ProofGraph

```
☐ Nodes linked by parent hash
☐ Lamport clock monotonically increasing
☐ Append-only (no mutation of existing nodes)
☐ Chain verification: removing any node breaks the chain
```

### 3.5 Policy Bundles

```
☐ Load YAML manifests per policy-bundle-v1.md
☐ Verify Ed25519 signature before loading
☐ Reject bundles with unknown apiVersion
☐ Fail closed on signature verification failure
☐ Record loaded bundles in EvidencePack
```

## 4. Conformance Testing

Run the test vectors against your implementation:

```bash
# Using the HELM conformance runner
helm conform run \
  --vectors protocols/conformance/v1/test-vectors.json \
  --endpoint http://your-implementation:4001 \
  --level 4
```

Or implement each vector programmatically — see `test-vectors.json`.

## 5. Certification Process

| Level   | What you pass                          | Badge                   |
| ------- | -------------------------------------- | ----------------------- |
| Level 1 | Verdict vectors (ALLOW/DENY/ESCALATE)  | Conformant              |
| Level 2 | Receipt invariants                     | Conformant + Receipts   |
| Level 3 | ProofGraph chain verification          | Conformant + Provenance |
| Level 4 | All above + fail-closed + reason codes | Fully Conformant        |

### Certification Steps

1. Fork the conformance vectors
2. Implement your EffectBoundary
3. Run `helm conform run --level 4`
4. Submit results via GitHub issue
5. HELM team reviews and grants certification

## 6. Reference Implementations

| Language   | Location             | Status              |
| ---------- | -------------------- | ------------------- |
| Go         | `core/pkg/guardian/` | Canonical reference |
| Python     | `sdk/python/`        | SDK reference       |
| TypeScript | `sdk/ts/`            | SDK reference       |
| Java       | `sdk/java/`          | SDK reference       |
| Rust       | `sdk/rust/`          | SDK reference       |

## 7. RFC Evolution Process

1. **Proposal**: Open GitHub issue with `rfc:` prefix
2. **Discussion**: 14-day comment period
3. **Draft**: Submit PR with RFC in `protocols/specs/rfc/`
4. **Review**: Minimum 2 maintainer approvals
5. **Final**: Merge with `status: final`
6. **Versioning**: SemVer — minor for additions, major for breaking

## 8. Bridge Documents (2027+)

For upcoming AI agent platforms:

| Platform                 | Bridge Status | Notes                  |
| ------------------------ | ------------- | ---------------------- |
| Anthropic Antigravity    | Planned       | MCP-native integration |
| Amazon Bedrock AgentCore | Reference     | AWS bridge pattern     |
| Google Kiro              | Planned       | ADK-based integration  |
| Cohere Toolkit           | Planned       | Middleware integration |
