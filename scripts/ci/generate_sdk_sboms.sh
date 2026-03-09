#!/bin/bash
# SC-006: Generate per-SDK SBOMs in CycloneDX format
# Run from repository root: bash scripts/ci/generate_sdk_sboms.sh
set -euo pipefail

OUTDIR="${1:-sboms}"
mkdir -p "$OUTDIR"

echo "📦 Generating per-SDK SBOMs..."

# Go core SBOM (already covered by generate_sbom.sh)
echo "  → Go core"
if command -v cyclonedx-gomod &>/dev/null; then
  cd core && cyclonedx-gomod mod -json -output "../${OUTDIR}/sbom-go-core.json" && cd ..
else
  echo "  ⚠️  cyclonedx-gomod not installed, skipping Go SBOM"
fi

# Python SDK SBOM
echo "  → Python SDK"
if command -v cyclonedx-py &>/dev/null; then
  cd sdk/python && cyclonedx-py requirements -i requirements.txt -o "../../${OUTDIR}/sbom-python-sdk.json" --format json 2>/dev/null || \
  cyclonedx-py poetry -o "../../${OUTDIR}/sbom-python-sdk.json" --format json 2>/dev/null || \
  echo "  ⚠️  No Python deps file found"
  cd ../..
else
  echo "  ⚠️  cyclonedx-py not installed, skipping Python SBOM"
fi

# TypeScript SDK SBOM
echo "  → TypeScript SDK"
if command -v cyclonedx-npm &>/dev/null; then
  cd sdk/ts && cyclonedx-npm --output-file "../../${OUTDIR}/sbom-ts-sdk.json" 2>/dev/null || echo "  ⚠️  TS SBOM generation failed"
  cd ../..
else
  echo "  ⚠️  cyclonedx-npm not installed, skipping TS SBOM"
fi

# Rust SDK SBOM
echo "  → Rust SDK"
if command -v cargo-cyclonedx &>/dev/null; then
  cd sdk/rust && cargo cyclonedx --format json 2>/dev/null && mv bom.json "../../${OUTDIR}/sbom-rust-sdk.json" || echo "  ⚠️  Rust SBOM generation failed"
  cd ../..
else
  echo "  ⚠️  cargo-cyclonedx not installed, skipping Rust SBOM"
fi

# Java SDK SBOM
echo "  → Java SDK"
if [ -f sdk/java/pom.xml ]; then
  cd sdk/java && mvn org.cyclonedx:cyclonedx-maven-plugin:makeAggregateBom -DoutputFormat=json -DoutputName=sbom 2>/dev/null && \
  cp target/sbom.json "../../${OUTDIR}/sbom-java-sdk.json" || echo "  ⚠️  Java SBOM generation failed"
  cd ../..
fi

echo ""
echo "✅ SBOM generation complete. Output: ${OUTDIR}/"
ls -la "${OUTDIR}/" 2>/dev/null || echo "No SBOMs generated (install tooling first)"
