# HELM OSS Final Assurance: Adoption Blockers

Generated: 2026-03-07

Question answered here:

> What still prevents HELM OSS from being a complete, industry-grade operating standard that third parties can adopt with near-zero ambiguity?

## Cross-Cutting Blockers

These blockers apply to almost every adopter class:

1. There is no single authoritative public contract.
   - The repo ships two OpenAPI specs plus a third, different runtime route surface.
   - Third parties cannot tell which one is normative.
2. The proof/verification story is not trustworthy enough to be standard-grade.
   - `helm verify` does not perform real signature verification.
   - Proof Report does not bind to the actual EvidencePack bytes.
3. The ecosystem surface overclaims reality.
   - Responses WS, `.mcpb`, sandbox providers, and several framework adapters are presented as shipping despite being stubbed or missing.
4. Release and install surfaces are inconsistent.
   - Wrong repo URLs, broken `go install`, mismatched asset naming, missing public packages, and incomplete latest release assets.
5. Policy-bundle trust is specified but not enforced.
   - No single bundle format, no mandatory signature verification, no trust-root or revocation enforcement.

## By Adopter Type

### 1. Framework Authors

What blocks them:

- Public adapter semantics are not substrate-truthful.
  - TS/Python adapters fabricate receipts or call nonexistent endpoints.
- The runtime contract they are supposed to target is ambiguous.
  - SDKs are generated from a different spec than the one CI validates, and both differ from server routes.
- Claimed framework coverage exceeds shipped packages.
  - Google ADK, CrewAI, LlamaIndex, PydanticAI, Semantic Kernel, AutoGen, and the .NET bridge are not independently consumable packages in this repo.

Why that matters:

- A framework author cannot build a conformant adapter without reverse-engineering Go internals and deciding which public contract to believe.

Minimum fixes before adoption:

1. Publish one canonical API/IDL surface.
2. Make adapters consume real kernel-issued receipts and evidence.
3. Publish only the frameworks that exist as tested, installable packages.

### 2. Runtime Vendors

What blocks them:

- Provider compatibility claims are currently based on stubbed CLI behavior.
- `sandbox conform` always passes static checks instead of exercising real provider behavior.
- Provider-specific version binding, digest binding, and replay semantics are not proven through the public runtime path.

Why that matters:

- A runtime vendor cannot certify compatibility from the shipped CLI or weekly matrix because those surfaces do not currently prove real execution semantics.

Minimum fixes before adoption:

1. Route CLI/provider conformance through the real adapters.
2. Fail CI on provider compatibility drift instead of masking failures.
3. Publish provider-specific conformance evidence and reproducible smoke tests.

### 3. Auditors and Independent Verifiers

What blocks them:

- The verifier’s cryptographic verification is effectively a no-op.
- Evidence/export/replay commands do not form a single reproducible public workflow.
- Proof Report hash binding is incorrect.
- Bundle trust requirements from the RFC are not enforced by the runtime.

Why that matters:

- An auditor cannot rely on HELM’s “offline verification” claim if the verification path accepts structurally valid but unauthenticated artifacts.

Minimum fixes before adoption:

1. Implement real public-key verification in `helm verify`.
2. Bind reports to actual artifact bytes.
3. Make demo, export, verify, and replay share one canonical EvidencePack layout.
4. Enforce bundle trust before load.

### 4. Enterprises

What blocks them:

- Auth/identity claims are ahead of the runtime.
- Package licenses and repository metadata disagree.
- Release/install trust is not stable enough for procurement or production governance.
- Several public routes are intentionally left unauthenticated while docs imply a broader auth matrix.

Why that matters:

- Enterprises need one defensible answer for install provenance, auth boundaries, module identity, licensing, and policy enforcement. HELM OSS does not currently provide that answer.

Minimum fixes before adoption:

1. Align module paths, repo URLs, and package licenses.
2. Publish only auth modes that are truly wired end to end.
3. Ship verifiable public releases with consistent assets and install docs.
4. Remove or lock down header/default identity fallbacks.

### 5. Open-Source Developers

What blocks them:

- Core docs instruct users to run commands that do not work as written.
- `go install` from README fails because module identity still points at the old repo path.
- Package-install claims for CLI and adapters are not true in public registries.
- Duplicate and fake-green CI surfaces make it hard to know what is actually healthy.

Why that matters:

- Contributors cannot easily reproduce the documented happy path or trust the published compatibility story.

Minimum fixes before adoption:

1. Make every README/quickstart command part of CI.
2. Fix repo/module/release identity drift.
3. Remove duplicate compatibility workflows and masked failures.
4. Remove tracked/generated build output and tighten clean-tree checks.

## Stakeholder-Specific Bottom Line

| Adopter | Current outcome | Primary blockers |
| --- | --- | --- |
| Framework authors | `No` | ambiguous public contract, fabricated adapter receipts, missing package surfaces |
| Runtime vendors | `No` | stubbed provider CLI and false-positive conformance |
| Auditors | `No` | fake signature verification, broken proof binding, non-canonical EvidencePack flow |
| Enterprises | `No` | auth drift, license/repo drift, install/release trust gaps |
| OSS developers | `Partial` | source is available, but documented install/use path is not reliable |

## What Still Prevents “Near-Zero Ambiguity”

The decisive blockers are not missing polish. They are missing truth alignment:

1. one contract
2. one trustworthy verification path
3. one real bundle trust model
4. one reproducible install/release story
5. one honest ecosystem matrix that only marks shipping surfaces as shipping

Until those five are fixed, HELM OSS remains an ambitious codebase with strong ideas, not a third-party-ready operating standard.
