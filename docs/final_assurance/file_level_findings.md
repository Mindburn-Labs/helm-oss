# HELM OSS Final Assurance: File-Level Findings

Generated: 2026-03-07

Scope notes:

- Every tracked file in the repository was included in the scan.
- This document lists every file or directory that requires action or materially contributes to a blocker.
- Files not listed here were scanned and did not surface a material repo-wide blocker in this pass.

Classification key:

- `OK` — scanned, no material blocker found
- `REWRITE` — fundamental behavior or contract needs redesign
- `DELETE` — remove from repo or published surface
- `INCOMPLETE` — partially implemented, not shippable
- `MISLEADING` — user-facing claim exceeds runtime truth
- `SECURITY RISK` — trust, auth, signing, or verification flaw
- `SPEC DRIFT` — implementation diverges from spec or canonical contract
- `TEST GAP` — public claim lacks meaningful automated proof
- `RELEASE GAP` — release/install/package channel is broken or missing

## Canonical / Runtime-Critical Files

| Path | Classification | Reason |
| --- | --- | --- |
| `core/cmd/helm/subsystems.go` | `REWRITE` | Registers a mock `/v1/chat/completions`, uses `PASS` instead of canonical verdicts, and does not expose the API surface SDKs expect. |
| `core/cmd/helm/proxy_cmd.go` | `INCOMPLETE` | WS mode is documented as usable but returns 501; SSE governance is deferred post hoc. |
| `core/cmd/helm/mcp_cmd.go` | `REWRITE` | `mcp serve` never starts an MCP protocol loop; `.mcpb` pack does not write output. |
| `core/cmd/helm/sandbox_cmd.go` | `MISLEADING` | Non-mock providers are stubs and conformance is hardcoded pass. |
| `core/cmd/helm/export_cmd.go` | `SPEC DRIFT` | Export contract does not match docs or demo output layout; tar output behavior does not align with documented usage. |
| `core/cmd/helm/verify_cmd.go` | `SECURITY RISK` | Invokes signature verification with a verifier callback that always succeeds. |
| `core/cmd/helm/replay_cmd.go` | `SPEC DRIFT` | Requires `08_TAPES` directory layout that the documented export path does not produce. |
| `core/cmd/helm/proof_report.go` | `SPEC DRIFT` | Labels `policyHash` as `evidencepack_sha256`, weakening proof/report binding. |
| `core/pkg/api/openai_proxy.go` | `INCOMPLETE` | More realistic than server-mode route, but still not the canonical handler actually registered by server mode. |
| `core/pkg/api/apidoc_drift_test.go` | `TEST GAP` | Validates `docs/api/openapi.yaml` instead of the spec used for SDK generation. |
| `core/pkg/conform/report.go` | `SECURITY RISK` | Signature verification is delegated to a callback; current caller passes a no-op verifier. |
| `core/pkg/conform/gates/gx_sdk_drift.go` | `TEST GAP` | Checks stale OpenAPI and SDK paths, so it cannot catch current drift. |
| `core/pkg/verifier/verifier.go` | `INCOMPLETE` | Structural verification only; no tar support, no semantic Lamport or replay verification. |
| `core/pkg/governance/bundles/loader.go` | `SPEC DRIFT` | Loads YAML bundles without RFC-required signature, trust-root, or revocation checks. |
| `core/pkg/governance/bundles/signer.go` | `INCOMPLETE` | Supports signing/verification in isolation, but not as an enforced runtime load path. |
| `core/pkg/policyloader/loader.go` | `REWRITE` | Defines a second, incompatible bundle format and action vocabulary. |
| `core/pkg/auth/middleware.go` | `SPEC DRIFT` | Published auth matrix does not match actual runtime auth behavior or public-route exemptions. |
| `core/pkg/credentials/handlers.go` | `SECURITY RISK` | Falls back to `X-Operator-ID` or `default-operator`, undermining identity binding. |
| `core/pkg/console/server.go` | `SPEC DRIFT` | Public-path bypass and route exposure do not line up with published API surface. |
| `core/go.mod` | `RELEASE GAP` | Module path still points at `github.com/Mindburn-Labs/helm/core`, breaking documented `go install` from `helm-oss`. |

