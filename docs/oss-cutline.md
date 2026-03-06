---
title: OSS_CUTLINE.md
---

# OSS_CUTLINE.md
HELM OSS Cutline - helm-core-oss v0.1
Date: 2026-02-15
Status: FINAL (Release Blocking)
Scope: This document defines exactly what ships in the public `helm-public` repository, what is quarantined, and what is removed. If a package is not listed as SHIP or QUARANTINE, it must not exist in the OSS repo.

---

## 0) OSS Identity

HELM OSS is the reference implementation of the HELM Unified Canonical Standard v1.2 for a deterministic execution kernel.

- Models propose. The kernel disposes.
- HELM OSS is NOT an agent framework, not a planning system, not a self-improving system, and not an orchestrator.
- HELM OSS is a cryptographic execution firewall, ProofGraph recorder, EvidencePack exporter, and replay verifier.
- HELM OSS is designed to wrap existing agent stacks via OpenAI-compatible proxy mode and MCP gateway mode.

---

## 1) Shipped Binaries (the only supported distribution surface)

Only these commands ship and are supported:

- `core/cmd/helm-node` - API server, kernel enforcement, ProofGraph persistence
- `core/cmd/helm` - conformance, export, verify, replay tools

Anything else must be removed or quarantined.

---

## 2) Supported Use Cases (must be executable via scripts/usecases)

Minimum OSS-supported use cases:
- UC-001 PEP allows safe tool execution with ProofGraph nodes (INTENT -> ATTESTATION -> EFFECT)
- UC-002 Fail-closed schema validation (unknown fields and missing required)
- UC-003 RFC-005 approval ceremony blocks and unblocks execution
- UC-004 RFC-004 WASI pure transformer execution (no FS/net)
- UC-005 RFC-004 bounded compute terminates malicious wasm (gas/time/memory)
- UC-006 ACID idempotency prevents double execution across restart
- UC-007 EvidencePack export produces deterministic tar.gz
- UC-008 Replay verification works offline from EvidencePack
- UC-009 Connector output drift fails closed with ERR_CONNECTOR_CONTRACT_DRIFT
- UC-010 Trust key rotation remains replayable via TRUST_EVENT at Lamport height
- UC-011 Island Mode restricts cross-boundary operations during partition
- UC-012 OpenAI proxy tool loop works (only when enabled)

---

## 3) SHIP (required packages)

These packages are required and must be reachable from shipped binaries. They constitute the OSS product.

### Kernel enforcement (TCB core)
- `core/pkg/guardian` - PEP policy enforcement, intent issuance, verdict attestation
- `core/pkg/executor` - SafeExecutor, executes effects only with signed intents
- `core/pkg/boundary` - effect boundary isolation primitives (if present and used)
- `core/pkg/contracts` - signed record formats (DecisionRecord, Intent, Receipt, ApprovalReceipt)
- `core/pkg/canonicalize` - RFC 8785 JCS canonicalization
- `core/pkg/crypto` - hashing, signing, domain separation
- `core/pkg/proofgraph` - canonical node encoding, hashing, signing, verification
- `core/pkg/trust` - Trust Registry state reconstruction and TRUST_EVENT handling
- `core/pkg/audit` - structured logging and audit event emission (must not be primary truth)

### State and storage
- `core/pkg/database` - DB connections, migrations
- `core/pkg/store` - storage interfaces and Postgres implementations
- `core/pkg/ledger` - obligation/intent state persistence (if used)
- `core/pkg/receipts` - receipt storage and queries (if separate from proofgraph)
- `core/pkg/artifacts` - content-addressed artifact store used by EvidencePack

### Runtime isolation
- `core/pkg/runtime` - runtime entrypoints
- `core/pkg/runtime/sandbox` - WASI sandbox implementation
- `core/pkg/runtime/budget` - RFC-004 compute budgets and enforcement
- `core/pkg/pack` - pack resolution and wasm module loading (or artifact resolution)
- `core/pkg/manifest` - tool/connector schemas and validation logic
- `core/pkg/conform` - schema conformity helpers
- `core/pkg/conformance` - conformance harness and vectors

### Interop and API
- `core/pkg/api` - HTTP server, handlers
- `core/pkg/mcp` - MCP host/gateway support
- `core/pkg/llm` - minimal provider-neutral client surface (OpenAI-compatible mode required)
- `core/pkg/connectors` and `core/pkg/connector` - only passive connector interfaces/examples (no background triggers)
- `core/pkg/domains` - reference tools/actions (must be strictly gated by Guardian)

