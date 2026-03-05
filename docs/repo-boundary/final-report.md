# Repo Boundary Canonicalization — Final Report

**Date:** 2026-03-05  
**Branches:** `chore/canonical-oss-boundary` (both repos)

## Summary

Established `helm-public` (OSS) as the single canonical source of truth for all kernel/authority code. Commercial (`helm`) now consumes these packages via a sync script—no independent implementations remain.

### What was done

| Action                      | Scope                                                                                                                                                                                                    |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Upstream ported**         | 377 files from commercial → OSS (kernel, contracts, crypto, evidencepack, proofgraph, receipts, verifier, connectors/sandbox, conformance, incubator/audit, integrations, api, trust/registry, guardian) |
| **Protocols ported**        | 216 JSON schemas + proto definitions + specs                                                                                                                                                             |
| **SDK adapters ported**     | LangChain (Python), Mastra + OpenAI Agents (TypeScript)                                                                                                                                                  |
| **Transitive deps ported**  | orgdna/types, sandbox, store/objstore, knowledge/graph, policy, integrations/capgraph, integrations/manifest                                                                                             |
| **External deps added**     | `go-sqlite3`, `minio-go` to OSS `go.mod`                                                                                                                                                                 |
| **Sync script created**     | `tools/sync-oss-kernel.sh` — rsync of 18 protected paths from OSS → commercial                                                                                                                           |
| **Drift guardrail created** | `tools/verify-boundary.sh` — fails on any diff across 18 protected paths                                                                                                                                 |
| **Makefile targets added**  | `make verify-boundary` (both repos), `make sync-oss-kernel` (commercial)                                                                                                                                 |

### Protected Paths (18)

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

## Dependency Approach

**Strategy:** File-sync via `tools/sync-oss-kernel.sh` (rsync-based).

**Rationale:** Both repos declare `module github.com/Mindburn-Labs/helm/core` — the same Go module path. This makes Go-level `replace` directives self-referential (they can't replace a module with itself). Git subtree/submodule would add complexity without benefit since the sync script achieves the same result with zero build system changes.

**Workflow:**

1. Develop kernel/authority code in OSS (helm-public)
2. Run `make sync-oss-kernel` in commercial before building
3. CI runs `make verify-boundary` to fail-fast on drift

## Verification Results

### OSS (helm-public)

- **Build:** `go build ./...` ✅ clean
- **Tests:** All protected-path tests pass (kernel 6 pkgs, contracts 3, crypto 9, proofgraph 3, receipts 2, verifier 1)
- **Known:** `kernel/cpi` fails at link time — requires Rust `helm-policy-vm` library (pre-existing, not a regression)

### Commercial (helm)

- **Build:** `go build ./...` ✅ clean after sync
- **Drift guardrail:** `verify-boundary.sh` ✅ — all 18 paths pass

## Commands to Reproduce

```bash
# Verify boundary sync
make verify-boundary                    # either repo

# Sync OSS → commercial (from commercial repo root)
make sync-oss-kernel

# Build both
cd ../helm-public/core && go build ./...
cd ../helm/core && go build ./...

# Test protected paths (OSS)
cd ../helm-public/core && go test ./pkg/kernel/... ./pkg/contracts/... \
  ./pkg/crypto/... ./pkg/evidencepack/... ./pkg/proofgraph/... \
  ./pkg/receipts/... ./pkg/verifier/...
```

## Risks and Follow-ups

| Risk                                                                       | Mitigation                                                                     |
| -------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Sync script must be run manually                                           | Wire `make sync-oss-kernel` as a pre-build step in commercial CI               |
| `kernel/cpi` CGo bridge needs Rust library                                 | Build `crates/helm-policy-vm` in CI; not a boundary issue                      |
| New packages added to commercial may create implicit authority duplication | Review new `core/pkg/` additions in PRs; update protected paths list as needed |
| `go.mod` / `go.sum` may drift between repos                                | `go mod tidy` after sync; consider syncing `go.mod` too                        |

## HEADs at Work Start

- Commercial: `19b0c2e0b47767b8baca8434488f09ba40f51127`
- OSS: `14228017df5adbb522095e154c49774360b3077a`
