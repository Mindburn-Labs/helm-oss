# Security Policy

## Reporting a Vulnerability

**DO NOT** open a public issue for security vulnerabilities.

Email: **security@mindburn.org**

You will receive acknowledgment within 48 hours and a detailed response within 7 days.

## Security Model

HELM is a **fail-closed execution kernel**. The security model assumes:

- **The model is untrusted.** Models propose; the kernel disposes.
- **Tool inputs are untrusted.** Every tool call is schema-validated, canonicalized (JCS/RFC 8785), and hash-bound before execution.
- **Tool outputs are untrusted.** Connector outputs are validated against pinned schemas. Contract drift is a hard error.
- **Untrusted code is sandboxed.** WASI execution has deny-by-default capabilities (no FS, no network) with gas, time, and memory budgets.
- **History is immutable.** Every execution produces a signed receipt linked in a ProofGraph DAG with Lamport clocks.

## What HELM Stops

| Attack | Defense |
|--------|---------|
| Prompt injection → unauthorized tool call | Guardian policy engine blocks undeclared tools |
| Argument tampering | JCS canonicalization + SHA-256 hash binding |
| Output spoofing by malicious connector | Pinned output schema validation (fail-closed) |
| Resource exhaustion via WASM | Gas/time/memory budgets with deterministic traps |
| Receipt forgery | Ed25519 signatures on canonical payloads |
| Replay attacks | Lamport clock monotonicity + causal PrevHash chain |
| Approval bypass | Timelock + deliberate confirmation hash + domain separation |

## What HELM Does NOT Stop

- Prompt injection that stays within the text/conversation domain (HELM governs execution, not generation)
- Vulnerabilities in upstream LLM providers
- Side-channel attacks on the host OS
- Social engineering of human approvers

## TCB (Trusted Computing Base)

The kernel TCB is 8 packages. See `docs/TCB_POLICY.md`.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅        |

## Disclosure Timeline

We follow coordinated disclosure with a 90-day window.
