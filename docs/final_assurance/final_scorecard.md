# HELM OSS Final Assurance: Final Scorecard

Generated: 2026-03-07

Scoring scale:

- `0-2` = not credible for public standard positioning
- `3-4` = promising components exist, but the end-to-end surface is still inconsistent
- `5-6` = usable with significant caveats
- `7-8` = strong, externally defensible
- `9-10` = industry-grade, low-ambiguity, independently verifiable

## Scorecard

| Category | Score | Justification |
| --- | --- | --- |
| Standard maturity | `3/10` | The repo contains serious spec work, reason-code registries, packs, RFCs, and formal fragments, but there is no single authoritative public contract and major runtime surfaces still diverge from the published standard. |
| Runtime truth | `2/10` | Core server, proxy, MCP, sandbox, EvidencePack, and adapter paths do not line up. Multiple user-facing commands are stubbed or behave differently from their claims. |
| Security / trust | `2/10` | `helm verify` does not perform real signature verification, bundle trust is not enforced, and auth/identity claims are broader than the runtime. |
| Distribution quality | `2/10` | Install script, public release assets, package registry presence, module identity, and repo URLs are inconsistent. |
| Ecosystem readiness | `2/10` | Claimed framework/client coverage materially exceeds tested and installable surfaces. Several published integrations are docs-first or proof-fabricating. |
| Independent verifiability | `2/10` | Verification is mostly structural, not cryptographic; proof report binding is wrong; public contracts differ from runtime. |
| Global operability | `3/10` | Jurisdiction and industry packs exist, but pack trust/composition semantics are not enforced through one canonical runtime path. |
| Developer experience | `4/10` | The repo is rich in docs and examples, but too many of the flagship commands and install paths fail as written. |
| Release engineering | `2/10` | Broken build paths, masked workflow failures, incomplete external channels, and release/version skew prevent a credible release story. |
| Google-level polish | `1/10` | The repo does not currently meet the “assume nothing and verify everything” bar. Too many surfaces are aspirational or contradictory. |

## Executive Answers

Can top Google engineers adopt this now?

- `No`

Would a third party implementer succeed without private knowledge?

- `No`

What must be fixed before public positioning as an operating standard?

1. Collapse to one canonical public contract and enforce runtime/spec parity in CI.
2. Make `helm verify` perform real cryptographic verification and bind reports to actual artifact bytes.
3. Replace stubbed product surfaces (`/v1/responses` WS, MCP server, `.mcpb`, provider conformance) with working implementations or demote them from shipping status.
4. Enforce one real policy-bundle trust model with signatures, trust roots, revocation, and fail-closed loading.
5. Repair the install/release chain: repo URLs, module paths, asset naming, package metadata, and external registries must agree.
6. Remove fabricated adapter proofs and require all published adapters to consume kernel-issued receipts.

## Short Verdict

HELM OSS is not ready to position as a complete operating standard.

The repo contains substantial raw material:

- a large standards narrative
- meaningful core primitives
- real protocol/spec effort
- some solid runtime and SDK components

But the current product surface fails the decisive standard test:

> a skeptical third party should be able to install it, run the documented path, verify the artifact chain, and implement against the public contract without reading private tribal knowledge

That is not true today.
