# HELM Repo Separation — Final Report

**Date:** 2026-03-05
**OSS HEAD:** `03897f37ac9fb163d26a1204771a3ecfad6b5186`
**Commercial HEAD:** `4fd632bf21b2d6f94ad1f8b2ef1daee8b4bdf14e`

---

## 1. End State Summary

### OSS (helm-public / helm-oss) — Single Canonical Source of Truth

All kernel/authority code lives exclusively here:

| Component                                       | Path                           |
| ----------------------------------------------- | ------------------------------ |
| Kernel execution boundary                       | `core/pkg/kernel/`             |
| Contracts + schemas                             | `core/pkg/contracts/`          |
| Cryptographic authority                         | `core/pkg/crypto/`             |
| EvidencePack/export/verify                      | `core/pkg/evidencepack/`       |
| ProofGraph semantics                            | `core/pkg/proofgraph/`         |
| Receipts                                        | `core/pkg/receipts/`           |
| Verifier tooling                                | `core/pkg/verifier/`           |
| Sandbox governance adapters                     | `core/pkg/connectors/sandbox/` |
| Conformance suite                               | `core/pkg/conformance/`        |
| Audit subsystem                                 | `core/pkg/incubator/audit/`    |
| Integrations (receipts/capgraph/manifest)       | `core/pkg/integrations/`       |
| API authority handlers                          | `core/pkg/api/`                |
| Trust registry                                  | `core/pkg/trust/registry/`     |
| Guardian                                        | `core/pkg/guardian/`           |
| Protocol schemas + specs                        | `protocols/`                   |
| JSON schemas                                    | `schemas/`                     |
| SDK adapters (LangChain, Mastra, OpenAI Agents) | `sdk/`                         |

### Commercial (helm) — Product + Monetization Only

Contains: Studio UI, enterprise features (SSO/RBAC/SCIM), premium packs, marketplace, deployment targets. **No independent kernel/authority implementations.**

Protected paths in commercial are populated exclusively by the deterministic sync mechanism from OSS.

---

## 2. Canonical protected.manifest

Located at: `tools/boundary/protected.manifest` (in OSS)

**Format:** `SHA256  PATH` — one line per protected file.

**Stats:** 554 files across 18 protected path groups.

**Regenerate:** `bash tools/boundary/generate-manifest.sh`

**Protected path groups:**

```
core/pkg/kernel           core/pkg/receipts
core/pkg/contracts        core/pkg/verifier
core/pkg/crypto           core/pkg/connectors/sandbox
core/pkg/evidencepack     core/pkg/conformance
core/pkg/proofgraph       core/pkg/incubator/audit
core/pkg/api              core/pkg/integrations/receipts
core/pkg/trust/registry   core/pkg/integrations/capgraph
core/pkg/guardian         core/pkg/integrations/manifest
protocols                 schemas
```

---

## 3. How to Bump OSS_COMMIT and Run Sync

```bash
# 1. In OSS repo — make changes to kernel/authority code, commit
cd helm-public
# ... edit, test, commit ...
bash tools/boundary/generate-manifest.sh   # regenerate manifest
git add -A && git commit -m "feat: update manifest after kernel changes"

# 2. In commercial repo — update the pinned commit
cd helm
# Edit tools/oss.lock → set OSS_COMMIT to the new OSS HEAD SHA
vim tools/oss.lock

# 3. Run deterministic sync
bash tools/sync-oss-kernel.sh ../helm-public
# This will:
#   - Use git-archive at the pinned commit (NOT working tree)
#   - Extract protected paths, delete extraneous files
#   - Update oss.lock timestamp and manifest hash
#   - Run verify-boundary.sh automatically

# 4. Commit with [OSS-SYNC] tag (required by CI)
git add -A && git commit -m "chore: bump OSS kernel to <SHA> [OSS-SYNC]"
```

---

## 4. Verification Commands

### OSS (helm-public)

```bash
# Format check
gofmt -l core/            # should output nothing

# Build
cd core && go build ./...

# Test protected paths
cd core && go test -count=1 \
  ./pkg/kernel/... \
  ./pkg/contracts/... \
  ./pkg/crypto/... \
  ./pkg/evidencepack/... \
  ./pkg/proofgraph/... \
  ./pkg/receipts/... \
  ./pkg/verifier/...

# Manifest self-check
bash tools/verify-boundary.sh

# Conformance suite
cd core && go test -v ./pkg/conformance/... -count=1
```

### Commercial (helm)

```bash
# Sync from OSS (if needed)
bash tools/sync-oss-kernel.sh ../helm-public

# Verify boundary integrity (SHA256)
bash tools/verify-boundary.sh

# Build
cd core && go build ./...

# Test
cd core && go test ./pkg/... -count=1

# Makefile shortcuts
make verify-boundary
make sync-oss-kernel
```

---

## 5. CI Guardrails Summary

### OSS — `.github/workflows/boundary-verification.yml`

| Job               | Purpose                                                        |
| ----------------- | -------------------------------------------------------------- |
| `verify-boundary` | Runs `tools/verify-boundary.sh` — validates manifest integrity |
| `build-kernel`    | `go build ./...`                                               |
| `test-kernel`     | Tests all 18 protected-path packages                           |
| `conformance`     | Runs conformance suite                                         |

**Triggers:** push and PR to `main`.

### Commercial — `.github/workflows/boundary-enforcement.yml`

| Job                    | Purpose                                                                                |
| ---------------------- | -------------------------------------------------------------------------------------- |
| `protected-path-guard` | **Blocks PRs** that modify protected paths unless commit message contains `[OSS-SYNC]` |
| `verify-boundary`      | SHA256 manifest verification via `tools/verify-boundary.sh`                            |
| `build`                | `go build ./...` (runs after boundary verification passes)                             |

**Triggers:** push and PR to `main`.

---

## 6. Known Constraints

| Constraint              | Details                                                                                                               | How to Test                                                                                                               |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `kernel/cpi` CGo bridge | Requires Rust `helm-policy-vm` compiled library at `crates/helm-policy-vm/target/release/`                            | Build Rust lib first: `cd crates/helm-policy-vm && cargo build --release`, then `cd core && go test ./pkg/kernel/cpi/...` |
| Same Go module path     | Both repos declare `module github.com/Mindburn-Labs/helm/core` — cannot use Go `replace` directive (self-referential) | Solved via file-level git-archive sync pinned to exact commit                                                             |
| Manifest staleness      | If OSS files change without regenerating manifest, verify-boundary detects staleness but won't fail in OSS            | Always run `tools/boundary/generate-manifest.sh` after changing protected files                                           |

---

## Tools Inventory

| File                                          | Repo       | Purpose                                            |
| --------------------------------------------- | ---------- | -------------------------------------------------- |
| `tools/boundary/protected.manifest`           | OSS        | Canonical SHA256 file list (554 files)             |
| `tools/boundary/generate-manifest.sh`         | OSS        | Regenerates manifest from current tree             |
| `tools/verify-boundary.sh`                    | Both       | SHA256-based drift detection                       |
| `tools/oss.lock`                              | Commercial | Pins to specific OSS commit                        |
| `tools/sync-oss-kernel.sh`                    | Commercial | Deterministic git-archive sync                     |
| `.github/workflows/boundary-verification.yml` | OSS        | CI: manifest + kernel tests + conformance          |
| `.github/workflows/boundary-enforcement.yml`  | Commercial | CI: protected-path guard + boundary verify + build |
