# HELM Certification Model

> Certification levels, badges, and renewal for HELM-conformant implementations.

## 1. Certification Levels

| Level | Name           | Requirements                                                                    | Badge                                                   |
| ----- | -------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------- |
| 1     | **Compatible** | Pass Level 1-2 conformance vectors. Self-certified.                             | ![Compatible](https://helm.sh/badges/compatible-v1.svg) |
| 2     | **Verified**   | Pass Level 4 conformance. CI artifacts published. Reviewed by HELM maintainers. | ![Verified](https://helm.sh/badges/verified-v1.svg)     |
| 3     | **Sovereign**  | Verified + TLA+ invariant coverage + independent audit.                         | ![Sovereign](https://helm.sh/badges/sovereign-v1.svg)   |

## 2. Certification Process

### 2.1 Self-Certification (Compatible)

1. Run conformance vectors: `helm conform run --level 2`
2. Submit results PR to `compatibility-registry.json`
3. Badge granted upon merge

### 2.2 Verified Certification

1. Run full conformance: `helm conform run --level 4`
2. Publish CI workflow artifact showing pass
3. Submit PR with CI link
4. HELM maintainer reviews within 14 days
5. Badge granted upon approval

### 2.3 Sovereign Certification

1. Complete Verified certification
2. Engage independent auditor from approved auditor list
3. Auditor verifies TLA+ invariant alignment
4. Submit audit report
5. HELM governance committee reviews
6. Badge granted upon committee approval

## 3. Renewal

| Level      | Renewal Period | Required                                             |
| ---------- | -------------- | ---------------------------------------------------- |
| Compatible | 12 months      | Re-run vectors against latest spec version           |
| Verified   | 6 months       | Re-run Level 4 conformance + CI proof                |
| Sovereign  | 12 months      | Re-audit (desk review acceptable if no spec changes) |

## 4. Badge Usage

```markdown
[![HELM Compatible](https://helm.sh/badges/compatible-v1.svg)](https://helm.sh/conformance)
[![HELM Verified](https://helm.sh/badges/verified-v1.svg)](https://helm.sh/conformance)
[![HELM Sovereign](https://helm.sh/badges/sovereign-v1.svg)](https://helm.sh/conformance)
```

## 5. Revocation

Certification may be revoked if:

- Conformance vectors fail against a new spec version and are not remediated within 90 days
- Material security vulnerability is found in the implementation
- Misleading claims about certification level
