# HELM OSS Final Assurance: Repo Truth Matrix

Generated: 2026-03-07

Scan summary:

- Tracked files reviewed: `1547`
- Top-level distribution: `core 927`, `protocols 279`, `docs 121`, `sdk 56`, `examples 38`, `tools 31`, `scripts 29`, `packages 19`, `.github 14`
- Extra note: the local checkout also contains large ignored build residue (`sdk/rust/target`, `sdk/java/target`, `sdk/ts/dist`, local binaries). Those are not the main blockers below, but they do make clean-state auditing harder.

Legend:

- `✅` fully present and validated in this audit
- `⚠️` present but partial, contradictory, or unverified
- `❌` missing, broken, or false

## Core Standard and Runtime

| Subsystem | Spec | Runtime | Tests | CI | Demo | Release Artifact | Adoption Claim | Third-Party Consumable | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Receipts / receipt RFC | ✅ | ⚠️ | ⚠️ | ⚠️ | ✅ | ⚠️ | ✅ | ⚠️ | Core receipts exist, but adapters fabricate receipts and server/proxy paths disagree on fields and verdicts. |
| Reason-code registry | ✅ | ⚠️ | ⚠️ | ⚠️ | ✅ | ⚠️ | ✅ | ⚠️ | Registry exists, but runtime surfaces also emit legacy/alternate vocabularies and pack loaders do not enforce one path. |
| Verdict vocabulary | ✅ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | ✅ | ⚠️ | Canonical `ALLOW/DENY/ESCALATE` exists, but server-mode `/v1/chat/completions` still checks for `PASS`. |
| Proof Report | ⚠️ | ⚠️ | ❌ | ❌ | ✅ | ⚠️ | ✅ | ❌ | Report renders, but evidence hash is computed from `policyHash`, not pack bytes. |
| EvidencePack export | ✅ | ❌ | ❌ | ❌ | ⚠️ | ⚠️ | ✅ | ❌ | Docs and demo path do not match runtime requirements or output layout. |
| EvidencePack verify | ✅ | ❌ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ✅ | ❌ | Verifier is structural and signature verification is effectively disabled. |
| Replay verify | ✅ | ⚠️ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | Replay requires `08_TAPES` directory layout that demo/export path does not produce. |
| EffectBoundary | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | Core concepts exist, but public HTTP and adapter surfaces do not expose a stable cross-language contract. |
| PDP | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | Guardian/PDP code exists, but external bundle formats, verdict plumbing, and public APIs are inconsistent. |
| Policy bundle format | ✅ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | RFC requires signed manifest bundles; runtime ships incompatible YAML and JSON loaders without trust enforcement. |
| Jurisdiction packs | ✅ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | Packs exist as files, but the enforced loader path does not match the published bundle RFC. |
| Industry packs | ✅ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | Same issue as jurisdiction packs; presence on disk does not equal enforceable standard behavior. |
| Reason-code / pack composition semantics | ✅ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | Composition rules are documented, but not executed through a single conformant bundle engine. |

## Formal Methods and Conformance

| Subsystem | Spec | Runtime | Tests | CI | Demo | Release Artifact | Adoption Claim | Third-Party Consumable | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| TLA+ model | ✅ | N/A | ❌ | ⚠️ | ❌ | ❌ | ✅ | ⚠️ | TLC runs only when TLA files change, not when runtime semantics drift. |
| Conformance CLI | ✅ | ⚠️ | ⚠️ | ⚠️ | ✅ | ⚠️ | ✅ | ⚠️ | Conformance gates run, but some gates validate stale paths and do not execute published vectors. |
| Conformance vectors (`protocols/conformance/v1/test-vectors.json`) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | Vectors are referenced only in docs, not by runtime or CI code. |
| OpenAPI drift check | ✅ | ❌ | ⚠️ | ⚠️ | ❌ | ❌ | ✅ | ❌ | CI validates `docs/api/openapi.yaml`, while SDKs are generated from `api/openapi/helm.openapi.yaml`. |
| SDK drift gate | ✅ | ❌ | ⚠️ | ⚠️ | ❌ | ❌ | ✅ | ❌ | Gate looks for outdated paths like `sdk/typescript` and misses the real TypeScript package. |

## Runtimes, Proxy, and Gateway

| Subsystem | Spec | Runtime | Tests | CI | Demo | Release Artifact | Adoption Claim | Third-Party Consumable | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Mock sandbox runtime | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | This is the only runtime surface that behaves close to its claims. |
| WASI sandbox | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | Core runtime code exists, but public replay/export story is still inconsistent. |
| OpenSandbox provider | ⚠️ | ⚠️ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | Adapter exists, but CLI and compatibility matrix use stubbed execution. |
| E2B provider | ⚠️ | ⚠️ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | Same as OpenSandbox. |
| Daytona provider | ⚠️ | ⚠️ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | Same as OpenSandbox. |
| OpenAI-compatible proxy (`helm proxy`) | ✅ | ⚠️ | ⚠️ | ⚠️ | ✅ | ✅ | ✅ | ⚠️ | Proxy sidecar exists, but SSE governance is deferred and WS mode is unimplemented. |
| Server-mode `/v1/chat/completions` | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | ✅ | ❌ | Server path is a mock handler, not the contract published in OpenAPI/SDKs. |
| Responses WS mode | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | Runtime returns 501. |
| Health/version contract | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | ✅ | ❌ | SDKs and primary OpenAPI use `/healthz` and `/version`; runtime ships `:8081/health` and `/api/v1/version`. |

