# HELM OSS Final Assurance: Security and Trust Audit

Generated: 2026-03-07

This audit focuses on the trust chain, install chain, auth model, bundle trust, artifact verification, and external distribution truth.

## 1. Install Chain and Binary Trust

### Primary findings

- `install.sh` is not aligned with the actual release pipeline.
  - It hardcodes `REPO="Mindburn-Labs/helm"`.
  - It downloads `${BIN_NAME}-${OS}-${ARCH}` while `.goreleaser.yml` publishes archives named `helm-{{ .Os }}-{{ .Arch }}`.
  - It expects `${BINARY_URL}.sha256` while goreleaser emits `SHA256SUMS.txt`.
  - It exposes `HELM_SKIP_VERIFY=1`, which turns the happy path into an unverifiable install.
- `docs/VERIFY_INSTALL.md` documents `helm version --verify`, but the CLI does not implement verification in the `version` command.
- The release workflow promises more than the latest public GitHub release currently contains.

### Evidence

- `install.sh:7-79`
- `.goreleaser.yml:33-47`
- `docs/VERIFY_INSTALL.md:7-80`
- `core/cmd/helm/main.go:113-119`
- External check on 2026-03-07: latest GitHub release found was `v0.1.0` with 6 assets only:
  - `helm-darwin-amd64`
  - `helm-darwin-arm64`
  - `helm-linux-amd64`
  - `helm-linux-arm64`
  - `helm-node`
  - `SHA256SUMS.txt`

### Security impact

- Users cannot follow a single, trustworthy install-and-verify path from repo docs.
- Release verification material described in docs is not consistently present at the public release surface.

## 2. Artifact Verification and Evidence Trust

### Primary findings

- `helm verify` does not perform real cryptographic verification of report signatures.
  - `runVerifyCmd` passes a verifier callback that always returns success.
  - `VerifyReport` trusts that callback for the final signature decision.
- `runConform` emits a fallback `.sig` with `"algorithm": "sha256-hmac"` without computing or verifying a keyed MAC.
- `core/pkg/verifier/verifier.go` is positioned as an adversarial standalone verifier, but most checks are structural:
  - lamport monotonicity = “N receipt files present”
  - policy decision hashes = structural pass
  - replay determinism = “tape files exist”
- Proof Report hash binding is wrong: `evidencepack_sha256` is derived from `policyHash`, not the exported pack bytes.

### Evidence

- `core/cmd/helm/verify_cmd.go:47-77`
- `core/pkg/conform/report.go:87-134`
- `core/cmd/helm/conform.go:130-165`
- `core/pkg/verifier/verifier.go:105-278`
- `core/cmd/helm/proof_report.go:472-509`

### Security impact

- Forged or altered evidence can pass the advertised verification flow if it preserves the expected structure.
- Users are told they have cryptographic proof when the implementation mostly offers format checks.

## 3. Policy Bundle Trust

### Primary findings

- The RFC requires:
  - signed bundles
  - trust roots
  - revocation checks
  - `apiVersion` enforcement
  - fail-closed loading
  - EvidencePack recording of active bundles
- The shipped runtime does not enforce that model.
  - YAML bundle loader hashes files and loads them.
  - JSON CEL bundle loader defines a second, incompatible bundle format.
  - No trust-root check is performed before load.
  - No revocation check is performed.
  - No single canonical bundle structure matches the RFC end to end.

### Evidence

- `protocols/specs/rfc/policy-bundle-v1.md:27-195`
- `core/pkg/governance/bundles/loader.go:84-143`
- `core/pkg/governance/bundles/signer.go:31-82`
- `core/pkg/policyloader/loader.go:17-153`

### Security impact

- Bundle authenticity is optional in practice.
- A third party cannot tell which bundle format is authoritative or what runtime checks are mandatory before a bundle is trusted.

## 4. Auth and Identity Trust

### Primary findings

- The public auth matrix is materially ahead of the runtime.
  - Shipping claims exist for `Client-local session` and `MCP header auth`.
  - Smoke-test commands such as `helm smoke-test` are documented but not implemented.
- Runtime auth is JWT-only middleware on non-public paths.
  - `/v1/chat/completions` is public.
  - `core/pkg/console/server.go` bypasses auth for multiple routes.
- Credential handlers derive operator identity from `X-Operator-ID` or default to `default-operator`.
- Google OAuth code exists, but the productized end-to-end mode described in docs is not what the main runtime exposes today.

### Evidence

- `docs/specs/AUTH_MATRIX.md:7-121`
- `core/pkg/auth/middleware.go:51-135`
- `core/pkg/console/server.go:245-266`
- `core/pkg/credentials/handlers.go:41-49`
- `core/pkg/credentials/google_oauth.go:22-182`

