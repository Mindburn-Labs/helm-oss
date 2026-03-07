# HELM Release Trust Surface

> Ensuring every release is verifiable, reproducible, and auditable.

## Trust Chain

```
Source Code → CI Build → Signed Binary → SBOM → Transparency Log
    ↓            ↓           ↓            ↓          ↓
  git tag    attestation   cosign      CycloneDX    Rekor
```

## Release Artifacts

Every HELM release publishes:

| Artifact              | Format           | Purpose                      |
| --------------------- | ---------------- | ---------------------------- |
| Binary (per-platform) | ELF/Mach-O/PE    | Executable                   |
| SHA256SUMS            | Text             | Checksum verification        |
| SHA256SUMS.sig        | Cosign signature | Signature over checksums     |
| SBOM                  | CycloneDX JSON   | Software Bill of Materials   |
| SLSA Provenance       | In-toto JSON     | Build provenance attestation |
| Transparency entry    | Rekor            | Immutable record of release  |

## Verification Flow

```bash
# 1. Download binary and checksums
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/helm-darwin-arm64
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/SHA256SUMS
curl -LO https://github.com/Mindburn-Labs/helm-oss/releases/download/v1.0.0/SHA256SUMS.sig

# 2. Verify checksum
sha256sum --check SHA256SUMS

# 3. Verify signature
cosign verify-blob --key https://helm.sh/keys/release.pub \
  --signature SHA256SUMS.sig SHA256SUMS

# 4. Verify SBOM
helm verify sbom --release v1.0.0

# 5. Verify SLSA provenance
slsa-verifier verify-artifact helm-darwin-arm64 \
  --source-uri github.com/Mindburn-Labs/helm-oss \
  --source-tag v1.0.0
```

## GoReleaser Pipeline

```yaml
# .goreleaser.yml integration
signs:
  - cmd: cosign
    args:
      [
        "sign-blob",
        "--key=env://COSIGN_KEY",
        "--output-signature=${signature}",
        "${artifact}",
      ]
    artifacts: checksum

sboms:
  - cmd: syft
    args: ["${artifact}", "--output", "cyclonedx-json=${document}"]
    artifacts: binary
```

## Provenance

SLSA Level 3 provenance is generated via GitHub Actions:

- `slsa-framework/slsa-github-generator` reusable workflow
- Provenance stored in Sigstore Rekor transparency log
- Build is hermetic and reproducible

## Key Management

| Key                       | Purpose                | Storage        |
| ------------------------- | ---------------------- | -------------- |
| Cosign release key        | Sign release checksums | GitHub Secrets |
| Policy bundle signing key | Sign policy bundles    | HSM / Vault    |
| OIDC identity             | Keyless signing (CI)   | GitHub OIDC    |

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
