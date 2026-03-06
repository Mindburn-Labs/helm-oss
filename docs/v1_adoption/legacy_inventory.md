# HELM OSS v1.0 — Legacy Inventory

> Full repo scan performed 2026-03-05. Every file classified against the v1.0 Adoption Spec.

## Classification Legend

| Action    | Meaning                                                                 |
| --------- | ----------------------------------------------------------------------- |
| REWRITE   | Content exists but contradicts v1.0 spec — must be updated in place     |
| DELETE    | File is superseded, duplicate, or misleading — remove entirely          |
| MOVE      | File is useful but misplaced — relocate                                 |
| DEPRECATE | Content references a capability that exists but is not the default path |
| KEEP      | File is compliant with v1.0 spec                                        |

---

## 1. Documentation Issues

### 1.1 `.tar.gz` References (v1.0 says `.tar` is default, `.tar.gz` optional)

| File                               | Action  | Rationale                                               |
| ---------------------------------- | ------- | ------------------------------------------------------- |
| `README.md:271`                    | REWRITE | Diagram says `EvidencePack (.tar.gz)` — must say `.tar` |
| `docs/DEMO.md:58-69`               | REWRITE | All export examples use `.tar.gz` — must use `.tar`     |
| `docs/START_HERE.md:46-66`         | REWRITE | Export example + description say `.tar.gz`              |
| `docs/INTEGRATE_IN_5_MIN.md:54-63` | REWRITE | Export examples say `.tar.gz`                           |
| `docs/SECURITY_MODEL.md:78`        | REWRITE | Says sessions export as `.tar.gz`                       |
| `docs/cli_v3/FORMAT.md:59`         | REWRITE | Asset name example uses `.tar.gz`                       |

### 1.2 Version Number References (`v0.1` → `v1.0`)

| File                      | Action  | Rationale                         |
| ------------------------- | ------- | --------------------------------- |
| `README.md:283`           | REWRITE | Header says "Shipped in OSS v0.1" |
| `docs/DEPENDENCIES.md:3`  | REWRITE | Says "HELM OSS v0.1"              |
| `deploy/DEMO_HOSTED.md:3` | REWRITE | Says "HELM OSS v0.1"              |

### 1.3 Go Version References (`1.24` → `1.25`)

| File                             | Action  | Rationale                      |
| -------------------------------- | ------- | ------------------------------ |
| `docs/QUICKSTART.md:9`           | REWRITE | Says "Go 1.24+"                |
| `examples/go_client/README.md:8` | REWRITE | Says "Go 1.24+"                |
| `Dockerfile:3`                   | REWRITE | Pinned to `golang:1.24-alpine` |

### 1.4 `proxy up` Legacy Alias

| File                                        | Action  | Rationale                        |
| ------------------------------------------- | ------- | -------------------------------- |
| `docs/VERIFICATION.md:89`                   | REWRITE | Says `helm proxy up --websocket` |
| `docs/INTEGRATIONS/ORCHESTRATORS.md:50`     | REWRITE | Says `helm proxy up --websocket` |
| `docs/INTEGRATIONS/PROXY_SNIPPETS.md:52,76` | REWRITE | Says `helm proxy up --websocket` |
| `sdk/python/openai_agents/README.md:57`     | REWRITE | Says `helm proxy up`             |
| `core/cmd/helm/main.go:153`                 | REWRITE | Help text says "proxy up"        |
| `core/cmd/helm/onboard_cmd.go:106`          | REWRITE | Output says "helm proxy up"      |

### 1.5 `/v1/responses` WebSocket Endpoint Docs (valid endpoint, but docs need reconciliation)

| File                                     | Action  | Rationale                                          |
| ---------------------------------------- | ------- | -------------------------------------------------- |
| `docs/VERIFICATION.md:98-100`            | REWRITE | Shows WS endpoint with `proxy up` alias            |
| `docs/INTEGRATIONS/ORCHESTRATORS.md:52`  | REWRITE | Comment references `/v1/responses` with `proxy up` |
| `docs/INTEGRATIONS/PROXY_SNIPPETS.md:58` | REWRITE | Comment references `/v1/responses` with `proxy up` |

### 1.6 Wrong Repository URL

| File              | Action  | Rationale                                    |
| ----------------- | ------- | -------------------------------------------- |
| `docs/DEMO.md:10` | REWRITE | Clone URL says `helm.git` not `helm-oss.git` |

### 1.7 Missing Required Docs

| File                | Action  | Rationale                                                |
| ------------------- | ------- | -------------------------------------------------------- |
| `docs/v1_adoption/` | CREATE  | Phase 0 output directory                                 |
| `RELEASE.md`        | REWRITE | Manual steps; must reflect automated one-button workflow |

