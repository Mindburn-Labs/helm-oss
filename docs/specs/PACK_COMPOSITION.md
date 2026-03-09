# HELM Pack Composition Rules

> How jurisdiction, industry, and business packs combine at runtime.

## 1. Pack Types and Precedence

Packs are evaluated in precedence order (highest first):

| Priority    | Pack Type          | Example              | Effect                           |
| ----------- | ------------------ | -------------------- | -------------------------------- |
| 1 (highest) | Jurisdiction       | `eu-gdpr`            | Legal compliance — overrides all |
| 2           | Industry           | `healthcare-hipaa`   | Regulatory compliance            |
| 3           | Business Function  | `engineering`        | Organizational policy            |
| 4 (lowest)  | Corporate Baseline | `corporate-baseline` | Default policies                 |

## 2. Composition Semantics

### 2.1 DENY is Final

A `DENY` verdict from any pack at any priority level is **final**. Lower-priority
packs cannot override a higher-priority DENY.

### 2.2 ESCALATE Propagates

An `ESCALATE` from any pack propagates to the caller. Multiple ESCALATEs merge
into a single escalation with combined reason codes.

### 2.3 ALLOW Requires Consensus

An `ALLOW` verdict is only granted if **no active pack** returns DENY or ESCALATE
for the given effect. This is the fail-closed principle.

## 3. Conflict Resolution

### 3.1 Same-Priority Conflicts

When two packs at the same priority level conflict:

| Conflict          | Resolution              |
| ----------------- | ----------------------- |
| DENY vs ALLOW     | DENY wins (fail-closed) |
| ESCALATE vs ALLOW | ESCALATE wins           |
| DENY vs ESCALATE  | DENY wins               |

### 3.2 Cross-Priority Override

Higher-priority packs cannot be overridden by lower-priority packs:

- Jurisdiction DENY → industry ALLOW = **DENY**
- Industry ESCALATE → business ALLOW = **ESCALATE**
- Business DENY → baseline ALLOW = **DENY**

## 4. Deny Code Extension Model

### 4.1 Namespace Convention

Extended deny codes use a domain prefix:

```
DENY_{DOMAIN}_{CODE}
```

Examples:

- `DENY_GDPR_CROSS_BORDER_TRANSFER` — EU jurisdiction
- `DENY_FINANCE_SOD_VIOLATION` — Finance industry
- `DENY_HEALTHCARE_PHI_UNENCRYPTED` — Healthcare industry

### 4.2 Registration

Extended deny codes MUST be registered in the pack's `deny_codes` section.
The conformance runner validates that every reason code used in a policy
rule has a corresponding deny code registration.

### 4.3 Core vs Extended

| Category        | Namespace                                   | Source                    |
| --------------- | ------------------------------------------- | ------------------------- |
| Core registry | `POLICY_VIOLATION`, `BUDGET_EXCEEDED`, etc. | `reason-codes-v1.json`    |
| Extended        | `DENY_{DOMAIN}_{CODE}`                      | Pack `deny_codes` section |

## 5. "What This Governs" Declaration

Each pack declares what categories of effects it governs:

```yaml
spec:
  governs:
    - Data exports
    - Financial transactions
    - Model training
```

This enables:

- Runtime optimization (skip packs that don't apply)
- Audit trails showing which packs evaluated each effect
- Pack overlap detection

## 6. Pack Registry

### 6.1 Registry Structure

```json
{
  "registry_version": "1.0.0",
  "packs": [
    {
      "name": "eu-gdpr",
      "type": "jurisdiction",
      "version": "1.0.0",
      "jurisdiction": "EU",
      "content_hash": "sha256:...",
      "governs": [
        "data_export",
        "data_process",
        "data_retention",
        "ai_inference"
      ],
      "deny_codes": ["DENY_GDPR_CROSS_BORDER_TRANSFER", "..."]
    }
  ]
}
```

### 6.2 Discovery

```bash
helm pack list                      # List available packs
helm pack search --jurisdiction EU  # Search by criteria
helm pack inspect eu-gdpr           # Show pack details
helm pack validate ./my-pack.yaml   # Validate pack structure
```
