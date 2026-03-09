#!/bin/bash
# UC-005: WASI gas exhaustion
# Expected: deterministic termination with ERR_COMPUTE_GAS_EXHAUSTED
set -euo pipefail

echo "=== UC-005: WASI Gas Exhaustion ==="
cd "$(dirname "$0")/../../core"

go test -run TestWASI_InfiniteLoop ./pkg/runtime/sandbox/ -v -count=1
go test -run TestWASI_MemoryBomb ./pkg/runtime/sandbox/ -v -count=1
go test -run TestWASI_DeterministicTermination ./pkg/runtime/sandbox/ -v -count=1

echo "UC-005: PASS"
