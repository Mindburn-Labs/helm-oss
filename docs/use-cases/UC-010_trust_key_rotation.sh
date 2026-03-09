#!/bin/bash
# UC-010: Trust key rotation replay correctness
# Expected: event-sourced registry replays correctly at any Lamport height
set -euo pipefail

echo "=== UC-010: Trust Key Rotation ==="
cd "$(dirname "$0")/../../core"

go test -run TestTrustRegistry ./pkg/trust/registry/ -v -count=1

echo "UC-010: PASS"
