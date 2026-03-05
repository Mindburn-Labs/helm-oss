# Zero Debt Plan

## Goal
Reach measurable zero operational debt for current tracked categories, while keeping deploy gates stable.

## Baseline (2026-03-05)
- Coverage debt: 40 packages below 30% threshold.
- Structured logging debt: 388 raw print/log calls in production paths.
- Interface drift (heuristic): 25 flagged interfaces.
- Route coverage debt (heuristic): 52 routes without test references.
- Schema drift: 1 unreferenced schema.
- Test orphan debt: 1 orphan test file.
- Tooling debt: local govulncheck runtime mismatch; shellcheck missing.

## Workstreams

### WS1: Tooling Normalization
- Pin Go toolchain in CI and local bootstrap scripts to 1.24.x.
- Fix govulncheck execution path and add explicit failure handling for runtime/tool errors.
- Add shellcheck to CI image and enforce shell lint on `scripts/`.
- Exit criteria: `govulncheck` and `shellcheck` run successfully in CI and local dev bootstrap.

### WS2: Coverage Burn-Down
- Tier packages by risk:
  - Tier A: auth, governance, guardian, executor, runtime, console APIs.
  - Tier B: registry, receipts, policyloader, budget, tenants.
  - Tier C: low-risk utility and wrappers.
- Add package-level coverage targets with ratcheting:
  - Milestone 1: all Tier A >= 35%.
  - Milestone 2: all Tier A >= 50%, Tier B >= 30%.
  - Milestone 3: all non-generated packages >= 30%.
- Exit criteria: no package below configured floor.

### WS3: Structured Logging Migration
- Replace raw `fmt.Print*` / `log.Print*` in production paths with `slog`.
- Keep CLI user-facing output on explicit `io.Writer` channels (stdout/stderr), not mixed with internal logs.
- Introduce logger injection for packages still using global logger access.
- Exit criteria: zero raw logging violations in production scope.

### WS4: API Coverage and Contract Drift
- Generate route inventory from mux registrations and map each route to at least one test.
- Add table-driven smoke tests for uncovered API groups.
- Resolve interface drift:
  - Mark intentionally abstract interfaces with explicit annotations.
  - Add concrete adapters or remove dead interfaces.
- Resolve schema/test drift:
  - Wire unreferenced schema into validator path or remove it.
  - Migrate or delete orphan test file.
- Exit criteria: route coverage debt = 0, interface/schema/test drift = 0.

### WS5: Audit and Hardening
- Enforce non-root runtime users in all Dockerfiles (done).
- Keep doc-link integrity strict (done).
- Convert debt counters back to warning/fail mode per completed workstream.
- Exit criteria: repo audit remains compliant with strict thresholds enabled.

## Execution Order
1. WS1 Tooling normalization.
2. WS2 Tier A coverage + WS3 logging for Tier A components.
3. WS4 route/interface/schema/test cleanup.
4. WS2/WS3 remaining tiers.
5. WS5 strict-threshold re-enable and final hardening pass.

## Definition of Done
- No debt counters remaining in audit output.
- No warning downgrades needed to keep compliance.
- CI runs fully green with strict checks enabled.
