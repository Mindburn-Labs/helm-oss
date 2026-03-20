# Product Ladder

HELM ships as a four-tier product ladder. The free layers drive adoption; the paid layers deliver managed operations.

## Tiers

### Free 1 — Hosted Governed Agent (`try.mindburn.run`)

Use a HELM-governed AI assistant for free. No signup, no install. Every action gets a cryptographic receipt.

- OpenClaw agent on HELM governance
- Free models via OpenRouter (Llama 3.1 8B)
- 50 requests/day, 10K tokens/request
- Exportable, offline-verifiable receipts
- Fail-closed policy enforcement

→ [Deploy config](deploy/try/) · [Try it live](https://try.mindburn.run)

### Free 2 — OSS Agent Hardening Kit (this repo)

Bring your agent runtime, get fail-closed execution and receipts.

- DeerFlow, OpenClaw, and any OpenAI-compatible runtime
- GitHub Action for CI boundary checks
- Receipt verification (Python + TS)
- Golden artifacts and conformance testing
- SDKs for Go, TypeScript, Python, Rust, Java

→ [Quickstarts](examples/) · [Install](README.md#install)

### Free 3 — Public Proof Surface (`mindburn.org`)

Live intelligence and proof — no install required.

- Lab status with real governed runs and attestations
- Compatibility matrix (weekly provider conformance)
- 6 interactive demos (gate, verify, policy, ProofGraph, conformance, compliance)
- Ecosystem intelligence (DeerFlow, OpenClaw, Crucix, MiroFish)

→ [Lab Status](https://mindburn.org/lab/status) · [Ecosystem Intel](https://mindburn.org/ecosystem)

### Paid 1 — Managed Control Plane (`helm.mindburn.run`)

Private tenant on the HELM managed control plane.

- Tenant isolation with Ed25519 key rotation
- Hosted policy packs and approvals
- Evidence/receipt export pipelines
- Private connectors and onboarding
- SLA and monitoring

First SKU: **HELM Hardening Sprint** — 1–2 week managed engagement.

### Paid 2 — Premium (Titan)

Internal micro-cap pilot. Invite-only.

- DeerFlow reports and MiroFish scenarios in research/offline lane
- Not publicly marketed

---

## Day-1 Free vs Later

| What | Day 1 | Later | Never free |
|------|-------|-------|------------|
| helm-oss repo + releases | ✅ | — | — |
| DeerFlow/OpenClaw quickstarts | ✅ | — | — |
| GitHub Action / CI checks | ✅ | — | — |
| Public demos + receipt verification | ✅ | — | — |
| mindburn.org proof surface | ✅ | — | — |
| Hosted agent (try.mindburn.run) | ✅ | — | — |
| Limited hosted playground | — | ✅ after abuse controls | — |
| Managed deployment | — | — | ❌ |
| Private tenant | — | — | ❌ |
| Policy pack authoring | — | — | ❌ |
| Compliance intelligence | — | — | ❌ |

---

## Links

- Website: [mindburn.org](https://mindburn.org)
- Runtimes: `*.mindburn.run`
- OSS: [github.com/Mindburn-Labs/helm-oss](https://github.com/Mindburn-Labs/helm-oss)
