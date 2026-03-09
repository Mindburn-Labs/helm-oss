# Contributing to HELM

HELM is a kernel. Contributions are held to kernel standards.

---

## Where to Start

| Label | What to work on |
|-------|----------------|
| **`good first issue`** | Small, well-scoped tasks with clear acceptance criteria |
| **`conformance`** | Add conformance vectors, strengthen L1/L2 gates, propose L3 items |
| **`sdk`** | Python + TypeScript client improvements, new language SDKs |
| **`docs-truth`** | Fix claims not anchored to code/tests, improve QUICKSTART/DEMO |
| **`tcb-hardening`** | Strengthen the 8-package TCB, add CI gates, close attack surfaces |

---

## Rules

1. **No alternate execution paths.** One SafeExecutor. One Guardian. One ProofGraph.
2. **No TCB expansion** without explicit justification, new gates, and review.
3. **Every feature ships with:** tests, conformance vectors, and a runnable use case.
4. **No `// TODO` or `// FIXME` in merged code.** Fix it or don't ship it.
5. **Schema changes require migration.** No silent field additions.

---

## Development

```bash
make build    # Build helm + helm-node
make test     # Unit tests (all packages)
make crucible # 12 use cases + conformance L1/L2
make lint     # go vet + static analysis
```

---

## Pull Request Process

1. Fork → branch → implement → test → PR
2. All 16 CI gates must pass (see `.github/workflows/helm_core_gates.yml`)
3. One approval required from a maintainer
4. Squash merge only

---

## Code Style

- Go: `gofmt` + `go vet` + no lint warnings
- 100-char line soft limit
- Comments explain *why*, not *what*

---

## Roadmap

See [docs/ROADMAP.md](docs/ROADMAP.md) — 10 items, no dates, each tied to a conformance level.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting.
