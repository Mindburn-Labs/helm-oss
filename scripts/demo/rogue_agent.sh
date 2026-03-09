#!/usr/bin/env bash
# Rogue Agent Demo — Budget Exhaustion Reproduction
#
# Demonstrates HELM proxy governance:
# 1. Starts proxy with tight budget ($0.05 daily = 5 tool calls)
# 2. Sends requests that consume budget
# 3. On exhaustion, shows 403 + BUDGET_EXHAUSTED receipt
# 4. Exports EvidencePack and verifies integrity
#
# Usage: bash scripts/demo/rogue_agent.sh
#
# Prerequisites:
#   - helm binary built (cd core && go build -o ../bin/helm ./cmd/helm/)
#   - No other process on port 19090
set -euo pipefail

HELM="${HELM_BIN:-./bin/helm}"
PORT=19090
RECEIPTS_DIR=$(mktemp -d)
PROXY_URL="http://localhost:${PORT}"
BUDGET_CENTS=5  # $0.05 = 5 tool calls at 1 cent each

echo "═══════════════════════════════════════"
echo "  HELM Rogue Agent Demo"
echo "═══════════════════════════════════════"
echo ""

# Check if helm binary exists
if [ ! -x "$HELM" ]; then
  echo "❌ helm binary not found at $HELM"
  echo "   Build it: cd core && go build -o ../bin/helm ./cmd/helm/"
  exit 1
fi

# 1. Start proxy with tight budget
echo "▶ Starting proxy with ${BUDGET_CENTS}-cent daily budget..."
$HELM proxy \
  --upstream https://api.openai.com/v1 \
  --port "$PORT" \
  --sign demo-key \
  --tenant-id rogue-agent-demo \
  --daily-limit "$BUDGET_CENTS" \
  --monthly-limit 100 \
  --max-iterations 20 \
  --max-wallclock 60s \
  --receipts-dir "$RECEIPTS_DIR" \
  --verbose &
PROXY_PID=$!

# Clean up on exit
cleanup() {
  echo ""
  echo "▶ Stopping proxy (PID $PROXY_PID)..."
  kill "$PROXY_PID" 2>/dev/null || true
  wait "$PROXY_PID" 2>/dev/null || true
}
trap cleanup EXIT

# Wait for proxy to start
sleep 2
echo "  ✅ Proxy running on $PROXY_URL"
echo ""

# 2. Send requests that consume budget
echo "▶ Sending tool-call requests (budget: ${BUDGET_CENTS} cents)..."
echo ""

# Build a request with a tool_call in the response
# Note: Without a real API key, the upstream will reject — but the proxy
# still processes the response and records receipts. For a real demo,
# set OPENAI_API_KEY.
REQUEST_BODY='{"model":"gpt-4","messages":[{"role":"user","content":"What is 2+2?"}]}'

for i in $(seq 1 $((BUDGET_CENTS + 2))); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${PROXY_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "$REQUEST_BODY" 2>/dev/null || echo "000")

  RECEIPT_ID=$(curl -s -D - -o /dev/null \
    -X POST "${PROXY_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "$REQUEST_BODY" 2>/dev/null | grep -i "X-Helm-Receipt-ID" | tr -d '\r' | awk '{print $2}')

  REASON=$(curl -s -D - -o /dev/null \
    -X POST "${PROXY_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "$REQUEST_BODY" 2>/dev/null | grep -i "X-Helm-Reason-Code" | tr -d '\r' | awk '{print $2}')

  echo "  Request $i: HTTP $STATUS | receipt=$RECEIPT_ID | reason=$REASON"
done

echo ""

# 3. Check receipts
echo "▶ Checking receipts..."
RECEIPT_FILES=$(find "$RECEIPTS_DIR" -name "*.jsonl" 2>/dev/null | head -1)
if [ -n "$RECEIPT_FILES" ]; then
  RECEIPT_COUNT=$(wc -l < "$RECEIPT_FILES" | tr -d ' ')
  echo "  ✅ $RECEIPT_COUNT receipts written to $RECEIPT_FILES"

  # Show last receipt
  echo ""
  echo "  Last receipt:"
  tail -1 "$RECEIPT_FILES" | python3 -m json.tool 2>/dev/null || tail -1 "$RECEIPT_FILES"
else
  echo "  ⚠️  No receipt files found (expected with upstream errors)"
fi

# 4. Check ProofGraph
echo ""
echo "▶ Checking ProofGraph..."
PG_FILE="$RECEIPTS_DIR/proofgraph.json"
if [ -f "$PG_FILE" ]; then
  NODE_COUNT=$(python3 -c "import json; d=json.load(open('$PG_FILE')); print(d.get('count', 0))" 2>/dev/null || echo "?")
  echo "  ✅ ProofGraph persisted: $NODE_COUNT nodes"
else
  echo "  ⚠️  No ProofGraph file (may not have processed any tool_calls)"
fi

# 5. Export pack
echo ""
echo "▶ Creating EvidencePack..."
PACK_PATH="/tmp/rogue-agent-demo-pack.tar.gz"
$HELM pack create \
  --session rogue-agent-demo \
  --receipts "$RECEIPTS_DIR" \
  --out "$PACK_PATH" 2>/dev/null && {
  echo "  ✅ Pack created: $PACK_PATH"

  # 6. Verify pack
  echo ""
  echo "▶ Verifying EvidencePack..."
  $HELM pack verify --bundle "$PACK_PATH" --json
} || echo "  ⚠️  Pack creation failed (may have no files)"

echo ""
echo "═══════════════════════════════════════"
echo "  Demo Complete"
echo "═══════════════════════════════════════"
echo ""
echo "Key takeaways:"
echo "  • Every tool call was governed by Guardian → ProofGraph → Budget"
echo "  • Budget exhaustion produces deterministic deny"
echo "  • Receipts form a causal chain (prevHash linking)"
echo "  • ProofGraph DAG persisted to disk"
echo "  • EvidencePack is verifiable offline"
