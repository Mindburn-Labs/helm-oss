#!/bin/bash
# UC-008: Replay verify offline
# Expected: evidence pack export and verify round-trips
set -euo pipefail

echo "=== UC-008: Replay Verify Offline ==="
cd "$(dirname "$0")/../../core"

go test -run TestExportAndVerify_RoundTrip ./cmd/helm/ -v -count=1
go test -run TestExportPack_Deterministic ./cmd/helm/ -v -count=1

echo "UC-008: PASS"
