# HELM OSS Repo Audit v2

> Full file-by-file audit of `Mindburn-Labs/helm-oss`. Every file categorized.
> **No implementation begins before this audit is reviewed and accepted.**

## 🚨 Critical Security Findings

| File            | Severity     | Issue                                                |
| --------------- | ------------ | ---------------------------------------------------- |
| `data/root.key` | **CRITICAL** | Private key committed to public repo                 |
| `data/root.pub` | HIGH         | Public key material in repo (should be in config/CI) |
| `data/helm.db`  | HIGH         | Database file committed — may contain secrets/state  |
| `data/keys/`    | HIGH         | Key material directory in public repo                |

## 🧹 Bloat: 367MB of Generated Artifacts

| Path                                       | Size  | Action                                 |
| ------------------------------------------ | ----- | -------------------------------------- |
| `helm` (root binary)                       | 40MB  | **DELETE** — compiled binary in source |
| `helm-node` (root binary)                  | 39MB  | **DELETE** — compiled binary in source |
| `bin/helm`                                 | ~40MB | **DELETE** — compiled binary           |
| `bin/helm-node`                            | ~40MB | **DELETE** — compiled binary           |
| `bin/helm-linux-amd64`                     | ~40MB | **DELETE** — cross-compiled binary     |
| `bin/helm-node-linux-amd64`                | ~40MB | **DELETE** — cross-compiled binary     |
| `packages/mindburn-helm-cli/node_modules/` | 91MB  | **DELETE** — committed node_modules    |
| `packages/helm-lab-runner/node_modules/`   | 109MB | **DELETE** — committed node_modules    |
| `sdk/ts/node_modules/`                     | 87MB  | **DELETE** — committed node_modules    |

**Total removable: ~530MB** of files that should never be in version control.

---

## Directory-Level Audit

### Root Files

| File                      | Action     | Reason                                                                          |
| ------------------------- | ---------- | ------------------------------------------------------------------------------- |
| `.env.demo`               | KEEP       | Demo configuration                                                              |
| `.env.example`            | KEEP       | Configuration template                                                          |
| `.gitignore`              | REWRITE    | Must add `bin/`, `helm`, `helm-node`, `node_modules`, `data/*.key`, `data/*.db` |
| `.golangci.yml`           | KEEP       | Linter config                                                                   |
| `.goreleaser.yml`         | KEEP       | Release automation                                                              |
| `CHANGELOG.md`            | KEEP       | Release history                                                                 |
| `CONTRIBUTING.md`         | KEEP       | Contribution guide                                                              |
| `Dockerfile`              | KEEP       | Primary container build                                                         |
| `Dockerfile.slim`         | KEEP       | Minimal container build                                                         |
| `LICENSE`                 | KEEP       | BUSL-1.1                                                                        |
| `Makefile`                | REWRITE    | Audit targets for obsolete commands                                             |
| `README.md`               | REWRITE    | Must reflect standard identity, not project/product pitch                       |
| `RELEASE.md`              | REWRITE    | Must match actual release process                                               |
| `RELEASE_NOTES.md`        | KEEP       | Current release notes                                                           |
| `SECURITY.md`             | KEEP       | Security policy                                                                 |
| `docker-compose.yml`      | KEEP       | Dev orchestration                                                               |
| `docker-compose.demo.yml` | KEEP       | Demo orchestration                                                              |
| `go.work`                 | KEEP       | Go workspace                                                                    |
| `go.work.sum`             | KEEP       | Go workspace checksums                                                          |
| `helm`                    | **DELETE** | 40MB compiled binary — must not be in source                                    |
| `helm-node`               | **DELETE** | 39MB compiled binary — must not be in source                                    |
| `install.sh`              | KEEP       | Installer (already hardened with checksums)                                     |

### `core/` (880 files)

