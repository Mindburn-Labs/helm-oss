# HELM OSS Full Standard Execution Plan

Status: `NOT READY`

Last reviewed: `2026-03-07`

This document is the execution program for getting HELM OSS to the state required for:

- public release as the full operating standard
- broad package/channel distribution
- externally defensible ecosystem claims

This is not a marketing roadmap. It is a release control document.

## Non-negotiable rule

A phase is complete only when all of the following are true:

1. normative spec exists
2. runtime implementation exists
3. CI enforces it
4. demo/example proves it
5. legacy contradiction is deleted or explicitly deprecated

If any one is missing, the phase remains `PARTIAL`.

## Current repo snapshot

As of `2026-03-07`, the repo has meaningful partial coverage across most phases, but it is not at the “complete operating standard” bar.

| Phase | Status | Blocking reality |
| --- | --- | --- |
| Phase 0: Release truth baseline | `PARTIAL` | Core build/test is healthier, but module identity, artifact hygiene, and channel truth are still mixed |
| Phase 1: Atomic standard | `PARTIAL` | Reason-code assets exist, but schema indexing and canonical vocabulary enforcement are not fully normalized |
| Phase 2: Portable substrate | `PARTIAL` | Substrate docs exist, but portable OpenAPI/protobuf IDL is not finalized under canonical paths |
| Phase 3: Formal assurance and conformance | `PARTIAL` | Conformance vectors exist, but runner/job naming and external-consumer verification surface are incomplete |
| Phase 4: Policy bundles | `PARTIAL` | Bundle specs exist, but runtime loader/verifier/fetch/revocation path is not yet a clearly isolated substrate package |
| Phase 5: Jurisdiction and industry packs | `PARTIAL` | Pack files exist, but explicit composition/conflict-resolution contract and schemas are not complete |
| Phase 6: Runtime determinism and replay | `PARTIAL` | Runtime and proof paths exist, but deterministic replay, WASI, and anchoring truth still need stricter gates |
| Phase 7: Shared codegen and auth | `PARTIAL` | Auth docs exist, but shared codegen pipeline and generated SDK source-of-truth are not complete |
| Phase 8: Client ecosystem | `PARTIAL` | Client docs and MCP surfaces exist, but canonical packaging/demo/smoke coverage is incomplete |
| Phase 9: Framework integrations | `PARTIAL` | Several adapters/examples exist, but primary integration matrix is not uniformly package + test + example + CI |
| Phase 10: Capability packs and business presets | `PARTIAL` | Pack/preset concepts exist, but normalized capability/business pack layout and runnable reference architectures are incomplete |
| Phase 11: Output economics and automation | `PARTIAL` | Verify/install and some trust docs exist, but artifact budgeting and automation visibility are not fully enforced |
| Phase 12: Release perfection and ecosystem insertion | `PARTIAL` | Reusable actions and some third-party docs exist, but full channel truth and compatibility publication are incomplete |

## Required execution order

Work must proceed in this order to avoid fake completeness:

1. Phase 0: baseline release truth
2. Phase 1: standard vocabulary and schema control
3. Phase 2: substrate IDL finalization
4. Phase 3: formal/conformance extraction
5. Phase 4 and Phase 5 together: bundles plus executable packs
6. Phase 6: runtime determinism hardening
7. Phase 7: codegen and auth matrix
8. Phase 8 and Phase 9 together: clients plus frameworks
9. Phase 10: capability/business operability
10. Phase 11: artifact economics and automation
11. Phase 12: release perfection and ecosystem insertion

No public “full operating standard” positioning is allowed before all twelve are complete.

## Phase 0: Release truth baseline

Objective: make the repo honest enough that future phases are measurable.

### Required outcomes

- eliminate committed/generated build junk from release-critical surfaces
- normalize path drift between old `helm` and current `helm-oss` identities
- ensure every published channel in CI corresponds to an actually supported package
- ensure release docs match the actual release workflow

### Current repo reality

- `core/go.mod` now correctly declares Go `1.25.0`
- `release.yml` is now syntactically valid and more honest
- npm adapters and NuGet publish are explicitly omitted until real
- old module identity remains in `tools/helm-node/go.mod`
- multiple speculative or in-progress workflow changes are still present in the worktree

