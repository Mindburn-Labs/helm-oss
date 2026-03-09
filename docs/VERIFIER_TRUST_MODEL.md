# Offline Verifier Trust Model
The HELM Offline Verifier is a standalone tool designed for high-assurance audits and air-gapped verification of EvidencePacks.

## Trust Assumptions
1. **Cryptographic Primitives**: The verifier trusts the mathematical correctness of Ed25519 signatures, SHA-256 hashes, and JCS (RFC 8785) canonicalization.
2. **Standard Compliance**: The verifier assumes the EvidencePack format adheres to the UCS v1.2 specification.
3. **No Network Trust**: The verifier does NOT require network access and does NOT trust results from the HELM server or any proxy.

## Verification Layers
1. **Structural Integrity**: Ensures the bundle contains required indices and manifests.
2. **Content Integrity**: Verifies that every file matches its hash in the signed manifest.
3. **Chain Integrity**: Validates the causal DAG (ProofGraph) and prevents reordering or deletion of events.
4. **Temporal Integrity**: Checks Lamport clock monotonicity across the event stream.
5. **Policy Binding**: Recomputes policy hashes to ensure the Kernel applied the correct rules.

## Auditor Mode
Using the `--json` flag, the verifier produces a machine-readable report containing every check performed, suitable for inclusion in formal compliance artifacts.