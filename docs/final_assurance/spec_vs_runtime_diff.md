# HELM OSS Final Assurance: Spec vs Runtime Diff

Generated: 2026-03-07

This document records concrete places where the published contract, docs, or normative text diverge from the behavior actually shipped in the repository.

## Contract-Level Diffs

| Claim Surface | What the spec/docs say | What the runtime actually does | Affected files |
| --- | --- | --- | --- |
| OpenAPI source of truth | A public OpenAPI contract exists and is validated. | Two different OpenAPI files exist. SDKs are generated from `api/openapi/helm.openapi.yaml`, but the drift test validates `docs/api/openapi.yaml`. | `api/openapi/helm.openapi.yaml`; `docs/api/openapi.yaml`; `scripts/sdk/gen.sh`; `core/pkg/api/apidoc_drift_test.go` |
| Health and version endpoints | Primary OpenAPI and SDKs use `/healthz` and `/version`. | Server mode exposes `:8081/health` and `/api/v1/version`; `/healthz` returns 404 and `/version` is not the documented public system endpoint. | `api/openapi/helm.openapi.yaml:357-388`; `sdk/*`; `core/cmd/helm/main.go:540-579`; `core/cmd/helm/subsystems.go:151-158` |
| `/v1/chat/completions` | Published as a governed OpenAI-compatible proxy with receipt headers and upstream semantics. | Server mode registers a mock handler that blocks on `PASS`, does not upstream, and returns a hardcoded response. Proxy sidecar implements a different code path. | `api/openapi/helm.openapi.yaml:56-107`; `core/cmd/helm/subsystems.go:25-77`; `core/pkg/api/openai_proxy.go:61-160`; `core/cmd/helm/proxy_cmd.go` |
| Responses WS support | README and integration docs tell users to run `helm proxy --websocket` and connect SDKs to `/v1/responses`. | The handler returns HTTP 501 with `websocket_not_ready`. | `README.md:81-98`; `docs/VERIFICATION.md:85-101`; `docs/INTEGRATIONS/ORCHESTRATORS.md:47-53`; `core/cmd/helm/proxy_cmd.go:511-549` |
| MCP server support | Docs present stdio/HTTP/SSE MCP server support and one-click desktop packaging. | `mcp serve` prints readiness text and blocks forever; no MCP protocol server is started. `.mcpb` packing never writes the requested output file. | `README.md:44-55`; `docs/INTEGRATIONS/MCP_CLIENTS.md:21-63`; `core/cmd/helm/mcp_cmd.go:55-99`; `core/cmd/helm/mcp_cmd.go:212-295` |
| EvidencePack wow path | Docs instruct `helm export --evidence ./data/evidence --out evidence.tar` followed by `helm verify --bundle evidence.tar`. | Export requires `--audit` or `--incident`; demo evidence is in a flat layout that exports zero canonical sections; verifier expects a directory, not the tar file docs instruct users to pass. | `README.md:81-98`; `docs/START_HERE.md:44-52`; `docs/QUICKSTART.md:75-90`; `core/cmd/helm/export_cmd.go:24-155`; `core/cmd/helm/verify_cmd.go:24-112`; `core/pkg/verifier/verifier.go:105-278` |
| Replay contract | Replay is described as part of the EvidencePack verification loop. | Replay requires an `08_TAPES` directory, which the documented demo/export path does not produce. | `docs/QUICKSTART.md:89-90`; `core/cmd/helm/replay_cmd.go:23-120` |
| Proof Report hash binding | Report is positioned as a proof artifact for the generated EvidencePack. | The report publishes `policyHash` as `evidencepack_sha256`, so report truth is not bound to the actual artifact bytes. | `core/cmd/helm/proof_report.go:460-509` |
| Bundle trust RFC | RFC requires manifest bundles, Ed25519 signature verification before load, trust roots, revocation, and EvidencePack recording of active bundles. | The YAML loader only hashes files and loads `*.policy.yaml`; a second JSON loader uses a different schema and action set. No trust root or revocation enforcement is wired into bundle load. | `protocols/specs/rfc/policy-bundle-v1.md`; `core/pkg/governance/bundles/loader.go`; `core/pkg/governance/bundles/signer.go`; `core/pkg/policyloader/loader.go` |
| Auth matrix | Auth docs claim shipping support for API key, client-local session, and MCP header auth, plus smoke-test commands. | Runtime auth is JWT middleware on non-public routes; `/v1/chat/completions` is public; `helm smoke-test` does not exist; credential handlers fall back to `X-Operator-ID` and `default-operator`. | `docs/specs/AUTH_MATRIX.md`; `core/cmd/helm/main.go`; `core/pkg/auth/middleware.go`; `core/pkg/credentials/handlers.go` |
| Client ecosystem commands | Docs instruct `helm config generate` and `mcp-server`. | Real CLI surface is `helm mcp <serve|install|pack|print-config>`. | `docs/specs/CLIENT_ECOSYSTEM.md`; `core/cmd/helm/main.go`; `core/cmd/helm/mcp_cmd.go` |
| SDK verify/replay request shape | OpenAPI uses multipart upload for `evidence/verify` and `replay/verify`. | Go and Java SDKs send JSON `bundle_b64`; TS/Python/Rust use multipart. All SDKs target endpoints absent from current server mode. | `api/openapi/helm.openapi.yaml:240-301`; `sdk/go/client/client.go:149-160`; `sdk/java/src/main/java/labs/mindburn/helm/HelmClient.java:156-175`; `sdk/python/helm_sdk/client.py:120-134`; `sdk/rust/src/lib.rs:185-220`; `sdk/ts/src/client.ts:117-150` |
| Framework adapter proof semantics | Adapter docs claim HELM-governed execution with receipts and EvidencePack generation. | Claimed adapters synthesize receipts locally, export local tarballs, or call endpoints not implemented in the runtime. | `sdk/ts/openai-agents/src/index.ts`; `sdk/ts/mastra/src/index.ts`; `sdk/python/openai_agents/helm_openai_agents.py`; `sdk/python/microsoft_agents/helm_ms_agent.py`; `docs/INTEGRATIONS/ORCHESTRATORS.md` |
| Compatibility matrix truth | Docs say compatibility is verified weekly. | One workflow masks failures with `continue-on-error`; another regexes logs after `|| true`; sandbox compatibility checks do not exercise real providers. | `docs/COMPATIBILITY.md`; `.github/workflows/compatibility_matrix.yml`; `.github/workflows/compat-matrix.yml`; `core/cmd/helm/sandbox_cmd.go` |