| Area                      | Files | Action         | Notes                                                   |
| ------------------------- | ----- | -------------- | ------------------------------------------------------- |
| `core/pkg/guardian/`      | ~15   | KEEP           | Recently unified verdict vocabulary                     |
| `core/pkg/contracts/`     | ~20   | KEEP           | Canonical verdict + receipt types                       |
| `core/pkg/proofgraph/`    | ~25   | KEEP           | ProofGraph + anchor + AIGP                              |
| `core/pkg/governance/`    | ~45   | KEEP           | Governance subsystem + jurisdiction + bundles           |
| `core/pkg/prg/`           | ~10   | KEEP           | Policy Requirement Graph                                |
| `core/pkg/finance/`       | ~10   | KEEP           | Budget tracking                                         |
| `core/pkg/artifacts/`     | ~8    | KEEP           | Artifact registry                                       |
| `core/pkg/executor/`      | ~6    | KEEP           | Execution engine                                        |
| `core/pkg/verifier/`      | ~5    | KEEP           | Receipt verification                                    |
| `core/pkg/crypto/`        | ~10   | KEEP           | Cryptographic primitives                                |
| `core/pkg/trust/`         | ~8    | KEEP           | Trust framework                                         |
| `core/pkg/api/`           | ~12   | REWRITE        | OpenAI proxy rewritten, but other handlers need audit   |
| `core/pkg/console/`       | 31    | REWRITE        | WebSocket console server — large, unclear scope for OSS |
| `core/pkg/incubator/`     | 35    | MOVE/DEPRECATE | Incubator = non-standard experimental code              |
| `core/pkg/orgdna/`        | 8     | KEEP           | Organization DNA runtime                                |
| `core/pkg/compliance/`    | ~15   | REWRITE        | Should become bundle-driven, not compiled               |
| `core/pkg/conform/`       | ~10   | KEEP           | Conformance framework                                   |
| `core/pkg/bridge/`        | ~5    | KEEP           | Kernel bridge                                           |
| `core/pkg/pack/`          | ~8    | KEEP           | Pack system                                             |
| `core/pkg/credentials/`   | ~5    | KEEP           | Credential management                                   |
| `core/pkg/buildguard/`    | ~3    | KEEP           | Build verification                                      |
| `core/pkg/observability/` | ~5    | KEEP           | OpenTelemetry integration                               |

### `protocols/` (238 files)

| Area                              | Action           | Notes                      |
| --------------------------------- | ---------------- | -------------------------- |
| `protocols/specs/rfc/`            | KEEP             | Receipt RFC finalized      |
| `protocols/specs/tla/`            | KEEP             | TLA+ specifications        |
| `protocols/json-schemas/`         | KEEP             | 124 JSON schemas (indexed) |
| `protocols/policy-schema/v1/`     | KEEP             | 17 proto files             |
| `protocols/proto/helm/kernel/v1/` | KEEP             | New core proto IDL         |
| `protocols/conformance/`          | KEEP             | Conformance vectors        |
| `protocols/bundles/`              | KEEP (if exists) | Policy bundle definitions  |

### `schemas/` (root-level, 4 files)

| File                                         | Action | Reason                               |
| -------------------------------------------- | ------ | ------------------------------------ |
| `schemas/compatibility-registry.schema.json` | MOVE   | Belongs in `protocols/json-schemas/` |
| `schemas/lab_mission.schema.json`            | MOVE   | Belongs in `protocols/json-schemas/` |
| `schemas/lab_receipts.schema.json`           | MOVE   | Belongs in `protocols/json-schemas/` |
| `schemas/orgdna.schema.json`                 | MOVE   | Belongs in `protocols/json-schemas/` |

**Action: DELETE `schemas/` dir after moving files.**

### `sdk/` (5 language SDKs)

| SDK           | Action  | Notes                               |
| ------------- | ------- | ----------------------------------- |
| `sdk/go/`     | KEEP    | Go SDK                              |
| `sdk/ts/`     | REWRITE | Has committed `node_modules` (87MB) |
| `sdk/python/` | KEEP    | Python SDK                          |
| `sdk/java/`   | KEEP    | Java SDK                            |
| `sdk/rust/`   | KEEP    | Rust SDK                            |

### `packages/` (2 packages)

| Package                       | Action  | Notes                                |
| ----------------------------- | ------- | ------------------------------------ |
| `packages/mindburn-helm-cli/` | REWRITE | Has committed `node_modules` (91MB)  |
| `packages/helm-lab-runner/`   | REWRITE | Has committed `node_modules` (109MB) |

### `apps/` (1 subdir)

| Path              | Action | Notes                                                   |
| ----------------- | ------ | ------------------------------------------------------- |
| `apps/helm-node/` | MOVE   | Should be `tools/helm-node/` or merged into main binary |

### `bin/` (4 compiled binaries)

| File                        | Action     | Reason                          |
| --------------------------- | ---------- | ------------------------------- |
| `bin/helm`                  | **DELETE** | Compiled binary in source       |
| `bin/helm-node`             | **DELETE** | Compiled binary in source       |
| `bin/helm-linux-amd64`      | **DELETE** | Cross-compiled binary in source |
| `bin/helm-node-linux-amd64` | **DELETE** | Cross-compiled binary in source |

### `data/`

