# HELM SDK Codegen Pipeline

> Generate SDK types from the canonical proto IDL.

## Source of Truth

```
protocols/proto/helm/kernel/v1/helm.proto
```

## Generated Targets

| Language   | Output                                      | Generator                              |
| ---------- | ------------------------------------------- | -------------------------------------- |
| Go         | `sdk/go/gen/kernelv1/`                      | `protoc-gen-go` + `protoc-gen-go-grpc` |
| Python     | `sdk/python/helm_sdk/generated/`            | `grpcio-tools`                         |
| TypeScript | `sdk/ts/src/generated/`                     | `ts-proto`                             |
| Java       | `sdk/java/src/main/java/sh/helm/kernel/v1/` | `protoc-gen-java`                      |
| Rust       | `sdk/rust/src/generated/`                   | `tonic-build`                          |

## Usage

```bash
# Generate all SDKs
make codegen

# Generate specific SDK
make codegen-go
make codegen-python
make codegen-ts
make codegen-java
make codegen-rust
```

## Makefile Targets

```makefile
PROTO_DIR := protocols/proto
PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')

.PHONY: codegen
codegen: codegen-go codegen-python codegen-ts codegen-java codegen-rust

codegen-go:
	protoc --go_out=sdk/go/gen --go-grpc_out=sdk/go/gen \
		-I$(PROTO_DIR) $(PROTO_FILES)

codegen-python:
	python -m grpc_tools.protoc --python_out=sdk/python/helm_sdk/generated \
		--grpc_python_out=sdk/python/helm_sdk/generated \
		-I$(PROTO_DIR) $(PROTO_FILES)

codegen-ts:
	protoc --plugin=./node_modules/.bin/protoc-gen-ts_proto \
		--ts_proto_out=sdk/ts/src/generated \
		-I$(PROTO_DIR) $(PROTO_FILES)

codegen-java:
	protoc --java_out=sdk/java/src/main/java \
		--grpc-java_out=sdk/java/src/main/java \
		-I$(PROTO_DIR) $(PROTO_FILES)

codegen-rust:
	cd sdk/rust && cargo build --features codegen
```

## CI Integration

The `sdk_gates.yml` workflow should run codegen and verify no drift:

```yaml
- name: Verify SDK types are up-to-date
  run: |
    make codegen
    git diff --exit-code sdk/
```

## What Gets Generated

From `helm.proto`, each SDK receives:

- `Verdict` enum (ALLOW/DENY/ESCALATE)
- `ReasonCode` enum (generated from `reason-codes-v1.json`)
- `Receipt` message struct
- `DecisionRecord` message struct
- `AuthorizedExecutionIntent` struct
- `Effect` struct
- `PDPRequest`/`PDPResponse` types
- `EffectRequest`/`EffectResponse` types
- gRPC client stubs for `EffectBoundaryService` and `PolicyDecisionPointService`

## Migration from Manual Types

SDKs currently define types manually. Migration path:

1. Generate types alongside existing manual types
2. Add type alias bridge (generated = canonical)
3. Remove manual types once all tests pass against generated
4. CI enforces codegen freshness
