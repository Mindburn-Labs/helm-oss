# HELM Verify Install Guide

> How to verify the integrity of a HELM installation.

## Quick Verification

```bash
# After installing via install.sh
helm version --verify
```

This command:

1. Prints the installed version, commit, and build date
2. Verifies the binary checksum against `SHA256SUMS.txt`
3. Verifies the cosign signature if `cosign` is installed
4. Outputs a pass/fail status

## Manual Verification

### 1. Download Checksums

```bash
VERSION=$(helm version --short)
curl -fsSL "https://github.com/Mindburn-Labs/helm-oss/releases/download/${VERSION}/SHA256SUMS.txt" \
  -o /tmp/SHA256SUMS.txt
```

### 2. Verify Binary Checksum

```bash
BINARY_PATH=$(which helm)
EXPECTED=$(grep "$(basename $BINARY_PATH)" /tmp/SHA256SUMS.txt | awk '{print $1}')
ACTUAL=$(shasum -a 256 "$BINARY_PATH" | awk '{print $1}')

if [ "$EXPECTED" = "$ACTUAL" ]; then
  echo "✅ Checksum verified"
else
  echo "❌ Checksum mismatch — binary may be tampered"
  exit 1
fi
```

### 3. Verify Cosign Signature

```bash
cosign verify-blob \
  --signature "https://github.com/Mindburn-Labs/helm-oss/releases/download/${VERSION}/SHA256SUMS.txt.sig" \
  --certificate-identity-regexp ".*@mindburn.io" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  /tmp/SHA256SUMS.txt
```

### 4. Verify SBOM

```bash
# Download SBOM
curl -fsSL "https://github.com/Mindburn-Labs/helm-oss/releases/download/${VERSION}/helm.sbom.json" \
  -o /tmp/helm.sbom.json

# Inspect with syft
syft parse /tmp/helm.sbom.json
```

## Container Image Verification

```bash
# Verify GHCR container image
cosign verify ghcr.io/mindburn-labs/helm:latest \
  --certificate-identity-regexp ".*@mindburn.io" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

## Trust Chain Summary

```
GitHub Actions (OIDC) → cosign keyless signing → SHA256SUMS.txt.sig
Binary → SHA-256 → SHA256SUMS.txt
SBOM → CycloneDX / SPDX-JSON → helm.sbom.json
Container → cosign image signing → GHCR attestation
```
