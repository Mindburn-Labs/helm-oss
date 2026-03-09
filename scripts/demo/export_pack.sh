#!/usr/bin/env bash
# HELM Evidence Pack Export Demo
# Runs a minimal conformance check and exports a verifiable Evidence Pack.
#
# Usage: bash scripts/demo/export_pack.sh [output_dir]
#
# Output: a directory containing 00_INDEX.json, 01_SCORE.json, and
#         supporting evidence — verifiable via `npx @mindburn/helm --bundle <dir>`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUTPUT_DIR="${1:-"$PROJECT_ROOT/tmp/demo-evidence-pack"}"

echo "HELM Evidence Pack Export Demo"
echo "=============================="
echo ""

# Step 1: Generate an Evidence Pack using the golden fixture as a template
echo "Step 1: Generating Evidence Pack at $OUTPUT_DIR ..."
mkdir -p "$OUTPUT_DIR"

# Use the fixture generator to create a deterministic pack
node "$PROJECT_ROOT/scripts/generate-fixture.mjs" 2>&1 | sed 's/fixtures\/minimal\//demo output: /'

# Copy the golden fixture as the demo output
cp -r "$PROJECT_ROOT/fixtures/minimal/"* "$OUTPUT_DIR/"
echo "  Evidence Pack written to: $OUTPUT_DIR"
echo ""

# Step 2: Verify the pack
echo "Step 2: Verifying Evidence Pack ..."
echo ""

VERIFY_RESULT=$(npx @mindburn/helm --ci --bundle "$OUTPUT_DIR" 2>/dev/null || true)
VERDICT=$(echo "$VERIFY_RESULT" | jq -r '.verdict // "ERROR"')
BUNDLE_ROOT=$(echo "$VERIFY_RESULT" | jq -r '.roots.manifest_root_hash // ""')
MERKLE_ROOT=$(echo "$VERIFY_RESULT" | jq -r '.roots.merkle_root // ""')

echo "  Verdict:     $VERDICT"
echo "  Bundle root: $BUNDLE_ROOT"
echo "  Merkle root: $MERKLE_ROOT"
echo ""

if [ "$VERDICT" = "PASS" ]; then
    echo "Evidence Pack is verifiable."
else
    echo "WARNING: Verification returned $VERDICT"
    exit 1
fi

echo ""
echo "To verify independently:"
echo "  npx @mindburn/helm --bundle $OUTPUT_DIR"
