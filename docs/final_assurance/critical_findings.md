# HELM OSS Final Assurance: Critical Findings

Generated: 2026-03-07
Audit scope: full tracked repository scan (`git ls-files` = 1547 files), targeted runtime reproductions, and external release/registry checks.

## P0 Existential Blockers

- ID: `F-001`
- Severity: `P0`
- Category: `Release Gap`
- File(s): `Makefile:6-7`; `.github/workflows/release.yml:76-81`; `.github/workflows/helm_core_gates.yml:133-155`
- Problem: The canonical build and release path still references `apps/helm-node`, but the repository ships `tools/helm-node`. Root build and release jobs fail before producing the advertised binary set.
- Why it matters: The repo cannot reproduce its own published build surface from HEAD. That breaks release credibility and blocks independent consumers from validating artifacts.
- Proof / evidence: `make build` fails with `go: chdir apps/helm-node: no such file or directory`.
- Reproduction path: From repo root, run `make build`.
- Recommended fix: Replace every `apps/helm-node` reference with `tools/helm-node` or restore the missing path. Add a CI gate that runs the exact root build used by users.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-002`
- Severity: `P0`
- Category: `Runtime Bug`
- File(s): `core/cmd/helm/mcp_cmd.go:212-295`; `README.md:44-55`; `docs/INTEGRATIONS/MCP_CLIENTS.md:21-34`; `docs/VERIFICATION.md:105-122`
- Problem: `helm mcp pack` reports success but never writes the requested `.mcpb` file. It creates a temporary directory, deletes it on exit, and tells users to install a file that does not exist.
- Why it matters: The repo claims one-click Claude Desktop distribution, but the shipped command cannot produce an installable bundle. This is a direct product-surface falsehood.
- Proof / evidence: The implementation only writes `outPath + ".tmp"` and returns success. Runtime reproduction: `helm mcp pack --client claude-desktop --out /tmp/helm.mcpb` prints success; `ls /tmp` shows no `helm.mcpb`.
- Reproduction path: Run `./bin/helm mcp pack --client claude-desktop --out ./helm.mcpb && ls -la .`.
- Recommended fix: Emit a real `.mcpb` archive at `--out`, keep it on disk, and add a smoke test that opens or validates the produced bundle.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? No
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-003`
- Severity: `P0`
- Category: `False Claim`
- File(s): `core/cmd/helm/proxy_cmd.go:511-549`; `README.md:81-98`; `docs/VERIFICATION.md:85-101`; `docs/INTEGRATIONS/ORCHESTRATORS.md:47-53`
- Problem: Responses WebSocket mode is documented as available, but the runtime returns HTTP 501 with `websocket_not_ready` and no upgrade path.
- Why it matters: Frameworks and SDKs that depend on `/v1/responses` WS semantics cannot work against the shipped runtime. This is not partial support; it is an explicitly unimplemented endpoint marketed as usable.
- Proof / evidence: The handler writes `http.StatusNotImplemented`. Runtime reproduction: start `helm proxy --websocket`; a WS request to `/v1/responses` gets a 501 error payload.
- Reproduction path: `./bin/helm proxy --websocket --upstream https://api.openai.com/v1`, then connect to `ws://localhost:9090/v1/responses`.
- Recommended fix: Remove all shipping claims until the endpoint performs real WS upgrade, frame handling, close semantics, receipt emission, and backpressure control under test.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-004`
- Severity: `P0`
- Category: `Spec Drift`
- File(s): `api/openapi/helm.openapi.yaml:56-373`; `docs/api/openapi.yaml:1-431`; `scripts/sdk/gen.sh:3-18`; `core/pkg/api/apidoc_drift_test.go:10-58`; `core/cmd/helm/subsystems.go:20-228`; `core/cmd/helm/main.go:521-579`
- Problem: There are two divergent OpenAPI specs. SDKs are generated from `api/openapi/helm.openapi.yaml`, but the drift test validates `docs/api/openapi.yaml`. The actual server registers neither surface completely.
- Why it matters: Third parties cannot infer the public contract from the repo. The repo simultaneously publishes multiple incompatible contracts and validates the wrong one in CI.
- Proof / evidence: `scripts/sdk/gen.sh` targets `api/openapi/helm.openapi.yaml`; `TestOpenAPISpec_Integrity` loads `docs/api/openapi.yaml`; the runtime exposes `/api/v1/version` and `:8081/health` while SDKs and primary OpenAPI use `/version` and `/healthz`. Runtime probe: `/healthz` returns 404, `/version` returns 401, `/api/v1/version` exists instead.
- Reproduction path: start `./bin/helm server`, then `curl http://localhost:8080/healthz`, `curl http://localhost:8080/version`, `curl http://localhost:8080/api/v1/version`.
- Recommended fix: Collapse to a single canonical OpenAPI file, generate SDKs from it, validate that exact file in CI, and fail builds when runtime route registration drifts.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-005`
- Severity: `P0`
- Category: `Security`
- File(s): `core/cmd/helm/verify_cmd.go:47-77`; `core/pkg/conform/report.go:87-134`; `core/cmd/helm/conform.go:130-165`
- Problem: `helm verify` does not actually verify report signatures. It passes a verifier callback that always returns `nil`, and unsigned fallback artifacts are labeled `sha256-hmac` without any HMAC key or MAC verification.
- Why it matters: The repo claims offline cryptographic verification, but the shipped verifier accepts forged signatures as long as the file layout looks right. That breaks the trust model at the point users are told to rely on it.
- Proof / evidence: `runVerifyCmd` calls `conform.VerifyReport(bundle, func(data []byte, sig string) error { return nil })`. `runConform` emits a fallback `.sig` with `"algorithm": "sha256-hmac"` but computes no HMAC.
- Reproduction path: Inspect the code paths above; tamper with `07_ATTESTATIONS/conformance_report.sig` while keeping structure valid and the verifier will not reject based on cryptographic signature failure.
- Recommended fix: Remove fake signature algorithms, require a real public-key verification path, and fail closed when no trusted verification key is configured.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-006`
- Severity: `P0`
- Category: `Ecosystem`
- File(s): `sdk/ts/openai-agents/src/index.ts:148-216`; `sdk/ts/mastra/src/index.ts:182-260`; `sdk/python/openai_agents/helm_openai_agents.py:112-231`; `sdk/python/microsoft_agents/helm_ms_agent.py:76-191`
- Problem: Claimed framework integrations fabricate receipts locally or call nonexistent runtime endpoints such as `/v1/tools/evaluate`. They do not consume kernel-issued proofs.
- Why it matters: The ecosystem surface claims governed execution with evidence, but third-party adopters would build on adapters that bypass the actual substrate and invent proof artifacts client-side.
- Proof / evidence: TS OpenAI Agents synthesizes `status: 'APPROVED'`, empty hashes, and `lamport_clock: 0`. Python adapters build their own Lamport chains and export tarballs locally. `/v1/tools/evaluate` is referenced by adapters but not implemented anywhere in `core/`.
- Reproduction path: `rg -n "/v1/tools/evaluate|lamport_clock: 0|status: 'APPROVED'" sdk core`.
- Recommended fix: Stop publishing adapters until they consume real server receipts and proof surfaces. Add integration tests that assert kernel-issued receipt IDs, hashes, signatures, and replayability.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

