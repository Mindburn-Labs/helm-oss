# HELM Output Economics

> Size budgets, compact capsule format, and retention tiers for HELM artifacts.

## 1. Artifact Size Budgets

| Artifact                   | Max Size | CI Gate  | Notes                                  |
| -------------------------- | -------- | -------- | -------------------------------------- |
| **Receipt** (individual)   | 10 KB    | ❌ warn  | Single governance decision             |
| **Proof Report** (HTML)    | 2 MB     | ✅ block | Includes embedded JSON capsule         |
| **EvidencePack** (tar)     | 5 MB     | ✅ block | All receipts + decisions for a session |
| **Policy Bundle** (signed) | 1 MB     | ✅ block | Policies + signature + manifest        |
| **Compact Capsule** (JSON) | 50 KB    | ❌ warn  | Minimal machine-readable summary       |

## 2. Compact Capsule Format

The Compact Capsule is a < 50KB JSON document that summarizes a governance session:

```json
{
  "capsule_version": "1.0.0",
  "session_id": "urn:helm:session:abc123",
  "created_at": "2026-03-07T00:00:00Z",
  "summary": {
    "total_effects": 42,
    "allowed": 38,
    "denied": 3,
    "escalated": 1,
    "receipts_generated": 42,
    "chain_valid": true
  },
  "jurisdiction": "eu-gdpr",
  "industry": "healthcare-hipaa",
  "preset": "engineering",
  "verdict_distribution": {
    "ALLOW": 38,
    "DENY": 3,
    "ESCALATE": 1
  },
  "deny_reasons": {
    "DENY_GDPR_CROSS_BORDER_TRANSFER": 2,
    "POLICY_VIOLATION": 1
  },
  "chain_head": "sha256:latest-receipt-hash",
  "proof_report_url": "https://helm.sh/reports/abc123.html"
}
```

## 3. Retention Tiers

| Tier          | Retention  | Storage                  | Use Case                             |
| ------------- | ---------- | ------------------------ | ------------------------------------ |
| **Hot**       | 30 days    | In-memory / Redis        | Active governance, real-time queries |
| **Warm**      | 1 year     | PostgreSQL / S3          | Audit trails, compliance reviews     |
| **Cold**      | 7 years    | S3 Glacier / GCS Archive | Legal holds, regulatory requirement  |
| **Immutable** | Indefinite | Rekor / transparency log | Proof anchoring, tamper evidence     |

## 4. CI Budget Enforcement

```yaml
# In .github/workflows/repo-cleanup-guards.yml
- name: Check artifact budgets
  run: |
    # Check EvidencePack < 5MB
    find dist/ -name '*.evidence-pack.tar' -size +5M -exec echo "❌ {} exceeds 5MB" \; -exec exit 1 \;
    # Check ProofReport < 2MB
    find dist/ -name '*.proof-report.html' -size +2M -exec echo "❌ {} exceeds 2MB" \; -exec exit 1 \;
```