## Demo and Example Diffs

| Surface | Claimed behavior | Observed behavior | Affected files |
| --- | --- | --- | --- |
| Demo completion banner | Tells users to `helm export --evidence data/evidence --out e.tar` then `helm verify --bundle e.tar`. | First command fails unless `--audit` or `--incident` is added. | `core/cmd/helm/demo_cmd.go` runtime output; `core/cmd/helm/export_cmd.go` |
| Claude Desktop installation | `helm mcp pack --client claude-desktop --out helm.mcpb` produces a bundle users can double-click. | Runtime prints success but leaves no `helm.mcpb` file behind. | `README.md`; `docs/INTEGRATIONS/MCP_CLIENTS.md`; `core/cmd/helm/mcp_cmd.go` |
| `helm version --verify` | Verify-install guide says it validates checksum and signature. | CLI ignores `--verify` and only prints version/schema lines. | `docs/VERIFY_INSTALL.md`; `core/cmd/helm/main.go` |

## Release / Distribution Diffs

External checks here were performed on 2026-03-07.

| Surface | Claimed behavior | Observed behavior | Affected files |
| --- | --- | --- | --- |
| GitHub release completeness | Workflow and docs imply release bundles include checksums, SBOM, evidence bundle, golden artifacts, and verification material. | Latest GitHub release found was `v0.1.0` with 6 assets only: `helm-*`, `helm-node`, and `SHA256SUMS.txt`. No `.sig`, SBOM, `.mcpb`, compatibility matrix, or golden artifacts were present. | `.github/workflows/release.yml`; `docs/VERIFY_INSTALL.md`; GitHub Releases |
| Homebrew install | README tells users to `brew install mindburn-labs/tap/helm`. | `Mindburn-Labs/homebrew-tap` returned 404 during audit. | `README.md`; `.goreleaser.yml`; Homebrew/GitHub |
| npm CLI/adapters | README and release workflow claim `@mindburn/helm`, `@mindburn/helm-openai-agents`, and `@mindburn/helm-mastra`. | Those packages were not found on npm during audit; only `@mindburn/helm-sdk` was found. | `README.md`; `.github/workflows/release.yml`; `packages/mindburn-helm-cli/package.json`; npm registry |
| PyPI adapters | README claims `pip install helm-openai-agents helm-agent-framework helm-langchain`. | Those packages were not found on PyPI during audit; only `helm-sdk` was found. | `README.md`; `.github/workflows/release.yml`; PyPI |
| Maven/NuGet | Docs and workflows imply Java and .NET distribution. | No Maven Central artifact and no NuGet package were found during audit. NuGet workflow explicitly skips if `sdk/dotnet` is absent. | `.github/workflows/release.yml`; `sdk/java/pom.xml`; `docs/INTEGRATIONS/ORCHESTRATORS.md` |
| `go install` from README | README tells users to install from `github.com/Mindburn-Labs/helm-oss/core/cmd/helm@latest`. | Command fails because `core/go.mod` still declares `github.com/Mindburn-Labs/helm/core`. | `README.md`; `core/go.mod` |

## Bottom Line

HELM OSS currently has three simultaneous contract surfaces:

1. the legacy `docs/api/openapi.yaml`
2. the newer `api/openapi/helm.openapi.yaml` used for SDK generation
3. the runtime route set actually registered by the server and CLI

As long as those three surfaces remain different, the repository cannot credibly position itself as a low-ambiguity operating standard.