## MCP and Client Ecosystem

| Subsystem | Spec | Runtime | Tests | CI | Demo | Release Artifact | Adoption Claim | Third-Party Consumable | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| MCP server (stdio/http/sse) | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ | `mcp serve` prints a banner and blocks forever; no MCP protocol loop is started. |
| Claude Code plugin | ✅ | ⚠️ | ❌ | ❌ | ⚠️ | ⚠️ | ✅ | ⚠️ | Plugin files are generated, but they target the stub MCP server. |
| Claude Desktop `.mcpb` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | Packaging command never writes the bundle file. |
| Windsurf/Codex/VS Code/Cursor configs | ✅ | ⚠️ | ❌ | ❌ | ⚠️ | ⚠️ | ✅ | ⚠️ | `print-config` exists, but generated configs point to a stubbed server and docs still reference obsolete commands. |
| Generic MCP client compatibility | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | No end-to-end protocol verification exists in the shipped runtime. |

## SDKs and Framework Adapters

| Subsystem | Spec | Runtime | Tests | CI | Demo | Release Artifact | Adoption Claim | Third-Party Consumable | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Go SDK | ✅ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ✅ | ⚠️ | Client is generated against the primary OpenAPI, but runtime endpoints do not match and `go install` from README fails. |
| TypeScript SDK | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ⚠️ | Published on npm, but points at endpoints the runtime does not expose. |
| Python SDK | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ⚠️ | Published on PyPI, same contract drift as TS/Go. |
| Rust SDK | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ⚠️ | Published on crates.io, same contract drift. |
| Java SDK | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | ❌ | ✅ | ⚠️ | Maven metadata exists locally, but no Maven Central artifact was found during this audit. |
| OpenAI Agents TS adapter | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Fabricates approval receipts locally. |
| OpenAI Agents Python adapter | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Builds local Lamport chain and tarball; calls nonexistent `/v1/tools/evaluate`. |
| Microsoft Agent Framework Python adapter | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Same local receipt fabrication as Python OpenAI adapter. |
| LangChain Python adapter | ✅ | ⚠️ | ⚠️ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Wrapper exists, but it does not prove substrate-level governance semantics. |
| Mastra TS adapter | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Wraps Daytona directly and synthesizes receipts locally. |
| Other claimed frameworks (Google ADK, CrewAI, LlamaIndex, PydanticAI, Semantic Kernel, AutoGen, .NET bridge, Vercel AI SDK) | ⚠️ | ❌ | ❌ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Repo has docs/examples, not shipping adapters or release artifacts. |

## Release and Distribution Surfaces

External registry/release checks in this section were verified on 2026-03-07.

| Subsystem | Spec | Runtime | Tests | CI | Demo | Release Artifact | Adoption Claim | Third-Party Consumable | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GitHub Releases | ✅ | ⚠️ | ⚠️ | ⚠️ | ❌ | ⚠️ | ✅ | ⚠️ | Latest release found was `v0.1.0` with 6 assets only; repo/runtime version is already `v0.2.0`. |
| Install script | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | Wrong repo, wrong asset naming, wrong checksum flow. |
| Homebrew | ✅ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ✅ | ❌ | `Mindburn-Labs/homebrew-tap` returned 404 during audit. |
| npm `@mindburn/helm-sdk` | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ⚠️ | Published (`0.1.1`), but its runtime target is drifted. |
| npm `@mindburn/helm` CLI | ✅ | ❌ | ⚠️ | ⚠️ | ⚠️ | ❌ | ✅ | ❌ | Package was not found on npm during audit. |
| npm adapters (`@mindburn/helm-openai-agents`, `@mindburn/helm-mastra`) | ✅ | ❌ | ⚠️ | ⚠️ | ⚠️ | ❌ | ✅ | ❌ | Packages were not found on npm during audit. |
| PyPI `helm-sdk` | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ⚠️ | Published (`0.1.1`), but runtime target is drifted. |
| PyPI adapters (`helm-openai-agents`, `helm-agent-framework`, `helm-langchain`) | ✅ | ❌ | ⚠️ | ❌ | ⚠️ | ❌ | ✅ | ❌ | Packages were not found on PyPI during audit. |
| crates.io `helm-sdk` | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ | ✅ | ⚠️ | Published (`0.1.1`), but runtime target is drifted. |
| Maven Central `ai.mindburn.helm:helm-sdk` | ✅ | ❌ | ⚠️ | ⚠️ | ⚠️ | ❌ | ✅ | ❌ | No artifact found on Maven Central during audit. |
| NuGet `Mindburn.Helm.Governance` | ✅ | ❌ | ❌ | ⚠️ | ⚠️ | ❌ | ✅ | ❌ | No package found on NuGet; workflow skips if `sdk/dotnet` is absent. |
| GHCR container image | ✅ | ⚠️ | ⚠️ | ⚠️ | ❌ | ⚠️ | ✅ | ⚠️ | Workflow claims publish/signing, but unauthenticated manifest inspection returned `manifest unknown`. |
| `go install github.com/Mindburn-Labs/helm-oss/core/cmd/helm@latest` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | Fails because `core/go.mod` still declares the old module path. |
