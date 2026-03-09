#!/bin/bash
# UC-004: WASI pure transform
# Expected: sandbox defaults enforce bounded compute
set -euo pipefail

echo "=== UC-004: WASI Pure Transform ==="
cd "$(dirname "$0")/../../core"

go test -run TestWASI_DenyByDefault ./pkg/runtime/sandbox/ -v -count=1

echo "UC-004: PASS"
