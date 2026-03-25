---
title: DEPENDENCIES
---

# Dependencies

HELM OSS v1.0 dependency justification. Prefer standard library. External deps are pinned.

## Runtime Dependencies

| Module                          | Version | Justification                                                                                                               |
| ------------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------- |
| `github.com/jackc/pgx/v5`       | v5.x    | PostgreSQL driver. Required for ACID determinism, Lamport clocks, and budget locking. Industry-standard Go Postgres driver. |
| `github.com/tetratelabs/wazero` | v1.x    | WASM runtime. RFC-004 mandates deny-by-default sandboxed execution. wazero is the only pure-Go WASM runtime (no CGO).       |

## Standard Library Usage (TCB)

TCB packages restrict imports to:

- `crypto/*` — Ed25519 signing, SHA-256 hashing
- `encoding/json` — JCS canonicalization
- `fmt`, `errors` — Error formatting
- `sort` — Deterministic key ordering
- `sync` — Mutex for in-memory stores
- `time` — Lamport timestamps, budget enforcement
- `context` — Cancellation propagation
- `bytes`, `strings` — Buffer operations
- `math` — Numeric operations

## Forbidden in TCB

These packages must NOT be imported by kernel TCB packages:

- `net/http` — Only allowed in `api/` handlers (non-TCB)
- `os/exec` — Prohibited everywhere
- `syscall` — Prohibited except explicit minimal use
- `reflect` — Prohibited in hot path (breaks determinism guarantees)
- Vendor SDKs (OpenAI, Anthropic, etc.) — Prohibited in kernel

## Version Pinning

All dependencies are pinned in `go.mod` via `go.sum` checksums.
No floating versions. No `latest` references.