## P1 Serious Blockers

- ID: `F-007`
- Severity: `P1`
- Category: `Bundle Trust`
- File(s): `protocols/specs/rfc/policy-bundle-v1.md:27-195`; `core/pkg/governance/bundles/loader.go:84-143`; `core/pkg/governance/bundles/signer.go:31-82`; `core/pkg/policyloader/loader.go:17-153`
- Problem: The RFC requires signed manifest-based bundles, trusted keys, revocation checks, and fail-closed loading. The runtime ships two incompatible loaders instead: one YAML loader that only hashes files and one JSON CEL loader with a different schema and action vocabulary.
- Why it matters: Policy packs cannot be treated as verifiable, portable, or standard-compliant if bundle loading semantics differ from the RFC and from each other.
- Proof / evidence: The YAML loader never verifies signatures or trust roots and only scans `*.policy.yaml`; the JSON loader reads `.json` bundles with actions `"BLOCK"`, `"WARN"`, `"LOG"`. No revocation or `apiVersion` enforcement exists.
- Reproduction path: Compare the RFC structure and trust requirements with the loader implementations above.
- Recommended fix: Choose one bundle format, make signature verification mandatory before load, enforce trust roots and revocation, and add conformance tests against the RFC examples.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-008`
- Severity: `P1`
- Category: `Release Gap`
- File(s): `install.sh:7-79`; `.goreleaser.yml:33-47`; `README.md:34-41`; `docs/VERIFY_INSTALL.md:7-80`
- Problem: The install and verification path is internally inconsistent and externally incomplete. `install.sh` targets `Mindburn-Labs/helm`, downloads raw binaries instead of goreleaser archives, expects per-binary `.sha256` files instead of `SHA256SUMS.txt`, and offers `HELM_SKIP_VERIFY=1` bypass. The verify-install guide documents `helm version --verify`, but the CLI does not implement verification on `version`.
- Why it matters: Supply-chain trust is only as strong as the first-install path. Today that path either 404s, skips verification, or pretends to verify without any runtime support.
- Proof / evidence: `helm version --verify` prints version information only. `install.sh` hardcodes the wrong repo and asset layout.
- Reproduction path: Run `./bin/helm version --verify`; inspect `install.sh`; compare with `.goreleaser.yml`.
- Recommended fix: Rewrite the installer around actual published assets, remove unverifiable bypasses from the happy path, and implement a real `verify-install` command if it is going to be documented.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-009`
- Severity: `P1`
- Category: `Replay`
- File(s): `README.md:81-98`; `docs/START_HERE.md:44-52`; `docs/QUICKSTART.md:75-90`; `docs/TROUBLESHOOTING.md:96-106`; `core/cmd/helm/export_cmd.go:24-155`; `core/cmd/helm/replay_cmd.go:23-120`; `core/pkg/verifier/verifier.go:45-278`
- Problem: The documented EvidencePack export/verify/replay loop does not match the runtime. `helm export` requires `--audit` or `--incident`; demo output is a flat `data/evidence/` directory instead of the directory structure `export`, `verify`, and `replay` expect; `helm verify` expects a directory while the docs tell users to pass a tar file.
- Why it matters: The repo’s primary proof story is not reproducible by a new adopter. This is the opposite of “near-zero ambiguity.”
- Proof / evidence: Runtime reproduction:
  - `helm demo company --template starter --provider mock` prints `helm export --evidence data/evidence --out e.tar`.
  - Running that command fails with `Error: specify --audit or --incident <id>`.
  - `helm export --evidence ./data/evidence --out exported --audit --tar` exports zero items from the demo directory.
  - `helm verify --bundle exported.tar` fails immediately because the verifier requires a directory.
