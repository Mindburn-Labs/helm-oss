---
title: HELM Roadmap
---

# HELM Roadmap

No dates. Each item tied to a conformance level.

---

| # | Item | Conformance | Status |
|---|------|-------------|--------|
| 1 | **Conformance L3** — formal verification (SMT/LTL solvers) | L3 | Spec'd, not shipped |
| 2 | **ZK-CPI** — zero-knowledge constraint proofs for cross-org verification | L3 | Spec'd (RFC-002) |
| 3 | **Hardware TEE attestation** — bind kernel verdicts to silicon quotes (Nitro, TDX, SEV-SNP) | L2+ | Spec'd (RFC-002) |
| 4 | **Post-quantum cryptography** — PQ signatures for long-term evidence | L2+ | Research |
| 5 | **Multi-org ProofGraph federation** — cross-tenant merge with deterministic conflict resolution | L3 | Spec'd |
| 6 | **Production key management** — HSM integration, key rotation ceremony | L2 | Partial (ephemeral signer shipped) |
| 7 | **Cognitive engine pinning** — detect silent model swaps, force re-wargaming | L2+ | Spec'd (Section 4.3) |
| 8 | **Dead Man's Clock** — heartbeat-based freeze + recovery quorum | L2 | Spec'd (Section 4.5) |
| 9 | **Proof condensation** — cryptographic compaction for long-running workflows | L2 | Spec'd (Section 8) |
| 10 | **A2A envelope** — agent-to-agent interop with trust negotiation | L3 | Spec'd (Section 12) |

---

Items 1–5 are enterprise-track. Items 6–10 are hardening the OSS kernel.

Contributions welcome: see [CONTRIBUTING.md](../CONTRIBUTING.md).
