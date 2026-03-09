# HELM OSS v1.0 — Migration Plan

> Ordered changes to reach v1.0 compliance without regressions.
> Each step is independent-safe: can be applied/tested in order.

---

## Step 1: Fix Documentation (zero-risk, text-only)

**Goal**: Eliminate all legacy text contradictions.

### 1a: `.tar.gz` → `.tar` (default format)

- `README.md:271` — diagram line
- `docs/DEMO.md:58-69` — export examples
- `docs/START_HERE.md:46-66` — export example + description
- `docs/INTEGRATE_IN_5_MIN.md:54-63` — export examples
- `docs/SECURITY_MODEL.md:78` — description
- `docs/cli_v3/FORMAT.md:59` — asset name

### 1b: Version `v0.1` → `v1.0`

- `README.md:283` — table header
- `docs/DEPENDENCIES.md:3` — intro
- `deploy/DEMO_HOSTED.md:3` — intro

### 1c: Go version `1.24` → `1.25`

- `docs/QUICKSTART.md:9` — prerequisites
- `examples/go_client/README.md:8` — prerequisites

### 1d: `proxy up` → `proxy` (canonical form)

- `docs/VERIFICATION.md:89` — command example
- `docs/INTEGRATIONS/ORCHESTRATORS.md:50` — command
- `docs/INTEGRATIONS/PROXY_SNIPPETS.md:52,76` — commands
- `sdk/python/openai_agents/README.md:57` — command

### 1e: Fix repo URL

- `docs/DEMO.md:10` — `helm.git` → `helm-oss.git`

### 1f: Update RELEASE.md

- Rewrite to reflect CI-automated release (no manual steps)
- Reference exact tag workflow

---

## Step 2: Fix Code (low-risk, backward-compatible)

### 2a: CLI help text

- `core/cmd/helm/main.go:153` — help text references `proxy up`
- `core/cmd/helm/onboard_cmd.go:106` — onboard output says `proxy up`

### 2b: helm-node export

- `apps/helm-node/demo.go:370` — `.tar.gz` content-disposition
- `apps/helm-node/export_cmd.go:19,38,142` — `.tar.gz` comments/flag

### 2c: Fix test files

- `core/cmd/helm/export_pack_test.go` — `.tar.gz` → `.tar`

### 2d: Fix Dockerfile

- `Dockerfile:3` — update Go version pin to 1.25

---

## Step 3: Add Release Smoke Test (new script)

Create `scripts/ci/smoke_test.sh`:

1. Create temp dirs
2. `helm onboard --yes --data-dir $TMP`
3. `helm demo company --template starter --provider mock --out $TMP`
4. Assert `evidence.tar` + `run-report.html` + receipts exist
5. `helm verify evidence.tar`
6. Validate HTML report markers
7. Generate `.mcpb` and validate manifest
8. Generate compatibility snapshot
9. Enforce wall-clock limit (10 min)
10. Upload artifacts on failure

---

## Step 4: Update Release Workflow

### 4a: Add smoke test job (hard gate before release)

### 4b: Add golden artifact generation (evidencepack.tar + run-report.html)

### 4c: Add slim Docker image build + push

### 4d: Add adapter npm publish (openai-agents, mastra)

### 4e: Add golden artifacts to GoReleaser extra_files

### 4f: Add compatibility snapshot attachment to GH release

---

## Step 5: Consolidate CI Workflows

### 5a: Merge duplicate SBOM jobs in `helm_core_gates.yml`

### 5b: Add adapter unit test gates to ci.yml (TS + Python adapters)

---

## Step 6: Final Audit

### 6a: Grep guardrails

- No references to `.tar.gz` as default
- No references to `proxy up` in primary docs
- No references to `v0.1` in spec-facing docs
- No references to Go 1.24 in prerequisites

### 6b: Produce `docs/v1_adoption/final_compliance_report.md`