### Required repo changes

- normalize repo/module/package identity across:
  - `core/go.mod`
  - `sdk/go/go.mod`
  - `tools/helm-node/go.mod`
  - docs/install snippets
  - package metadata
- remove or isolate committed generated outputs from source control:
  - `sdk/python/build/`
  - `sdk/python/dist/`
  - `sdk/ts/dist/`
  - `sdk/rust/target/`
  - other generated release artifacts
- reconcile `.github/workflows/*.yml` to only advertise real channels
- create a single release-channel truth table in docs

### CI gates

- release workflow lint
- repo hygiene guard
- install verification smoke test
- package metadata consistency check

### Stop condition

Phase 0 is done only when the repo’s published channels, docs, metadata, and workflows describe exactly the same release surface.

## Phase 1: Atomic standard

Objective: make the core vocabulary final and machine-enforced.

### Required assets

- canonical reason-code registry
- canonical verdict vocabulary
- artifact versioning and identifier RFC alignment
- `schema_version` on all emitted artifacts

### Current repo reality

- reason-code RFC and JSON schema exist
- schema index exists at `protocols/json-schemas/SCHEMA_INDEX.md`
- vocabulary drift still exists in parts of the repo and docs
- not all asset locations match the target normalized plan

### Required repo changes

- enforce one canonical verdict enum source
- enforce one reason-code registry source
- add CI guard to reject old or duplicate vocabulary
- prove emitted report/EvidencePack/receipt outputs always include `schema_version`
- normalize target path naming where needed:
  - either keep `protocols/json-schemas/SCHEMA_INDEX.md` as canonical or move to `protocols/json-schemas/index/`
  - update docs and CI to match the chosen location

### CI gates

- `spec-vocabulary`
- schema-version presence check
- emitted artifact vocabulary regression test

### Stop condition

Phase 1 is done only when vocabulary and reason codes round-trip from spec to runtime to emitted artifacts without hand-maintained drift.

## Phase 2: Portable substrate

Objective: expose `effect()` and PDP as portable, language-neutral substrate contracts.

### Current repo reality

- substrate docs exist:
  - `docs/specs/EFFECT_BOUNDARY.md`
  - `docs/specs/PDP_IDL.md`
  - `docs/specs/PORTABLE_EFFECT_MODEL.md`
- canonical IDL layout is not yet complete under `protocols/idl/openapi/` and `protocols/idl/protobuf/`
- some route and runtime behavior is still anchored in Go implementation details

### Required repo changes

- add or normalize:
  - `protocols/idl/openapi/effect-boundary-v1.yaml`
  - `protocols/idl/openapi/pdp-v1.yaml`
  - `protocols/idl/protobuf/effect_boundary.proto`
  - `protocols/idl/protobuf/pdp.proto`
- align HTTP and gRPC lifecycle semantics:
  - `submit`
  - `approve`
  - `deny`
  - `execute`
  - `complete`
  - `get lifecycle`
  - `check idempotency`
- ensure CLI substrate-facing commands call the same canonical lifecycle
- add one non-Go wire-client test in CI

### CI gates

- IDL sync gate
- wire parity tests for HTTP and gRPC
- idempotency semantics regression suite

### Stop condition

Phase 2 is done only when a non-Go consumer can use the public substrate contract without reading Go internals.

## Phase 3: Formal assurance and conformance extraction

Objective: bind formal invariants to real vectors and external verification.

### Current repo reality

- TLA-related CI exists at `.github/workflows/tla-check.yml`
- conformance vectors exist under `protocols/conformance/v1/`
- guide naming and runner tooling are not yet normalized to the target plan

### Required repo changes

- add canonical runner tooling under `tools/conformance/`
- normalize conformance guide placement or document the canonical path
- add explicit external-consumer verifier examples
- fail CI on altered invalid vectors

### CI gates

- TLA model-checker job
- conformance vector runner job
- external verifier compatibility job

### Stop condition

Phase 3 is done only when runtime invariants are enforced by public vectors and at least one verifier path does not import internal packages.

## Phase 4: Policy bundles become real