| File              | Action     | Reason                        |
| ----------------- | ---------- | ----------------------------- |
| `data/root.key`   | **DELETE** | 🚨 Private key in public repo |
| `data/root.pub`   | MOVE       | To CI/config, not source      |
| `data/helm.db`    | **DELETE** | Database state in source      |
| `data/keys/`      | **DELETE** | Key material in source        |
| `data/artifacts/` | KEEP       | Sample artifacts for testing  |
| `data/evidence/`  | KEEP       | Sample evidence               |

### `docs/` (75 files)

| File                      | Action    | Notes                                            |
| ------------------------- | --------- | ------------------------------------------------ |
| `docs/GOVERNANCE_SPEC.md` | KEEP      | New normative spec                               |
| `docs/ROADMAP.md`         | REWRITE   | Must reflect master scope, not aspirational list |
| `docs/v1_adoption/`       | DEPRECATE | Stale adoption content                           |
| `docs/cli_v3/`            | KEEP      | CLI docs                                         |
| `docs/use-cases/`         | KEEP      | Use case documentation                           |
| `docs/INTEGRATIONS/`      | REWRITE   | Must match actual integration status             |
| Other docs                | KEEP      | Most are useful                                  |

### `tools/` (13 files)

| Tool                       | Action | Notes                        |
| -------------------------- | ------ | ---------------------------- |
| `tools/boundary/`          | KEEP   | Boundary verification        |
| `tools/cedar-pdp/`         | KEEP   | Cedar PDP integration        |
| `tools/dispute-viewer/`    | KEEP   | Dispute viewing tool         |
| `tools/doccheck/`          | KEEP   | Documentation checker        |
| `tools/replay-ui/`         | KEEP   | Replay viewer                |
| `tools/tcbcheck/`          | KEEP   | TCB verification             |
| `tools/verify-boundary.sh` | KEEP   | Boundary verification script |

### `scripts/` (29 files)

| Area                           | Action | Notes                   |
| ------------------------------ | ------ | ----------------------- |
| `scripts/ci/` (17 files)       | KEEP   | CI scripts              |
| `scripts/release/` (4 files)   | KEEP   | Release automation      |
| `scripts/demo/` (3 files)      | KEEP   | Demo scripts            |
| `scripts/deploy_demo.sh`       | KEEP   | Demo deployment         |
| `scripts/generate-fixture.mjs` | KEEP   | Test fixture generation |
| `scripts/sdk/`                 | KEEP   | SDK build scripts       |
| `scripts/usecases/`            | KEEP   | Use case scripts        |
| `scripts/bench/`               | KEEP   | Benchmarking            |

### `deploy/`

| File                 | Action | Notes                 |
| -------------------- | ------ | --------------------- |
| `deploy/Caddyfile`   | KEEP   | Reverse proxy config  |
| `deploy/grafana/`    | KEEP   | Monitoring dashboards |
| `deploy/prometheus/` | KEEP   | Metrics config        |

### `examples/` (35 files)

| Area                       | Action | Notes                     |
| -------------------------- | ------ | ------------------------- |
| `examples/golden/`         | KEEP   | Golden test fixtures      |
| Language-specific examples | KEEP   | Reference implementations |
| `examples/README.md`       | KEEP   | Index                     |

### `fixtures/` (4 files)

| Path                | Action | Notes                 |
| ------------------- | ------ | --------------------- |
| `fixtures/minimal/` | KEEP   | Minimal test fixtures |

### `.github/`

| File                                  | Action | Notes                 |
| ------------------------------------- | ------ | --------------------- |
| `workflows/ci.yml`                    | KEEP   | Main CI               |
| `workflows/release.yml`               | KEEP   | Release workflow      |
| `workflows/helm_core_gates.yml`       | KEEP   | Core gate checks      |
| `workflows/sdk_gates.yml`             | KEEP   | SDK gate checks       |
| `workflows/tla-check.yml`             | KEEP   | TLA+ CI               |
| `workflows/compatibility_matrix.yml`  | KEEP   | Compatibility testing |
| `workflows/boundary-verification.yml` | KEEP   | Boundary verification |
| `workflows/scorecard.yml`             | KEEP   | Security scorecard    |
| `actions/`                            | KEEP   | Reusable actions      |

### `api/`

| File           | Action | Notes         |
| -------------- | ------ | ------------- |
| `api/openapi/` | KEEP   | OpenAPI specs |

### `qa/`

| File        | Action | Notes      |
| ----------- | ------ | ---------- |
| `qa/tools/` | KEEP   | QA tooling |

---

## Summary Statistics

| Category      | Count                            |
| ------------- | -------------------------------- |
| **DELETE**    | ~15 files/dirs (530MB+ of bloat) |
| **REWRITE**   | ~12 files                        |
| **MOVE**      | ~6 files                         |
| **DEPRECATE** | ~3 dirs                          |
| **KEEP**      | ~900+ files                      |
