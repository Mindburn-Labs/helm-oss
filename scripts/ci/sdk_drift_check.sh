#!/usr/bin/env bash
# SDK Drift Check — CI gate
# Regenerates SDK types and fails if there's any diff.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "SDK Drift Check"
echo "═══════════════"

# Run the generator
bash "$PROJECT_ROOT/scripts/sdk/gen.sh"

# Check for drift
if ! git diff --quiet -- sdk/; then
    echo ""
    echo "❌ SDK drift detected! Generated types don't match committed files."
    echo ""
    git diff --stat -- sdk/
    echo ""
    echo "Fix: run 'bash scripts/sdk/gen.sh' and commit the changes."
    exit 1
fi

echo "✅ No SDK drift — generated types match committed files."