Objective: move governance from compiled-only logic into signed, loadable bundles.

### Current repo reality

- policy-bundle RFC and trust-model docs exist
- bundle files exist under `protocols/bundles/`
- runtime package layout does not yet present a clean, canonical `core/pkg/bundles/` contract as described in the plan

### Required repo changes

- implement or normalize bundle runtime package:
  - loader
  - signer
  - verifier
  - fetcher
  - revocation
- bind active bundle metadata into proof artifacts
- finish CLI surface:
  - install
  - list
  - verify
  - update
  - pin
  - revoke or disable

### CI gates

- tamper rejection
- strict-mode unsigned bundle rejection
- revocation tests
- update path tests

### Stop condition

Phase 4 is done only when policy execution truth depends on loaded bundles, not only compiled Go paths.

## Phase 5: Jurisdiction and industry packs become executable

Objective: make packs enforceable, composable, and visible in runtime.

### Current repo reality

- jurisdiction and industry bundle files exist
- some composition docs already exist in `docs/specs/`
- conflict-resolution contract and canonical pack schemas are incomplete

### Required repo changes

- finalize canonical schemas for jurisdiction and industry packs
- implement explicit conflict resolution and unresolved-conflict escalation
- bind active pack set into Proof Report and EvidencePack
- reject array-order-driven semantics

### CI gates

- pack composition regression tests
- namespace-aware deny-code validation
- deterministic multi-jurisdiction conflict suite

### Stop condition

Phase 5 is done only when composed packs govern runtime behavior and surface audibly in proof artifacts.

## Phase 6: Runtime completion and determinism hardening

Objective: eliminate stub, nondeterministic, or unverifiable runtime behavior.

### Current repo reality

- runtime, replay, and proof subsystems exist
- mock and sandbox coverage exist
- deterministic replay, WASI, and anchoring still need stronger proof-oriented gates

### Required repo changes

- isolate or remove non-production stubs
- add deterministic replay test suite
- add strict wall-clock audit for proof-critical packages
- add end-to-end WASI and anchoring integration tests

### CI gates

- deterministic replay suite
- WASI execution suite
- anchoring integration suite
- `time.Now()` grep guard in restricted packages

### Stop condition

Phase 6 is done only when replay stability and proof verification are reproducible across reruns and supported runtimes.

## Phase 7: Shared codegen and auth matrix

Objective: remove manual SDK drift and make auth claims testable.

### Current repo reality

- auth docs exist
- multiple SDKs exist
- type generation is not yet the single source of truth
- adapter publishability still trails core SDK publishability

### Required repo changes

- add canonical codegen pipeline under `tools/codegen/`
- create `sdk/generated/`
- make generated artifacts authoritative for shared wire types
- add auth smoke tests for:
  - provider API key
  - Google OAuth
  - client local session
  - MCP header auth
  - MCP OAuth passthrough
  - enterprise token

### CI gates

- codegen diff gate
- generated SDK compile gate
- auth matrix smoke jobs

### Stop condition

Phase 7 is done only when shared SDK types are generated and every claimed auth profile has a passing smoke path.

## Phase 8: Client ecosystem

Objective: make HELM installable where developers actually work.

### Current repo reality

- MCP packaging exists
- client docs exist, but under mixed locations
- examples/client packaging is not yet normalized to the target matrix

### Required repo changes

- normalize client docs/examples under canonical locations
- ensure each claimed client has:
  - install path
  - generated config
  - demo
  - smoke test
- validate `.mcpb` outputs in CI

### CI gates

- client config generation validation
- `.mcpb` validation
- client smoke tests where automation is feasible

### Stop condition

Phase 8 is done only when every claimed client surface is installable, documented, packaged, and tested.

## Phase 9: Framework and orchestration integrations

Objective: own the governed tool boundary inside the frameworks that matter.

### Current repo reality

- several framework SDKs/examples exist
- support depth is uneven across primary targets
- not every claimed integration has package + test + example + CI

### Required repo changes

- classify integrations as:
  - native
  - bridge
  - experimental
  - docs-only
- promote primary frameworks to full package/test/example/CI coverage
- keep secondary frameworks behind honest labels until equivalent coverage exists

