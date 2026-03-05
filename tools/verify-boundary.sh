#!/bin/bash
# verify-boundary.sh — Verify that protected kernel/authority paths are in sync
# between the OSS and commercial repos. Fails on any diff.
#
# Usage: ./tools/verify-boundary.sh [OSS_ROOT] [COMMERCIAL_ROOT]
# Defaults: OSS=../helm-public, COMMERCIAL=<script's repo root>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Detect which repo this script is running from
if [ -d "$REPO_ROOT/commercial" ]; then
  # Running from commercial
  COMMERCIAL_ROOT="$REPO_ROOT"
  OSS_ROOT="${1:-$(cd "$REPO_ROOT/../helm-public" 2>/dev/null && pwd || echo "")}"
elif [ -d "$REPO_ROOT/core/pkg/kernel" ]; then
  # Running from OSS
  OSS_ROOT="$REPO_ROOT"
  COMMERCIAL_ROOT="${1:-$(cd "$REPO_ROOT/../helm" 2>/dev/null && pwd || echo "")}"
fi

# Allow explicit overrides
if [ $# -ge 2 ]; then
  OSS_ROOT="$1"
  COMMERCIAL_ROOT="$2"
fi

if [ -z "$OSS_ROOT" ] || [ ! -d "$OSS_ROOT/core/pkg/kernel" ]; then
  echo "ERROR: Cannot find OSS repo. Pass path as argument."
  echo "Usage: $0 [OSS_ROOT] [COMMERCIAL_ROOT]"
  exit 1
fi

if [ -z "$COMMERCIAL_ROOT" ] || [ ! -d "$COMMERCIAL_ROOT/core/pkg/kernel" ]; then
  echo "ERROR: Cannot find commercial repo. Pass path as argument."
  echo "Usage: $0 [OSS_ROOT] [COMMERCIAL_ROOT]"
  exit 1
fi

echo "═══════════════════════════════════════════════════════"
echo "  HELM Repo Boundary Verification"
echo "  OSS:        $OSS_ROOT"
echo "  Commercial: $COMMERCIAL_ROOT"
echo "═══════════════════════════════════════════════════════"
echo ""

# Protected paths — must be identical between repos
PROTECTED_PATHS=(
  "core/pkg/kernel"
  "core/pkg/contracts"
  "core/pkg/crypto"
  "core/pkg/evidencepack"
  "core/pkg/proofgraph"
  "core/pkg/receipts"
  "core/pkg/verifier"
  "core/pkg/connectors/sandbox"
  "core/pkg/conformance"
  "core/pkg/incubator/audit"
  "core/pkg/integrations/receipts"
  "core/pkg/integrations/capgraph"
  "core/pkg/integrations/manifest"
  "core/pkg/api"
  "core/pkg/trust/registry"
  "core/pkg/guardian"
  "protocols"
  "schemas"
)

PASS=0
FAIL=0
SKIP=0

for path in "${PROTECTED_PATHS[@]}"; do
  oss_path="$OSS_ROOT/$path"
  comm_path="$COMMERCIAL_ROOT/$path"

  if [ ! -d "$oss_path" ] && [ ! -d "$comm_path" ]; then
    SKIP=$((SKIP + 1))
    continue
  fi

  if [ ! -d "$oss_path" ]; then
    echo "  ✗ FAIL: $path — exists only in commercial (must be upstreamed to OSS)"
    FAIL=$((FAIL + 1))
    continue
  fi

  if [ ! -d "$comm_path" ]; then
    echo "  ✗ FAIL: $path — exists only in OSS (run sync-oss-kernel.sh)"
    FAIL=$((FAIL + 1))
    continue
  fi

  # Compare, ignoring generated/binary files
  DIFF_OUTPUT=$(diff -rq \
    --exclude="*.pyc" \
    --exclude="__pycache__" \
    --exclude=".ruff_cache" \
    --exclude="node_modules" \
    --exclude="dist" \
    --exclude="target" \
    --exclude=".package-lock.json" \
    "$oss_path" "$comm_path" 2>/dev/null || true)

  if [ -z "$DIFF_OUTPUT" ]; then
    echo "  ✓ PASS: $path"
    PASS=$((PASS + 1))
  else
    echo "  ✗ FAIL: $path"
    echo "$DIFF_OUTPUT" | head -10 | sed 's/^/      /'
    DIFF_COUNT=$(echo "$DIFF_OUTPUT" | wc -l | tr -d ' ')
    if [ "$DIFF_COUNT" -gt 10 ]; then
      echo "      ... and $((DIFF_COUNT - 10)) more differences"
    fi
    FAIL=$((FAIL + 1))
  fi
done

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo "═══════════════════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "ACTION REQUIRED: Run 'tools/sync-oss-kernel.sh' in commercial"
  echo "  or upstream changes to OSS and re-sync."
  exit 1
fi

echo ""
echo "All protected paths are in sync. ✓"
exit 0
