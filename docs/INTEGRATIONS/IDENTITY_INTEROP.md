---
title: IDENTITY_INTEROP
---

# Identity Interop

HELM is not an identity provider.

Identity systems answer:

- who is this principal?
- how was it authenticated?

HELM answers:

- is this side effect authorized under the current policy and scope?
- what proof exists for the allow or deny decision?

## Recommended split

- **OIDC / OAuth / SSO**: authenticate humans and services
- **SPIFFE / mTLS / workload identity**: authenticate runtime workloads
- **Teleport / bastion / session brokers**: broker privileged access
- **HELM**: enforce execution authority at the side-effect boundary

## Minimal metadata contract

When a trusted identity layer authenticates a principal, forward stable scope metadata into HELM receipts:

```json
{
  "organization_id": "acme-operations",
  "scope_id": "platform.prod.deploy",
  "principal_id": "devops_lead"
}
```

These fields are optional and backward-compatible. They make it possible to move from local agent governance toward organization-scoped execution without changing the kernel contract.
