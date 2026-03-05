# CI Baselines

These files define explicit, reviewable exceptions for repo audit gates.

Rules:
- Baselines are deny-by-default for regressions: new findings not present in a baseline fail CI.
- Baselines are monotonic-down only: if a finding disappears, remove it from the baseline in the same PR.
- Do not add entries without linking the mitigation plan in the PR description.

Files:
- `no_test_packages.allowlist`: `core/pkg` packages currently exempt from direct unit tests.
- `coverage_floor_exceptions.allowlist`: packages below the 30% coverage floor.
- `interface_drift.allowlist`: heuristic interface drift baseline (`Name|File`).
- `untested_routes.allowlist`: currently untested discovered API route paths.

Update workflow:
1. Fix findings first.
2. Regenerate or edit the relevant baseline file only when justified.
3. Include rationale and sunset plan in PR notes.
