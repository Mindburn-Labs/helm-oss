---
title: VERSION_SYNC_POLICY
---

# HELM SDK Version Sync Policy

> How SDK versions, proto IDL, and spec versions stay in sync.

## Version Relationships

```
SPEC_VERSION (1.0.0-alpha.1)
     │
     ├── Proto IDL (helm.proto) — version field in package
     │
     ├── Go SDK (sdk/go) — go.mod module version
     ├── Python SDK (sdk/python) — pyproject.toml version
     ├── TypeScript SDK (sdk/ts) — package.json version
     ├── Java SDK (sdk/java) — pom.xml version
     └── Rust SDK (sdk/rust) — Cargo.toml version
```

## Rules

### 1. Spec Version is Authoritative

The spec version in `protocols/specs/SPEC_VERSION` is the single source of truth.
All other version references derive from it.

### 2. SDK Major.Minor Tracks Spec

SDK versions MUST track the spec's major.minor:

- Spec `1.0.0` → SDK `1.0.x`
- Spec `1.1.0` → SDK `1.1.x`
- SDK patch versions are independent (bug fixes, performance)

### 3. Proto Version is Locked

The proto package `helm.kernel.v1` increments its `v{N}` suffix ONLY on
breaking wire format changes. Non-breaking additions do NOT change the version.

### 4. Generated Types MUST Match

All `*_gen.*` files across SDKs MUST be regenerated from the same proto
revision. The `codegen-check` CI target enforces this:

```bash
make codegen-check  # Fails if generated files differ from committed
```

### 5. Release Cadence

| Artifact     | Release Trigger  | Automation                     |
| ------------ | ---------------- | ------------------------------ |
| Spec         | RFC finalization | Manual                         |
| Proto IDL    | Spec change      | Manual                         |
| SDK types    | Proto change     | `make codegen`                 |
| SDK packages | SDK code change  | `goreleaser` / per-language CI |

### 6. Version Bump Process

1. Update `protocols/specs/SPEC_VERSION`
2. Regenerate: `make codegen`
3. Run: `make codegen-check` (CI will also verify)
4. Bump SDK versions in their respective manifests
5. Tag release: `git tag v{VERSION}`

## CI Enforcement

The `sdk_gates.yml` workflow triggers on proto changes and verifies:

- All `*_gen.*` files have `AUTO-GENERATED` headers
- Generated files are present for all 5 languages
- Build + test pass for each SDK