## Adapter / Integration Files

| Path | Classification | Reason |
| --- | --- | --- |
| `sdk/ts/openai-agents/src/index.ts` | `MISLEADING` | Fabricates approval receipts locally instead of consuming kernel-issued proofs. |
| `sdk/ts/mastra/src/index.ts` | `MISLEADING` | Bypasses substrate truth by wrapping Daytona directly and synthesizing receipts. |
| `sdk/python/openai_agents/helm_openai_agents.py` | `MISLEADING` | Calls nonexistent `/v1/tools/evaluate`, builds local Lamport chains, and exports local tar packs. |
| `sdk/python/microsoft_agents/helm_ms_agent.py` | `MISLEADING` | Same local receipt fabrication and nonexistent endpoint dependency. |
| `sdk/python/langchain/helm_langchain.py` | `INCOMPLETE` | Wrapper exists, but it does not prove substrate-level receipt/evidence semantics. |
| `sdk/go/client/client.go` | `SPEC DRIFT` | Targets endpoints the server does not expose and uses JSON `bundle_b64` for verify/replay despite multipart OpenAPI. |
| `sdk/java/src/main/java/labs/mindburn/helm/HelmClient.java` | `SPEC DRIFT` | Same endpoint drift as Go; verify/replay payload shape diverges from OpenAPI. |
| `sdk/python/helm_sdk/client.py` | `SPEC DRIFT` | Generated client targets `/healthz`, `/version`, proofgraph/evidence/replay endpoints absent from runtime. |
| `sdk/rust/src/lib.rs` | `SPEC DRIFT` | Same endpoint drift as Python/TS/Go/Java. |
| `sdk/ts/src/client.ts` | `SPEC DRIFT` | Same endpoint drift as other SDKs; only TS verify/replay matches OpenAPI multipart shape. |

## Test / Fixture / Formal Files

| Path | Classification | Reason |
| --- | --- | --- |
| `protocols/conformance/v1/test-vectors.json` | `TEST GAP` | Canonical vectors exist but are not executed by runtime or CI. |
| `protocols/specs/tla/HelmKernel.tla` | `TEST GAP` | Formal model exists, but CI runs only on TLA file changes, not semantic runtime changes. |
| `docs/api/openapi.yaml` | `DELETE` | Legacy second OpenAPI document kept alive by tests while SDKs use another file. |
| `api/openapi/helm.openapi.yaml` | `SPEC DRIFT` | Primary public contract disagrees with the actual server route surface. |
| `scripts/sdk/gen.sh` | `OK` | Correctly identifies the primary SDK-generation spec and thereby exposes the drift with tests/runtime. |

## Workflow / Release Files

| Path | Classification | Reason |
| --- | --- | --- |
| `Makefile` | `RELEASE GAP` | Root build points at nonexistent `apps/helm-node`. |
| `.github/workflows/release.yml` | `RELEASE GAP` | Broken helm-node path, masked npm adapter publishes, no real NuGet artifact, and release promises exceed current outputs. |
| `.github/workflows/helm_core_gates.yml` | `RELEASE GAP` | References nonexistent `apps/helm-node` path for build and vuln scan. |
| `.github/workflows/compatibility_matrix.yml` | `TEST GAP` | Uses `|| true` and `continue-on-error`, so matrix output is not a hard truth surface. |
| `.github/workflows/compat-matrix.yml` | `DELETE` | Duplicate compatibility workflow with different semantics and more masked failures. |
| `.github/workflows/repo-cleanup-guards.yml` | `MISLEADING` | Artifact-budget and signature steps print success without doing real validation. |
| `.github/workflows/tla-check.yml` | `TEST GAP` | Path-triggered only; runtime semantics can drift with no model check. |
| `.goreleaser.yml` | `RELEASE GAP` | Describes archive/checksum outputs that `install.sh` and docs do not actually consume consistently. |
| `install.sh` | `RELEASE GAP` | Wrong repo, wrong asset naming, wrong checksum model, and unverifiable bypass path. |

## Docs / Spec / User-Facing Claim Files

