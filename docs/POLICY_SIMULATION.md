---
title: POLICY_SIMULATION
---

# Policy Simulation

HELM OSS supports dry-run oriented demos for policy simulation without changing the boundary model.

## Canonical command

```bash
helm demo organization --template starter --provider mock --dry-run
```

This emits receipts and a manifest with bound scope metadata:

- `organization_id`
- `scope_id`
- `principal_id`
- `execution_mode = dry-run`

## Research-lab example

```bash
helm demo research-lab --template starter --provider mock --dry-run
```

This is useful when you want to test policy, scope, and proof generation before enabling a real deployment surface.

## Why it matters

Policy simulation is not a separate product. It is the same execution boundary, exercised in a mode where organizational authority is inspected before any production rollout.
