# Credential Security

HELM uses layered credential protection with AES-256-GCM at-rest encryption and strict access controls.

## Current Architecture

| Layer | Mechanism |
|-------|-----------|
| At-rest encryption | AES-256-GCM with 256-bit derived key |
| Key derivation | PBKDF2-HMAC-SHA256, 600 000 iterations |
| Memory | Credentials zeroed after use via `memguard` |
| Access | Environment variable `HELM_CREDENTIAL_KEY` (bootstrap) |
| Rotation | `RotationManager` with configurable TTL and grace period |

## Key Storage

The credential encryption key is currently stored as an environment variable (`HELM_CREDENTIAL_KEY`). For production deployments, operators **SHOULD** use an external KMS:

### HSM / KMS Integration Path

| Provider | Integration | Status |
|----------|-------------|--------|
| AWS KMS | `aws-sdk-go-v2/service/kms` — use `GenerateDataKey` for envelope encryption | Planned |
| GCP Cloud KMS | `cloud.google.com/go/kms` — use `Encrypt`/`Decrypt` with symmetric keys | Planned |
| Azure Key Vault | `azkeys` SDK — use `WrapKey`/`UnwrapKey` for key wrapping | Planned |
| HashiCorp Vault | Transit secrets engine — use `encrypt`/`decrypt` endpoints | Planned |
| PKCS#11 HSM | `miekg/pkcs11` — direct HSM integration for hardware-backed keys | Planned |

**Envelope encryption pattern**: KMS encrypts a Data Encryption Key (DEK); the DEK encrypts credentials locally. Only the encrypted DEK is stored alongside the ciphertext.

## Key Rotation

The `RotationManager` (in `core/pkg/credentials/rotation.go`) supports:

1. **Automatic rotation** — configurable TTL (default: 90 days)
2. **Grace period** — old key remains valid for decryption during transition (default: 24 hours)
3. **Re-encryption** — all credentials are re-encrypted with the new key during rotation
4. **Audit trail** — rotation events are logged to the Guardian audit log

### Rotation Procedure

```bash
# 1. Generate new key
export HELM_CREDENTIAL_KEY_NEW=$(openssl rand -hex 32)

# 2. Trigger rotation (re-encrypts all stored credentials)
helm credentials rotate --new-key-env HELM_CREDENTIAL_KEY_NEW

# 3. Update environment
export HELM_CREDENTIAL_KEY=$HELM_CREDENTIAL_KEY_NEW
unset HELM_CREDENTIAL_KEY_NEW
```

## Recommendations

> [!IMPORTANT]
> For production deployments handling sensitive data, use an external KMS rather than environment variables for the master encryption key.

- **Development**: Environment variable is acceptable
- **Staging**: Use cloud KMS with IAM-scoped access
- **Production**: Use cloud KMS or HSM with audit logging and automatic rotation
