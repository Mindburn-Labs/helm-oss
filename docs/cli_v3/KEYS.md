# HELM Signing Keys — Key Rotation Policy

## Active Keys

| Key ID | Algorithm | Since | Status |
|--------|-----------|-------|--------|
| `helm-release-2026-v1` | Ed25519 | 2026-02-21 | **Active** |

## Key Lifecycle

1. **Generation**: Keys are generated offline using `openssl genpkey -algorithm Ed25519`.
2. **Pinning**: Public keys are embedded in the CLI source at `src/crypto.ts`.
3. **Rotation**: New keys are appended. Old keys remain for back-compat verification.
4. **Retirement**: Keys are marked inactive after 12 months. They continue to verify existing signatures.

## Verification

The CLI automatically tries all pinned keys in order when verifying an attestation signature.
No configuration is needed — the key list is shipped with each CLI release.

## Where Keys Live

| Location | Content |
|----------|---------|
| `packages/mindburn-helm-cli/src/crypto.ts` | Pinned public keys (PEM format) |
| GitHub Secrets (`HELM_SIGNING_KEY`) | Private key (never committed) |
| `scripts/release/build-evidence-bundle.sh` | Uses `HELM_SIGNING_KEY` env var for signing |

## Key Generation

```bash
# Generate Ed25519 keypair
openssl genpkey -algorithm Ed25519 -out private.pem
openssl pkey -in private.pem -pubout -out public.pem

# Extract the base64 line for pinning
grep -v "BEGIN\|END" public.pem
```

## Emergency Rotation

If a private key is compromised:

1. Generate new keypair
2. Append new public key to `PINNED_PUBLIC_KEYS` in `crypto.ts`
3. Update GitHub Secrets
4. Publish CLI patch release
5. Mark old key as `compromised` in this document
6. Re-sign any attestations created with the compromised key
