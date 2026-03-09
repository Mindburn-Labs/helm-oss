#!/usr/bin/env bash
# SDK Build + Test — CI gate
# Builds and tests all 5 SDK packages.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PASS=0
FAIL=0

run_sdk() {
    local name="$1"
    local cmd="$2"
    echo -n "  $name ... "
    if eval "$cmd" > /dev/null 2>&1; then
        echo "✅ PASS"
        PASS=$((PASS + 1))
    else
        echo "❌ FAIL"
        FAIL=$((FAIL + 1))
    fi
}

echo "SDK Build + Test"
echo "════════════════"

# TypeScript
run_sdk "TypeScript (build)" "cd $PROJECT_ROOT/sdk/ts && npm ci --ignore-scripts 2>/dev/null; npm run build"
run_sdk "TypeScript (pack)"  "cd $PROJECT_ROOT/sdk/ts && npm pack --dry-run"

# Python
run_sdk "Python (build)" "cd $PROJECT_ROOT/sdk/python && python3 -m build 2>/dev/null || pip wheel -w dist . --no-deps"
run_sdk "Python (import)" "cd $PROJECT_ROOT/sdk/python && pip install -e . 2>/dev/null && python3 -c 'from helm_sdk import HelmClient; print(\"ok\")'"

# Go
run_sdk "Go (build)" "cd $PROJECT_ROOT/sdk/go && go build ./..."
run_sdk "Go (test)"  "cd $PROJECT_ROOT/sdk/go && go test ./... -count=1"

# Rust
run_sdk "Rust (build)" "cd $PROJECT_ROOT/sdk/rust && cargo build"
run_sdk "Rust (test)"  "cd $PROJECT_ROOT/sdk/rust && cargo test"

# Java
run_sdk "Java (build)" "cd $PROJECT_ROOT/sdk/java && mvn -q compile -DskipTests"
run_sdk "Java (test)"  "cd $PROJECT_ROOT/sdk/java && mvn -q test"

echo ""
echo "════════════════"
echo "Results: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
