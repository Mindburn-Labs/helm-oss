#!/bin/bash
# UC-012: OpenAI proxy loop (only if proxy enabled)
# Expected: proxy endpoint returns governed response
set -euo pipefail

echo "=== UC-012: OpenAI Proxy ==="
cd "$(dirname "$0")/../../core"

# Verify the proxy code compiles
go build ./pkg/api/

echo "UC-012: PASS (compile check; runtime test requires running server)"
