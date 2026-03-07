# HELM SDK Documentation

## Architecture

```
api/openapi/helm.openapi.yaml    ← Single source of truth
         │
    scripts/sdk/gen.sh           ← Generates types_gen.* files
         │
    ┌────┴────┬────────┬─────────┬──────────┐
    ▼         ▼        ▼         ▼          ▼
sdk/ts/    sdk/python/ sdk/go/  sdk/rust/  sdk/java/
 types.gen.ts  types_gen.py  types_gen.go  types_gen.rs  TypesGen.java
 client.ts     client.py     client.go     lib.rs        HelmClient.java
    │         │        │         │          │
    ▼         ▼        ▼         ▼          ▼
examples/  examples/  examples/ examples/  examples/
```

## Contract → Code → Wrapper → Example

| Layer | Purpose | Location |
|-------|---------|----------|
| **OpenAPI spec** | Single truth source | `api/openapi/helm.openapi.yaml` |
| **Generated types** | Typed models | `sdk/*/types_gen.*` (AUTO-GENERATED) |
| **Ergonomic wrapper** | HTTP client with error handling | `sdk/*/client.*` (hand-written) |
| **Examples** | Runnable demos | `examples/` |

## SDKs

| Language | Package | Install |
|----------|---------|---------|
| TypeScript | `@mindburn/helm-sdk` | `npm install @mindburn/helm-sdk` |
| Python | `helm-sdk` | `pip install helm-sdk` |
| Go | `github.com/Mindburn-Labs/helm-oss/sdk/go` | `go get ...` |
| Rust | `helm-sdk` | `cargo add helm-sdk` |
| Java | `ai.mindburn.helm:helm-sdk` | Maven |

## CI Gates

| Gate | Script | What it checks |
|------|--------|---------------|
| SDK drift | `scripts/ci/sdk_drift_check.sh` | Regenerate → diff = fail |
| SDK build/test | `scripts/ci/sdk_build_test.sh` | Build + test all 5 SDKs |
| OpenAPI lint | `.github/workflows/sdk_gates.yml` | Redocly lint |

## Per-SDK READMEs

- [TypeScript](../../sdk/ts/README.md)
- [Python](../../sdk/python/README.md)
- [Go](../../sdk/go/README.md)
- [Rust](../../sdk/rust/README.md)
- [Java](../../sdk/java/README.md)
