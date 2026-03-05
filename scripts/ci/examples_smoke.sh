#!/usr/bin/env bash
# HELM Examples Smoke Test
# Validates all example directories have correct structure.
# Optionally builds compilable examples.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
EXAMPLES_DIR="$PROJECT_ROOT/examples"

PASS=0
FAIL=0

has_entrypoint_file() {
    local dir="$1"
    local f
    local pattern

    for f in main.py main.ts main.js main.go main.sh main.rs Main.java shim.py; do
        if [ -f "$dir/$f" ]; then
            return 0
        fi
    done

    for pattern in "*_helm.py" "verify_*.py" "verify_*.ts"; do
        if compgen -G "$dir/$pattern" >/dev/null; then
            return 0
        fi
    done

    if [ -f "$dir/pom.xml" ] || [ -f "$dir/Cargo.toml" ] || [ -f "$dir/package.json" ] || [ -f "$dir/pyproject.toml" ]; then
        return 0
    fi

    # Data/spec examples (e.g. JSON templates).
    if compgen -G "$dir/*.json" >/dev/null; then
        return 0
    fi

    return 1
}

has_nested_example_group() {
    local dir="$1"
    local sub
    for sub in "$dir"/*/; do
        [ -d "$sub" ] || continue
        if has_entrypoint_file "$sub"; then
            return 0
        fi
    done
    return 1
}

check_example() {
    local dir="$1"
    local name
    name="$(basename "$dir")"
    echo -n "  $name ... "

    # Every example must have a README.md
    if [ ! -f "$dir/README.md" ]; then
        echo "❌ FAIL (missing README.md)"
        FAIL=$((FAIL + 1))
        return
    fi

    if ! has_entrypoint_file "$dir" && ! has_nested_example_group "$dir"; then
        echo "❌ FAIL (no recognized entrypoint found)"
        FAIL=$((FAIL + 1))
        return
    fi

    echo "✅ PASS"
    PASS=$((PASS + 1))
}

echo "HELM Examples Smoke Test"
echo "════════════════════════"
echo ""

for dir in "$EXAMPLES_DIR"/*/; do
    [ -d "$dir" ] && check_example "$dir"
done

# Build-check Go example if go is available
if command -v go &>/dev/null && [ -d "$EXAMPLES_DIR/go_client" ]; then
    echo ""
    echo "  go_client build check ... "
    if [ -f "$EXAMPLES_DIR/go_client/go.mod" ]; then
        if (cd "$EXAMPLES_DIR/go_client" && GOWORK=off go build -o /dev/null . 2>&1); then
            echo "  go_client build ✅"
        else
            echo "  go_client build ⚠️ (non-fatal)"
        fi
    else
        echo "  go_client build ⏭️  skipped (no go.mod in standalone example)"
    fi
fi

echo ""
echo "════════════════════════"
echo "Results: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
