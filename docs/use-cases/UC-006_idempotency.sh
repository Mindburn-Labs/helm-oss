#!/bin/bash
# UC-006: Idempotency prevents double execution
# Expected: budget checks enforce single execution
set -euo pipefail

echo "=== UC-006: Idempotency ==="
cd "$(dirname "$0")/../../core"

go test -run TestCheckGas ./pkg/runtime/budget/ -v -count=1

echo "UC-006: PASS"
