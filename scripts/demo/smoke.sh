#!/usr/bin/env bash
# HELM Demo Smoke Test
# Usage: ./smoke.sh [BASE_URL]
# Default BASE_URL: http://localhost:8080

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"

echo "HELM Demo Smoke Test: $BASE_URL"
echo "-----------------------------------"

# 1. Health Check
echo -n "1. Checking /healthz... "
resp=$(curl -s -k -L -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")
if [ "$resp" != "200" ]; then
    echo "FAIL ($resp)"
    exit 1
fi
echo "OK"

# 2. Tool Execute: DENY
echo -n "2. Checking /v1/tools/execute (DENY)... "
resp=$(curl -s -k -L -X POST "$BASE_URL/v1/tools/execute" \
    -H "Content-Type: application/json" \
    -d '{
        "tool": "fail_deny_demo",
        "args": {}
    }')

verdict=$(echo "$resp" | grep -o '"verdict":"[^"]*"' | cut -d'"' -f4)
if [ "$verdict" != "DENY" ]; then
    echo "FAIL (Expected DENY, got $verdict)"
    echo "Response: $resp"
    exit 1
fi
echo "OK ($verdict)"

# 3. Tool Execute: ALLOW (echo)
echo -n "3. Checking /v1/tools/execute (ALLOW echo)... "
resp=$(curl -s -k -L -X POST "$BASE_URL/v1/tools/execute" \
    -H "Content-Type: application/json" \
    -d '{
        "tool": "echo",
        "args": {"message": "smoke_test"}
    }')

verdict=$(echo "$resp" | grep -o '"verdict":"[^"]*"' | cut -d'"' -f4)
if [ "$verdict" != "ALLOW" ]; then
    echo "FAIL (Expected ALLOW, got $verdict)"
    echo "Response: $resp"
    exit 1
fi
# Check output
if [[ "$resp" != *"smoke_test"* ]]; then
    echo "FAIL (Output mismatch)"
    echo "Response: $resp"
    exit 1
fi
echo "OK ($verdict)"

# 4. Receipt Listing
echo -n "4. Checking /api/v1/receipts... "
resp=$(curl -s -k -L "$BASE_URL/api/v1/receipts?limit=5")
count=$(echo "$resp" | grep -o "receipt_id" | wc -l)
if [ "$count" -lt 1 ]; then
    echo "FAIL (No receipts found)"
    echo "Response: $resp"
    exit 1
fi
echo "OK ($count receipts)"

# 5. Export
echo -n "5. Checking /api/v1/export... "
resp=$(curl -s -k -L -I -X POST "$BASE_URL/api/v1/export")
if ! echo "$resp" | grep -i -q "Content-Disposition: attachment"; then
    echo "FAIL (Missing header)"
    echo "$resp"
    exit 1
fi
echo "OK"

# 6. Limits
echo -n "6. Checking /limits... "
resp=$(curl -s -k -L "$BASE_URL/limits")
if [[ "$resp" != *"allowed_tools"* ]]; then
    echo "FAIL"
    echo "Response: $resp"
    exit 1
fi
echo "OK"

echo "-----------------------------------"
echo "✅ All checks passed!"
