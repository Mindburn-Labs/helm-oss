# HELM Regional Compatibility

HELM supports regional deployment profiles that automatically configure governance
policies, data residency, encryption standards, and ceremony requirements.

## Supported Regions

| Region | Profile | Compliance | Encryption | Ceremony |
|--------|---------|------------|------------|----------|
| US     | `us`    | SOC2, NIST-800-53 | AES-256-GCM | Standard (2s timelock) |
| EU     | `eu`    | GDPR, SOC2, ISO-27001 | AES-256-GCM | Strict (5s timelock, challenge/response) |
| RU     | `ru`    | GOST-R-34.10, 152-FZ | GOST-28147-89 | Standard (3s timelock) |
| CN     | `cn`    | GB/T-35273, CSL | SM4 | Standard (3s timelock) |

## Configuration

Set the `HELM_REGION` environment variable:

```bash
export HELM_REGION=eu
```

Or in `helm.yaml`:

```yaml
region: eu
```

## EU-Specific Requirements

- **GDPR**: PII handling set to strict mode. All personal data processing
  requires explicit consent, logged as TRUST_EVENT in the ProofGraph.
- **Right to Erasure**: Supported via cryptographic key rotation. Data
  encrypted with tenant keys can be rendered inaccessible by revoking
  the key in the Trust Registry.
- **Data Residency**: All data stored in `eu-west-1`. Cross-region
  replication disabled by default.

## Ceremony Differences

The EU profile requires challenge/response verification for all
approval ceremonies, adding an extra layer of human verification.
This means the operator must type a confirmation phrase (e.g., "DELETE")
in addition to the standard timelock and hold requirements.

## Custom Profiles

Create a custom profile in `config/profiles/`:

```yaml
profiles:
  custom:
    name: "Custom Region"
    ceremony:
      min_timelock_ms: 10000
      min_hold_ms: 5000
      require_challenge: true
      domain_separation: "helm:approval:v1:custom"
    data_residency: "custom-dc-1"
    compliance:
      - "CUSTOM-STANDARD"
    encryption: "AES-256-GCM"
```