- Reproduction path: Run the 10-minute wow path exactly as documented in `README.md`.
- Recommended fix: Choose one EvidencePack layout and one invocation model, make demo output conform to it, and gate docs against an end-to-end smoke test.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-010`
- Severity: `P1`
- Category: `CI Gap`
- File(s): `.github/workflows/compatibility_matrix.yml:23-34`; `.github/workflows/compat-matrix.yml:27-48`; `.github/workflows/repo-cleanup-guards.yml:81-99`; `core/pkg/api/apidoc_drift_test.go:10-58`; `core/pkg/conform/gates/gx_sdk_drift.go:28-96`; `protocols/conformance/v1/test-vectors.json`
- Problem: Multiple quality gates are fake-green or validate the wrong target. Compatibility jobs swallow failures with `|| true`; repo-cleanup guards print success without checking artifacts or signatures; API drift tests validate an older spec; the SDK drift gate looks for `sdk/typescript` and outdated spec paths; the conformance vectors are documented but not executed anywhere in code or CI.
- Why it matters: CI appears broader than it is. That creates false confidence exactly where the repo claims standard-grade rigor.
- Proof / evidence: The workflows contain `continue-on-error: true`, `|| true`, and literal success echoes. `test-vectors.json` is only referenced in docs, not runtime or CI code.
- Reproduction path: Inspect the workflow lines above; run `rg -n "test-vectors.json" core scripts .github protocols`.
- Recommended fix: Remove masked failures, execute conformance vectors in CI, validate the canonical OpenAPI spec, and delete duplicate workflows.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-011`
- Severity: `P1`
- Category: `Auth`
- File(s): `docs/specs/AUTH_MATRIX.md:7-121`; `core/pkg/auth/middleware.go:51-135`; `core/pkg/console/server.go:245-266`; `core/pkg/credentials/handlers.go:41-49`
- Problem: The auth matrix marks client-local session and MCP header modes as shipping, but runtime auth is a JWT middleware on non-public routes while `/v1/chat/completions` remains public. Credential handlers fall back to `X-Operator-ID` or `default-operator`.
- Why it matters: The public governance path is not aligned with the documented identity model, and operator scoping can silently degrade to a caller-controlled header or a shared default identity.
- Proof / evidence: `AUTH_MATRIX.md` documents `helm smoke-test` flows and shipping auth modes; `main.go` exposes no `smoke-test` command; `getOperatorID` returns `"default-operator"` when no header is present.
- Reproduction path: Start the server, call public proxy routes without auth, and inspect credential handler identity resolution.
- Recommended fix: Publish only auth modes that are actually wired end-to-end, remove header/default identity fallbacks from production handlers, and add auth smoke tests to CI.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

