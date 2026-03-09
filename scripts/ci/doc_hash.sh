#!/usr/bin/env bash
# Canonical-Doc-Hash enforcement for HELM standard document.
# Verifies the sha256 hash in the document header matches computed content hash.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

DOC="$PROJECT_ROOT/docs/standard/HELM_Unified_Canonical_Standard.md"

# Fall back to root-level file if docs/standard/ version not found
if [ ! -f "$DOC" ]; then
    DOC="$PROJECT_ROOT/HELM_Unified_Canonical_Standard.md"
fi

if [ ! -f "$DOC" ]; then
    echo "ERROR: Canonical standard document not found"
    echo "  Checked: docs/standard/HELM_Unified_Canonical_Standard.md"
    echo "  Checked: HELM_Unified_Canonical_Standard.md"
    exit 1
fi

# Extract the current hash from the header (macOS + Linux compatible)
CURRENT_HASH=$(grep 'Canonical-Doc-Hash: sha256:' "$DOC" | head -1 | sed 's/.*sha256://' | grep -o '[a-f0-9]\{64\}' || true)

if [ -z "$CURRENT_HASH" ]; then
    echo "ERROR: No Canonical-Doc-Hash header found in document"
    exit 1
fi

echo "Found Canonical-Doc-Hash: sha256:$CURRENT_HASH"

# Zero the hash field per the spec, then compute sha256
ZEROED_HASH="0000000000000000000000000000000000000000000000000000000000000000"
COMPUTED_HASH=$(sed "s/Canonical-Doc-Hash: sha256:$CURRENT_HASH/Canonical-Doc-Hash: sha256:$ZEROED_HASH/" "$DOC" | shasum -a 256 | awk '{print $1}')

echo "Computed hash:          sha256:$COMPUTED_HASH"

if [ "$CURRENT_HASH" = "$COMPUTED_HASH" ]; then
    echo "✅ Canonical-Doc-Hash verified"
    exit 0
else
    echo "❌ Canonical-Doc-Hash MISMATCH"
    echo "   Expected: $CURRENT_HASH"
    echo "   Computed: $COMPUTED_HASH"
    echo ""
    echo "To fix: update the hash in the document header to sha256:$COMPUTED_HASH"
    exit 1
fi