### Tooling needed to be a standard
- `core/pkg/verification` - verification helpers used by conformance/replay
- `core/pkg/buildguard` - supply chain verification for OSS releases
- `core/pkg/versioning` - schema/protocol version identifiers and compatibility checks
- `core/pkg/util` - safe helpers (must not add hidden side effects)
- `core/pkg/config` - configuration loading and profile selection
- `core/pkg/security` - minimal security primitives used by kernel (if present and used)
- `core/pkg/ingestion` - only if OSS ships governed ingestion; otherwise move to QUARANTINE

---

## 4) QUARANTINE (allowed in repo but MUST NOT be reachable from shipped binaries)

These packages may remain in the OSS repo only if:
- they are excluded by default build tags OR
- they are only used by tests/tools, never by `helm` or `helm-node`

Quarantine candidates:
- `core/pkg/loadtest`
- `core/pkg/quality`
- `core/pkg/observability` (allowed if purely metrics/tracing helpers and not a bypass)
- `core/pkg/console` (allowed only if it is a thin UI backend with no execution path)
- `core/pkg/sdk` (allowed if it is a generated client and not imported by kernel)
- `apps/control-room-ui` (frontend is optional but recommended for adoption)

Quarantine policy:
- Add build tags such as `//go:build helm_quarantine` for anything that might accidentally enter TCB.
- CI must enforce “no quarantine imports” from shipped binaries.

---

## 5) REMOVE (must not exist in OSS)

These packages are prohibited in OSS because they introduce stochastic behavior, self-modification, background triggers, or enterprise-only scaffolding.

### Stochastic brain and autonomous loops (prohibited)
- `core/pkg/autogenesis`
- `core/pkg/neuromodulation`
- `core/pkg/homeostasis`
- `core/pkg/swarm`
- `core/pkg/orchestration`
- `core/pkg/planning`
- `core/pkg/vpl`
- `core/pkg/heuristic`
- `core/pkg/attention`
- `core/pkg/evaluator`
- `core/pkg/fas`
- `core/pkg/watcher`
- `core/pkg/experimental`

### Enterprise/platform scaffolds not required for OSS kernel
Remove unless strictly required for OSS use cases (default: remove):
- `core/pkg/ecosystem`
- `core/pkg/portal`
- `core/pkg/ux`
- `core/pkg/blueprint`
- `core/pkg/assembly`
- `core/pkg/builder`
- `core/pkg/module`
- `core/pkg/interp` / `core/pkg/interop` (if it is agent-to-agent platform glue, not MCP)
- `core/pkg/perimeter`
- `core/pkg/reconciler`
- `core/pkg/evolution`
- `core/pkg/change`
- `core/pkg/compiler`
- `core/pkg/hierarchy`
- `core/pkg/measurabled`
- `core/pkg/infra`
- `core/pkg/ops`

NOTE: If any removed package is required by compilation, refactor to eliminate the dependency. Do not keep it “just because it compiles”.

---

## 6) TCB Policy (Release Blocking)

The following packages define the kernel TCB boundary:
- `core/pkg/guardian`
- `core/pkg/executor`
- `core/pkg/proofgraph`
- `core/pkg/trust`
- `core/pkg/canonicalize`
- `core/pkg/crypto`
- `core/pkg/runtime/sandbox`
- `core/pkg/runtime/budget`

TCB forbidden imports (must be enforced by CI linter):
- `net/http` (TCB must not be an HTTP client/server)
- `os/exec`
- direct vendor SDKs (OpenAI, AWS, GCP, Azure, Stripe, etc.)
- filesystem writes except through `core/pkg/artifacts` and deterministic export routines
- any package in QUARANTINE or REMOVE lists

---

## 7) OSS Interop Surfaces (Adoption Wedges)

OSS must support:
- MCP gateway mode (preferred)
- OpenAI-compatible proxy mode (optional, disabled by default via `HELM_ENABLE_OPENAI_PROXY=1`)

Provider neutrality:
- Support OpenAI-compatible endpoints as the universal adapter.
- Additional provider adapters may exist outside the TCB, but must never be imported by TCB packages.

---

## 8) Verification (must pass before tag)

Commands:
- `go mod tidy`
- `go test ./...`
- `go run ./tools/doccheck -root docs`
- `bash scripts/usecases/run_all.sh`
- `./core/cmd/helm/helm test conformance --level L1`
- `./core/cmd/helm/helm test conformance --level L2 --profile smb`
- `./core/cmd/helm/helm export pack <session_id> --out artifacts/.../EvidencePack.tar.gz`
- `./core/cmd/helm/helm replay verify --pack artifacts/.../EvidencePack.tar.gz`

If any command fails, the release is blocked.

---

## 9) Non-Negotiable Principle

If it is not in SHIP, it does not exist for OSS.
If it is not reachable from shipped binaries, it does not affect the kernel TCB.
If it is not reproducible via EvidencePack + replay verify, it did not happen.