- ID: `F-012`
- Severity: `P1`
- Category: `Release Gap`
- File(s): `core/go.mod:1-4`; `README.md:38`; `sdk/java/pom.xml:14-22,34-38`; `sdk/rust/Cargo.toml:5-10`; `sdk/ts/package.json:23-26`; `sdk/ts/openai-agents/package.json:26-29`; `sdk/ts/mastra/package.json:26-29`; `packages/mindburn-helm-cli/package.json:35-40`
- Problem: Repository identity, module identity, and licensing all drift. The README tells users to `go install` from `helm-oss`, but `core/go.mod` still declares `github.com/Mindburn-Labs/helm/core`. Several packages point to the old repo URL, while package licenses disagree with the Apache-2.0 root license.
- Why it matters: Install instructions fail, source links are wrong, and legal review becomes ambiguous. This blocks serious adoption even before runtime issues are considered.
- Proof / evidence: `go install github.com/Mindburn-Labs/helm-oss/core/cmd/helm@latest` fails with `module declares its path as github.com/Mindburn-Labs/helm/core but was required as github.com/Mindburn-Labs/helm-oss/core`.
- Reproduction path: Run the `go install` command exactly as documented in `README.md`.
- Recommended fix: Align module paths, repository URLs, and license declarations across every distributable package before further releases.
- Standard impact:
  - does this block third-party implementation? Yes
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

## P2 Important Quality Issues

- ID: `F-013`
- Severity: `P2`
- Category: `Runtime Bug`
- File(s): `core/cmd/helm/subsystems.go:25-77`; `core/pkg/api/openai_proxy.go:61-160`; `core/pkg/contracts/verdict.go:1-27`
- Problem: Server-mode `/v1/chat/completions` is a mock route that blocks on `decision.Verdict != "PASS"` and returns a hardcoded response instead of governed upstream behavior. The more realistic proxy handler in `core/pkg/api/openai_proxy.go` is not what server mode registers.
- Why it matters: The runtime exposes a product-facing route whose behavior contradicts both the canonical verdict vocabulary and the proxy semantics described in docs and SDKs.
- Proof / evidence: `subsystems.go` compares against `"PASS"` while `contracts/verdict.go` defines `ALLOW/DENY/ESCALATE` as canonical.
- Reproduction path: Start `./bin/helm server` and call `POST /v1/chat/completions`; compare behavior with `helm proxy`.
- Recommended fix: Register one canonical proxy implementation for server mode and import verdict constants from `contracts`.
- Standard impact:
  - does this block third-party implementation? Partially
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Partially

- ID: `F-014`
- Severity: `P2`
- Category: `False Claim`
- File(s): `core/cmd/helm/sandbox_cmd.go:130-174,219-304`; `docs/COMPATIBILITY.md:35-54`
- Problem: Non-mock sandbox providers are stubbed in the CLI, and `sandbox conform` returns all-pass static checks regardless of provider availability or real execution.
- Why it matters: Provider compatibility claims cannot be trusted if the user-facing path never touches the real adapters.
- Proof / evidence: `sandbox exec` for `opensandbox`, `e2b`, and `daytona` prints `Command queued ... execute for real`; `sandbox conform` builds a hardcoded list of `Pass: true` checks.
- Reproduction path: Run `./bin/helm sandbox exec --provider daytona -- echo hi` and `./bin/helm sandbox conform --provider opensandbox --tier verified --json`.
- Recommended fix: Route CLI execution and conformance through the real connector packages and require provider-specific integration tests before marking compatibility as `✅`.
- Standard impact:
  - does this block third-party implementation? Partially
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Partially