### Security impact

- Identity binding is weaker than documented.
- Operator scoping can silently collapse to a caller-controlled header or shared default value.
- Public governance routes do not enforce the documented auth matrix.

## 5. MCP, Proxy, and Runtime Trust

### Primary findings

- Responses WS mode is advertised but returns 501.
- MCP server transports are advertised but no actual stdio/HTTP/SSE MCP protocol server is started.
- `.mcpb` packaging claims success while writing no installable bundle.
- Sandbox provider compatibility is overstated:
  - non-mock providers are stubbed in the CLI
  - `sandbox conform` is hardcoded to pass

### Evidence

- `core/cmd/helm/proxy_cmd.go:511-549`
- `core/cmd/helm/mcp_cmd.go:55-99`
- `core/cmd/helm/mcp_cmd.go:212-295`
- `core/cmd/helm/sandbox_cmd.go:130-174,219-304`

### Security impact

- Users can believe they are running governed remote/MCP/runtime flows when they are actually on stubs or nonfunctional surfaces.
- That weakens trust in receipts, policy enforcement, and audit artifacts coming out of those paths.

## 6. CI and Release Trust

### Primary findings

- Compatibility matrix jobs suppress failures with `|| true` and `continue-on-error: true`.
- Repo-cleanup guard steps print successful budget/signature checks without performing real validation.
- The OpenAPI drift test validates a legacy spec, not the spec used for SDK generation.
- The SDK drift gate checks stale paths and can miss real drift.
- Conformance vectors are present on disk but not executed by runtime or CI.

### Evidence

- `.github/workflows/compatibility_matrix.yml:23-34`
- `.github/workflows/compat-matrix.yml:27-48`
- `.github/workflows/repo-cleanup-guards.yml:81-99`
- `core/pkg/api/apidoc_drift_test.go:10-58`
- `core/pkg/conform/gates/gx_sdk_drift.go:28-96`
- `protocols/conformance/v1/test-vectors.json`

### Security impact

- CI green does not mean contract-conformant, release-ready, or independently verifiable.
- Public badges and compatibility claims overstate what is actually enforced.

## 7. External Distribution Reality Check

Registry checks performed on 2026-03-07.

| Channel | Observed state | Trust conclusion |
| --- | --- | --- |
| GitHub Releases | Latest release found: `v0.1.0`; only 6 binary/checksum assets visible | Public release state lags repo/runtime claims and lacks several verification artifacts described in docs |
| Homebrew | `Mindburn-Labs/homebrew-tap` returned `404` | README install path is not externally defensible |
| npm `@mindburn/helm-sdk` | Found `0.1.1` | One SDK channel exists, but it targets drifted runtime endpoints |
| npm `@mindburn/helm` | Not found | CLI distribution claim is not true today |
| npm `@mindburn/helm-openai-agents` | Not found | Adapter distribution claim is not true today |
| npm `@mindburn/helm-mastra` | Not found | Adapter distribution claim is not true today |
| PyPI `helm-sdk` | Found `0.1.1` | One SDK channel exists, but runtime drift remains |
| PyPI `helm-openai-agents` | Not found | Adapter distribution claim is not true today |
| PyPI `helm-agent-framework` | Not found | Adapter distribution claim is not true today |
| PyPI `helm-langchain` | Not found | Adapter distribution claim is not true today |
| crates.io `helm-sdk` | Found `0.1.1` | One SDK channel exists, but runtime drift remains |
| Maven Central `ai.mindburn.helm:helm-sdk` | Not found | Java distribution claim is not externally verified |
| NuGet `Mindburn.Helm.Governance` | Not found | .NET distribution claim is not externally verified |
| GHCR `ghcr.io/mindburn-labs/helm-oss/helm:latest` | Unauthenticated `docker manifest inspect` returned `manifest unknown` | Public container claim could not be confirmed from the external surface used in this audit |

## 8. Security Posture Summary

Current trust posture is not release-grade for an operating standard because the repo simultaneously has:

1. an install path that is not aligned with public release assets
2. a verification path that does not cryptographically verify what it claims
3. a bundle trust model that is specified but not enforced
4. an auth matrix that is broader than the runtime
5. CI surfaces that over-report compatibility and validation

The most important security fixes, in order:

1. Make `helm verify` perform real signature verification and remove fake fallback signatures.
2. Collapse to one canonical OpenAPI contract and enforce runtime/spec parity in CI.
3. Enforce bundle trust at load time with real trust roots and revocation checks.
4. Fix the install chain so public assets, checksums, docs, and CLI verification all match.
5. Remove or clearly demote stubbed WS/MCP/runtime surfaces until they are fully implemented and tested.