---

## 2. Workflow Issues

### 2.1 Release Workflow (`release.yml`)

| Issue                                           | Action  | Rationale                                                             |
| ----------------------------------------------- | ------- | --------------------------------------------------------------------- |
| No smoke test job                               | ADD     | Release must fail if 10-min WOW path fails                            |
| No golden-evidencepack.tar                      | ADD     | Must be published as release artifact                                 |
| No golden-run-report.html                       | ADD     | Must be published as release artifact                                 |
| No slim Docker image                            | ADD     | Spec requires `helm:<version>-slim` tag                               |
| No adapter npm publish                          | ADD     | Must publish `@mindburn/helm-openai-agents` + `@mindburn/helm-mastra` |
| GoReleaser extra_files missing golden artifacts | REWRITE | Must include golden-evidencepack.tar, golden-run-report.html          |
| Compat matrix not attached to GH release        | REWRITE | Artifacts generated but not uploaded to release                       |

### 2.2 Core Gates Workflow (`helm_core_gates.yml`)

| Issue                                      | Action      | Rationale           |
| ------------------------------------------ | ----------- | ------------------- |
| Duplicate SBOM jobs (sbom + sbom-generate) | CONSOLIDATE | One source of truth |

### 2.3 CI Workflow (`ci.yml`)

| Issue                       | Action | Rationale                            |
| --------------------------- | ------ | ------------------------------------ |
| Missing adapter build smoke | ADD    | CI should verify example builds work |

---

## 3. Code Issues

### 3.1 `helm-node` (apps/helm-node/)

| File                      | Issue                                   | Action                      |
| ------------------------- | --------------------------------------- | --------------------------- |
| `demo.go:370`             | `.tar.gz` content-disposition           | REWRITE to `.tar`           |
| `export_cmd.go:19,38,142` | `.tar.gz` comments and flag description | REWRITE to `.tar`           |
| `Dockerfile.api` (core/)  | Unused Dockerfile for API-only image    | KEEP if used, DELETE if not |

### 3.2 `core/cmd/helm/`

| File                             | Issue                           | Action                                      |
| -------------------------------- | ------------------------------- | ------------------------------------------- |
| `export_pack_test.go:9,36,37,70` | Test files use `.tar.gz` paths  | REWRITE                                     |
| `export_cmd.go:143`              | Fallback creates `.tar.gz`      | AUDIT — verify default path uses `.tar`     |
| `proxy_cmd.go:140`               | `proxy up` alias handler        | KEEP (backward compat) but update help text |
| `main.go:153`                    | Help text references `proxy up` | REWRITE                                     |
| `onboard_cmd.go:106`             | Output text says `proxy up`     | REWRITE                                     |

---

## 4. Files Classified as KEEP (Compliant)

| File/Area                                                 | Status                               |
| --------------------------------------------------------- | ------------------------------------ |
| `docs/QUICKSTART.md` (except Go version)                  | KEEP (minor fix needed)              |
| `docs/VERIFICATION.md` (except proxy up)                  | KEEP (minor fix needed)              |
| `docs/COMPATIBILITY.md`                                   | KEEP — tiers correct                 |
| `docs/TROUBLESHOOTING.md`                                 | KEEP                                 |
| `docs/INTEGRATIONS/MCP_CLIENTS.md`                        | KEEP                                 |
| `docs/INTEGRATIONS/SANDBOXES.md`                          | KEEP                                 |
| `docs/INTEGRATIONS/ORCHESTRATORS.md` (except proxy up)    | KEEP (minor fix needed)              |
| `.github/workflows/compatibility_matrix.yml`              | KEEP                                 |
| `.github/workflows/helm_core_gates.yml` (except dup SBOM) | KEEP                                 |
| `.github/workflows/ci.yml`                                | KEEP (add adapter gate)              |
| `.goreleaser.yml` (except extra_files)                    | KEEP                                 |
| `Makefile`                                                | KEEP                                 |
| `docs/CONFORMANCE.md`                                     | KEEP                                 |
| `docs/SECURITY_MODEL.md` (except tar.gz)                  | KEEP                                 |
| `docs/THREAT_MODEL.md`                                    | KEEP                                 |
| `docs/TCB_POLICY.md`                                      | KEEP                                 |
| `docs/RESPONSES_API.md`                                   | KEEP — correctly states no migration |
| `install.sh`                                              | KEEP                                 |
| `Dockerfile` (except Go pin)                              | KEEP                                 |
| `Dockerfile.slim`                                         | KEEP                                 |
| All `core/pkg/` source                                    | KEEP                                 |
| All SDK source                                            | KEEP                                 |
| All examples                                              | KEEP                                 |