- ID: `F-015`
- Severity: `P2`
- Category: `Determinism`
- File(s): `core/cmd/helm/proof_report.go:472-509`
- Problem: Proof Report JSON and summary text label `policyHash` as `evidencepack_sha256`, so the report does not bind to the actual EvidencePack bytes it claims to summarize.
- Why it matters: Proof artifacts that display the wrong digest cannot be used for offline verification, attestation exchange, or release evidence.
- Proof / evidence: `evidencepack_sha256` is assigned `shortHash(policyHash, 64)` rather than the hash of the exported pack.
- Reproduction path: Inspect the generated report payload and compare it with the actual hash of any exported evidence bundle.
- Recommended fix: Compute the digest from the actual EvidencePack artifact after export and fail report generation if the pack is absent.
- Standard impact:
  - does this block third-party implementation? Partially
  - does this block enterprise adoption? Yes
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Partially

- ID: `F-016`
- Severity: `P2`
- Category: `Release Gap`
- File(s): `sdk/python/build/lib/helm_sdk/__init__.py`; `sdk/python/build/lib/helm_sdk/client.py`; `sdk/python/build/lib/helm_sdk/types_gen.py`
- Problem: Generated Python build output is tracked in the repository.
- Why it matters: Generated artifacts create stale-source risk, muddy diffs, and can ship outdated generated code that no longer matches the source package.
- Proof / evidence: `git ls-files` includes those three `sdk/python/build/lib/helm_sdk/*` files.
- Reproduction path: Run `git ls-files | rg '^sdk/python/build/'`.
- Recommended fix: Remove tracked build output, keep it in `.gitignore`, and add a cleanliness check to CI.
- Standard impact:
  - does this block third-party implementation? No
  - does this block enterprise adoption? No
  - does this block release credibility? Partially
  - does this block ecosystem distribution? Partially

- ID: `F-017`
- Severity: `P2`
- Category: `False Claim`
- File(s): `docs/specs/CLIENT_ECOSYSTEM.md:20-119`; `core/cmd/helm/main.go:55-129`; `core/cmd/helm/mcp_cmd.go:19-52`
- Problem: Client ecosystem docs still instruct users to run nonexistent commands such as `helm config generate` and `mcp-server`, while the shipped CLI only exposes `helm mcp <serve|install|pack|print-config>`.
- Why it matters: Install docs fail before users can even reach the more substantive runtime problems.
- Proof / evidence: The documented commands are absent from `main.go` and `mcp_cmd.go`.
- Reproduction path: Run `./bin/helm config generate --client claude-code` or `./bin/helm mcp-server`.
- Recommended fix: Delete obsolete client docs or rewrite them against the real CLI surface and smoke-test every documented command.
- Standard impact:
  - does this block third-party implementation? Partially
  - does this block enterprise adoption? Partially
  - does this block release credibility? Yes
  - does this block ecosystem distribution? Yes

## P3 Polish / Improvement

- ID: `F-018`
- Severity: `P3`
- Category: `CI Gap`
- File(s): `.github/workflows/compatibility_matrix.yml`; `.github/workflows/compat-matrix.yml`
- Problem: Two different compatibility-matrix workflows exist and encode different notions of “compatibility.”
- Why it matters: Parallel, partially overlapping workflows make the public compatibility story harder to reason about and easier to drift.
- Proof / evidence: Both workflows are named `Compatibility Matrix`, but one measures sandbox/provider claims and the other regexes SDK test logs.
- Reproduction path: Inspect both workflow files side by side.
- Recommended fix: Merge them into one canonical compatibility pipeline with hard-fail semantics and explicit scope.
- Standard impact:
  - does this block third-party implementation? No
  - does this block enterprise adoption? No
  - does this block release credibility? Partially
  - does this block ecosystem distribution? Partially
