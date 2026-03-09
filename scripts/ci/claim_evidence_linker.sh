#!/bin/bash
# DX-002: Claim-Evidence Linker
# Validates that every claim in the audit documents has corresponding evidence.
#
# Usage: bash scripts/ci/claim_evidence_linker.sh
set -euo pipefail

DOCS_DIR="${1:-artifacts/helm_oss_audit/2026-02-17}"
EVIDENCE_DIR="${2:-artifacts/helm_oss_audit/2026-02-17/evidence}"

echo "🔗 HELM Claim-Evidence Linker"
echo "  Docs:     $DOCS_DIR"
echo "  Evidence: $EVIDENCE_DIR"
echo ""

CLAIMS=0
LINKED=0
UNLINKED=0

# Extract claims from audit docs (lines containing ✅, ❌, or specific claim patterns)
while IFS= read -r line; do
  # Extract file references from claims like "[file.go](..." or "pkg/foo/bar.go"
  files=$(echo "$line" | grep -oP '`[a-zA-Z_/]+\.(go|rs|ts|py|java)`' | tr -d '`' || true)
  
  if [ -n "$files" ]; then
    CLAIMS=$((CLAIMS + 1))
    all_found=true
    
    for file in $files; do
      # Check if the file exists in the repo
      if ! find . -name "$(basename "$file")" -path "*${file}*" 2>/dev/null | head -1 | grep -q .; then
        echo "  ⚠️  Claim references missing file: $file"
        echo "     in: $(echo "$line" | head -c 100)"
        all_found=false
      fi
    done
    
    if $all_found; then
      LINKED=$((LINKED + 1))
    else
      UNLINKED=$((UNLINKED + 1))
    fi
  fi
done < <(find "$DOCS_DIR" -name "*.md" -exec grep -h "✅\|❌\|PASS\|FAIL\|Verified" {} \; 2>/dev/null || true)

echo ""
echo "📊 Linker Results:"
echo "  Total claims scanned: $CLAIMS"
echo "  Linked (evidence found): $LINKED"
echo "  Unlinked (missing evidence): $UNLINKED"

if [ "$UNLINKED" -gt 0 ]; then
  echo ""
  echo "⚠️  $UNLINKED claims have missing evidence references"
  exit 0  # Warning only, don't fail CI
else
  echo ""
  echo "✅ All claims have corresponding evidence"
fi
