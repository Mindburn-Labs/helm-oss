---
title: POLICY_BUNDLES
---

# External Policy Bundle Loading (GOV-002)

## Overview

HELM's policy engine supports external policy bundles — signed YAML files
containing governance rules that can be loaded, verified, and composed at
runtime.

## Bundle Format

```yaml
# policy-bundle.yaml
apiVersion: helm.mindburn.run/v1
kind: PolicyBundle
metadata:
  name: production-policy
  version: "1.2.0"

rules:
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

  - id: rate-limit
    action: "*"
    expression: |
      state.calls_per_minute < 100
    verdict: BLOCK
    reason: "Rate limit exceeded"
```

## Go API

```go
import "github.com/Mindburn-Labs/helm-oss/core/pkg/bundles"

// Load from file
bundle, err := bundles.LoadFromFile("policy-bundle.yaml")

// Verify integrity
err = bundles.Verify(bundle, expectedHash)

// Compose multiple bundles
composed, err := bundles.Compose(bundle1, bundle2)

// Inspect without activating
info := bundles.Inspect(bundle)
```

## Validation

Bundles are validated on load:

1. **Schema check** — YAML structure matches expected format
2. **Rule validation** — All rules have id, action, and valid verdict
3. **Content hash** — SHA-256 of canonical bundle content (deterministic)
4. **Composition check** — Conflicting rule IDs across bundles are detected

## Bundle Composition

Multiple bundles can be composed into a single policy set:

- Rules are merged by ID (first bundle wins on conflict)
- Conflicts are detected and reported
- The composed result has a content-addressed hash
- Rule ordering is deterministic (sorted by ID)

## CLI

```bash
# List loaded bundles
helm bundle list

# Verify bundle integrity
helm bundle verify policy-bundle.yaml --hash <expected>

# Inspect bundle without loading
helm bundle inspect policy-bundle.yaml
```
