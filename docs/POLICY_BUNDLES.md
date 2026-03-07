# External Policy Bundle Loading (GOV-002)

## Overview

HELM's policy engine currently uses CEL rules defined inline in Go code. This guide describes the planned external policy bundle format for loading rules from files at runtime.

## Bundle Format

Policy bundles are YAML files with CEL expressions:

```yaml
# policy-bundle.yaml
apiVersion: helm.mindburn.com/v1
kind: PolicyBundle
metadata:
  name: production-policy
  version: "1.2.0"
  hash: "sha256:auto-computed-on-load"

rules:
  # Guardian-level rules (evaluated on every tool call)
  - id: require-approval-for-write
    action: "write.*"
    expression: |
      intent.risk_score < 0.7 || 
      has(artifacts, "type", "HUMAN_APPROVAL")
    verdict: BLOCK
    reason: "Write operations with risk > 0.7 require human approval"

  - id: budget-gate
    action: "*"
    expression: |
      state.budget_remaining > 10.0
    verdict: BLOCK
    reason: "Insufficient error budget"

  # PEP-level rules (post-execution validation)
  - id: output-size-limit
    action: "*"
    expression: |
      size(effect.output) < 1048576
    verdict: BLOCK
    reason: "Output exceeds 1MB limit"

  # Temporal rules
  - id: rate-limit
    action: "*"
    expression: |
      state.calls_per_minute < 100
    verdict: THROTTLE
    reason: "Rate limit exceeded"
```

## Loading

```go
import "github.com/Mindburn-Labs/helm-oss/core/pkg/guardian"

// Load at startup
bundle, err := guardian.LoadPolicyBundle("policy-bundle.yaml")
if err != nil {
    log.Fatal("failed to load policy bundle:", err)
}

// Content-addressed version (GOV-001)
log.Printf("Policy version: %s", bundle.ContentHash())

// Apply to Guardian
g := guardian.NewGuardian(signer, prg, registry)
g.ApplyPolicyBundle(bundle)
```

## Validation

Bundles are validated on load:

1. **Schema check** — YAML structure matches expected format
2. **CEL compilation** — All expressions must compile without errors
3. **Content hash** — SHA-256 of canonical bundle content (for GOV-001)
4. **Signature verification** — Optional Cosign/Ed25519 signature on bundle file

## Roadmap

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | YAML bundle format definition | ✅ Defined |
| 2 | Bundle loader with CEL compilation | Planned |
| 3 | Hot-reload on file change (fsnotify) | Planned |
| 4 | Bundle signing and verification | Planned |
| 5 | Remote bundle fetch (HTTP/Git) | Planned |
