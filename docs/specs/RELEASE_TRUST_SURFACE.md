# HELM Release Trust Surface

> Actual trust-bearing release surface for the current OSS distribution.

## Trust Chain

```
Source Code → GitHub Actions build → release artifacts → checksum signature → build provenance
    ↓                 ↓                    ↓                    ↓                  ↓
   git tag      workflow gates       binaries + SBOM      cosign keyless      attestation
```

## Release Artifacts

Current release workflow publishes:

| Artifact | Format | Purpose |
| --- | --- | --- |
| `helm-<os>-<arch>` | ELF/Mach-O/PE | Executable binary |
| `SHA256SUMS.txt` | Text | Binary checksum verification |
| `SHA256SUMS.txt.sig` | Cosign signature | Signature over `SHA256SUMS.txt` |
| `sbom.json` | CycloneDX JSON | Software bill of materials |
| `helm-evidence-*` | Bundle | Release evidence artifacts |
| `helm-attestation-*` | Bundle | Additional release attestations |
| `golden-evidencepack.tar` | Tar | Golden verification artifact |
| `golden-run-report.html` | HTML | Golden report artifact |
| `helm.mcpb` | Zip | MCP bundle artifact |

## Verification Flow

```bash
# 1. Download binary and release metadata
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/helm-darwin-arm64
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/SHA256SUMS.txt
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/SHA256SUMS.txt.sig
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/sbom.json

# 2. Verify checksum
shasum -a 256 -c SHA256SUMS.txt

# 3. Verify signature (keyless GitHub Actions identity)
cosign verify-blob \
  --signature SHA256SUMS.txt.sig \
  --certificate-identity-regexp ".*@mindburn.io" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  SHA256SUMS.txt
```

## Release Pipeline

The canonical release surface is `.github/workflows/release.yml`, not GoReleaser.

Current workflow truth:

- cross-compiles release binaries
- generates `SHA256SUMS.txt`
- generates `sbom.json`
- signs checksums with `cosign sign-blob`
- attaches build provenance via `actions/attest-build-provenance@v2`
- publishes only channels that are currently supported

## Provenance

Build provenance is generated in GitHub Actions and attached to release outputs.

- attestation action: `actions/attest-build-provenance@v2`
- checksum signatures: keyless cosign via GitHub OIDC
- container images: keyless cosign signing in the release workflow

## Key Material

| Material | Purpose | Current source |
| --- | --- | --- |
| GitHub OIDC identity | Keyless signing for checksums and images | GitHub Actions |
| `HELM_SIGNING_KEY` | Evidence bundle signing | Release environment secret |
| Registry credentials | Publish authenticated artifacts | Protected GitHub environments |

## Automation Lifecycle

### Maintenance Runs

```yaml
# Scheduled governance health checks
schedule:
  drift_check: daily
  policy_refresh: hourly
  certificate_rotation: monthly
  compliance_audit: weekly
```

### Drift Detection

```bash
# Detect policy drift
helm drift check --policies ./policies/ --baseline ./baseline/

# Detect SDK drift
helm drift check --sdks --proto protocols/proto/helm/kernel/v1/helm.proto

# Detect conformance drift
helm conform check --vectors protocols/conformance/v1/test-vectors.json
```

### Retention Tiers

| Tier      | Data              | Retention | Storage            |
| --------- | ----------------- | --------- | ------------------ |
| Hot       | Active receipts   | 90 days   | Local DB           |
| Warm      | Anchored receipts | 1 year    | Object storage     |
| Cold      | Archive           | 7 years   | Compliance archive |
| Immutable | Transparency log  | Forever   | Rekor/S3 Glacier   |
