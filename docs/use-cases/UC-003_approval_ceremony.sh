#!/bin/bash
# UC-003: Approval ceremony blocks/unblocks
# Expected: strict ceremony validation enforced
set -euo pipefail

echo "=== UC-003: Approval Ceremony ==="
cd "$(dirname "$0")/../../core"

go test -run TestValidateCeremony ./pkg/escalation/ceremony/ -v -count=1

echo "UC-003: PASS"