| Path | Classification | Reason |
| --- | --- | --- |
| `README.md` | `MISLEADING` | 10-minute proof loop, `.mcpb` install, WS mode, and package-install claims do not survive runtime validation. |
| `docs/VERIFICATION.md` | `MISLEADING` | Repeats broken export/verify and Responses WS claims. |
| `docs/START_HERE.md` | `MISLEADING` | Documents export/verify flow that does not match runtime behavior. |
| `docs/QUICKSTART.md` | `MISLEADING` | Same broken EvidencePack flow and old clone URL. |
| `docs/TROUBLESHOOTING.md` | `MISLEADING` | Tells users to run the same broken export/verify path as a fix. |
| `docs/COMPATIBILITY.md` | `MISLEADING` | Marks MCP clients and sandbox providers as compatible beyond runtime truth. |
| `docs/INTEGRATIONS/MCP_CLIENTS.md` | `MISLEADING` | Claims a cross-platform `.mcpb` bundle that the packer does not create. |
| `docs/INTEGRATIONS/ORCHESTRATORS.md` | `MISLEADING` | Claims Responses WS mode and NuGet/.NET surfaces not supported end to end. |
| `docs/specs/CLIENT_ECOSYSTEM.md` | `DELETE` | Documents obsolete commands (`helm config generate`, `mcp-server`) that do not exist. |
| `docs/specs/AUTH_MATRIX.md` | `MISLEADING` | Shipping status for auth modes does not match runtime wiring or available CLI surfaces. |
| `docs/VERIFY_INSTALL.md` | `MISLEADING` | Documents `helm version --verify`, cosign signature assets, and SBOM retrieval not proven by the current release state. |
| `docs/PUBLISHING.md` | `SPEC DRIFT` | Repository URLs and release instructions still point at `Mindburn-Labs/helm` in several places. |
| `docs/index.md` | `MISLEADING` | Contains old repo URLs and public-claim drift similar to `README.md`. |

## Package Metadata / Distribution Manifests

| Path | Classification | Reason |
| --- | --- | --- |
| `packages/mindburn-helm-cli/package.json` | `RELEASE GAP` | Package exists locally, but was not found on npm during audit; metadata points to old repo and BSL license. |
| `sdk/ts/package.json` | `SPEC DRIFT` | Points to old repo URL and assumes runtime endpoints that do not exist. |
| `sdk/ts/openai-agents/package.json` | `RELEASE GAP` | Package was not found on npm during audit. |
| `sdk/ts/mastra/package.json` | `RELEASE GAP` | Package was not found on npm during audit. |
| `sdk/python/pyproject.toml` | `OK` | PyPI SDK package exists, but runtime drift remains in generated client code. |
| `sdk/rust/Cargo.toml` | `SPEC DRIFT` | Published crate points to old repo URL and BUSL license, conflicting with root Apache license. |
| `sdk/java/pom.xml` | `RELEASE GAP` | Maven metadata points to old repo URL and no Maven Central artifact was found during audit. |
| `sdk/go/go.mod` | `SPEC DRIFT` | Still uses old repo path, reinforcing repo/module identity drift. |

## Generated / Legacy / Hygiene Problems

| Path | Classification | Reason |
| --- | --- | --- |
| `sdk/python/build/lib/helm_sdk/__init__.py` | `DELETE` | Tracked generated build output. |
| `sdk/python/build/lib/helm_sdk/client.py` | `DELETE` | Tracked generated build output. |
| `sdk/python/build/lib/helm_sdk/types_gen.py` | `DELETE` | Tracked generated build output. |
| `sdk/rust/target/` | `RELEASE GAP` | Ignored local residue is extremely large and makes clean-state auditing harder; CI should enforce clean worktrees around release/build steps. |
| `sdk/java/target/` | `RELEASE GAP` | Ignored local residue in checkout; not tracked, but should be guarded more aggressively. |
| `sdk/ts/dist/` | `RELEASE GAP` | Ignored generated output in checkout; not tracked, but contributes to clean-state drift risk. |
| `packages/mindburn-helm-cli/dist/` | `RELEASE GAP` | Ignored generated output in checkout; not tracked, but contributes to clean-state drift risk. |
