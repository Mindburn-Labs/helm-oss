# HELM Automation Lifecycle

> Maintenance runs, drift detection, incident registry, and docs freshness.

## 1. Scheduled Automation

| Task                          | Schedule     | CI Workflow               | Gate                 |
| ----------------------------- | ------------ | ------------------------- | -------------------- |
| Conformance regression        | Weekly (Mon) | `compat-matrix.yml`       | ❌ advisory          |
| Dependency vulnerability scan | Daily        | `scorecard.yml`           | ✅ block on critical |
| SDK drift check               | Every PR     | `sdk_gates.yml`           | ✅ block             |
| Schema version audit          | Every PR     | `repo-cleanup-guards.yml` | ✅ block             |
| Docs freshness check          | Weekly (Wed) | `docs-freshness.yml`      | ❌ advisory          |
| Bundle signature verification | Every PR     | `repo-cleanup-guards.yml` | ❌ advisory          |

## 2. Drift Detection

### 2.1 SDK Type Drift

Detected by `make codegen-check`: regenerates all SDK types from proto IDL
and compares against committed files.

### 2.2 Schema Drift

The `repo-cleanup-guards.yml` workflow checks:

- All schemas have version fields
- No deprecated verdict vocabulary in code
- All pack deny codes are registered

### 2.3 Spec Drift

Compare `SPEC_VERSION` against SDK version manifests:

```bash
SPEC=$(cat protocols/specs/SPEC_VERSION)
PY=$(grep version sdk/python/pyproject.toml | head -1)
TS=$(jq .version sdk/ts/package.json)
# All should share the same major.minor
```

## 3. Incident Registry

### Structure

```json
{
  "incidents": [
    {
      "id": "INC-2026-001",
      "severity": "P1",
      "title": "Receipt chain validation bypass in edge case",
      "discovered": "2026-01-15",
      "resolved": "2026-01-16",
      "affected_versions": ["0.2.x"],
      "fix_version": "0.3.0",
      "postmortem_url": "docs/incidents/INC-2026-001.md"
    }
  ]
}
```

Path: `docs/incidents/registry.json`

### Process

1. Incident discovered → file `INC-YYYY-NNN` in `docs/incidents/`
2. Triage within 4 hours (P0/P1) or 24 hours (P2+)
3. Fix committed with regression test
4. Postmortem published within 7 days
5. Registry updated

## 4. Docs Freshness

### Staleness Detection

```yaml
# docs-freshness.yml (advisory)
- name: Check doc freshness
  run: |
    find docs/ -name '*.md' -mtime +90 | while read f; do
      echo "⚠️ Stale doc (>90 days): $f"
    done
```

### Ownership Model

Each spec doc has an implicit owner:

- `EFFECT_BOUNDARY.md` → kernel team
- `AUTH_MATRIX.md` → security team
- `CLIENT_ECOSYSTEM.md` → integrations team
- `CERTIFICATION.md` → standards team

## 5. Maintenance Run Protocol

A "maintenance run" is a scheduled sweep to:

1. Update all dependency versions (`dependabot` or `renovate`)
2. Regenerate SDK types (`make codegen`)
3. Re-run full conformance suite
4. Review and close stale issues (>90 days inactive)
5. Audit security posture (`osv-scanner`, `scorecard`)
6. Update compatibility matrix