### CI gates

- one job per primary integration
- OpenAI JS WS path coverage
- proof artifact generation checks

### Stop condition

Phase 9 is done only when every primary integration can generate receipts, Proof Report, EvidencePack, and a verify path under CI.

## Phase 10: Capability packs, business presets, reference architectures

Objective: make HELM useful in real organizations, not only as a kernel demo.

### Current repo reality

- business and pack docs exist in partial form
- canonical `packs/` layout is not fully normalized
- reference architectures are not yet clearly surfaced as a serious catalog

### Required repo changes

- create or normalize:
  - `packs/capabilities/`
  - `packs/business/`
  - `docs/reference-architectures/`
- ensure `helm init business --preset <name>` produces a governed and useful setup
- provide at least five runnable serious reference architectures

### CI gates

- preset validation
- pack install and compose tests
- reference-architecture smoke tests

### Stop condition

Phase 10 is done only when presets and architectures are runnable and proof-visible, not brochure content.

## Phase 11: Output economics, automation, trust surface

Objective: make proof scalable, budgeted, and continuously maintained.

### Current repo reality

- verify-install surface exists
- automation and economics docs exist in partial/renamed form
- budget enforcement is not yet a single, explicit release gate

### Required repo changes

- normalize economics and retention docs
- enforce artifact size and mode budgets
- implement visible automation outputs for:
  - maintenance
  - provider drift
  - docs freshness
  - release drafting
  - registry/badge updates

### CI gates

- artifact budget gate
- install verification smoke test
- automation drift checks

### Stop condition

Phase 11 is done only when the wow path stays within explicit time and artifact budgets and automation output is visible and auditable.

## Phase 12: Release perfection and ecosystem insertion

Objective: make HELM something third parties can build around without private knowledge.

### Current repo reality

- reusable GitHub actions exist
- some third-party/RFC process docs exist
- full channel matrix and compatibility publication are incomplete
- some channels are correctly omitted because they are not yet real

### Required repo changes

- create the final implementation report and keep it live
- publish compatibility registry and update flow
- finalize reusable GitHub Action against a fixture repo
- complete third-party implementation guide with vector-based validation path
- keep unsupported channels explicitly out of public claims until implemented

### CI gates

- release artifact completeness
- fixture-repo reusable action test
- compatibility registry validation
- third-party vector packaging validation

### Stop condition

Phase 12 is done only when a third party can implement, verify, and consume HELM without reading private context or reverse-engineering Go internals.

## Branch and PR strategy

Use one branch per phase slice. Recommended order:

1. `codex/phase0-release-truth`
2. `codex/phase1-vocabulary`
3. `codex/phase2-portable-substrate`
4. `codex/phase3-formal-conformance`
5. `codex/phase4-bundles`
6. `codex/phase5-pack-execution`
7. `codex/phase6-runtime-determinism`
8. `codex/phase7-codegen-auth`
9. `codex/phase8-client-ecosystem`
10. `codex/phase9-framework-matrix`
11. `codex/phase10-packs-and-architectures`
12. `codex/phase11-artifact-economics`
13. `codex/phase12-release-perfection`

Each PR must contain:

- spec changes
- runtime changes
- CI changes
- demo/test evidence
- deletion/deprecation of legacy contradictions

## Honest channel policy

Do not publish or claim support for a channel unless its package, tests, and examples are green on the release branch.

Current policy:

- GitHub Releases: allowed once release artifact completeness is green
- GHCR: allowed once image verification and smoke tests are green
- npm core SDK and CLI: allowed once publish jobs and package verification are green
- npm adapters: keep withheld until standalone publishability is real
- PyPI: allowed once release workflow and package smoke tests are green
- crates.io: allowed once release workflow and publish dry run are green
- Maven Central: allowed once release workflow and package verification are green
- NuGet: withheld until an actual .NET SDK exists and is tested

## Final go/no-go rule

Public positioning as the full operating standard is blocked until:

- all twelve phases are `COMPLETE`
- the live implementation report says `READY`
- every claimed distribution channel is real
- every claimed client/framework/runtime surface has package + test + example + CI
- third-party implementation works without private knowledge
