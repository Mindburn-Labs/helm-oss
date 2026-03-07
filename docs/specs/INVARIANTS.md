# HELM Safety Invariants — TLA+ to Implementation Mapping

> Map each TLA+ invariant to its Go implementation and test coverage.

## Source

- **TLA+ Spec**: [`HelmKernel.tla`](file:///Users/ivan/Code/Mindburn-Labs/helm-oss/protocols/specs/tla/HelmKernel.tla)
- **TLC Config**: [`HelmKernel.cfg`](file:///Users/ivan/Code/Mindburn-Labs/helm-oss/protocols/specs/tla/HelmKernel.cfg)
- **CI**: [`.github/workflows/tla-check.yml`](file:///Users/ivan/Code/Mindburn-Labs/helm-oss/.github/workflows/tla-check.yml)

## Invariant Matrix

| #   | TLA+ Invariant        | Safety Property                                       | Go Implementation                                                                                                         | Test Coverage                                                                    |
| --- | --------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ----------------------------------------- | ------------------------------------------- |
| S1  | `FailClosed`          | No execution without a valid ALLOW decision           | `kernel.EffectBoundary.Submit` — returns `DENY` if PDP does not ALLOW; `executor.go` checks AEI validity before execution | `effect_boundary_test.go:TestDenyWithoutPolicy`, `kernel_test.go:TestFailClosed` |
| S2  | `MonotonicLamport`    | ProofGraph Lamport clock is strictly monotonic        | `proofgraph.go:Append` — increments and validates `lamport > prev.lamport`                                                | `proofgraph_test.go:TestLamportMonotonicity`                                     |
| S3  | `ReceiptCompleteness` | Every execution produces a receipt                    | `executor.go:Execute` — always calls `receipt.Issue` on completion; fail path issues DENY receipt                         | `executor_test.go:TestReceiptAlwaysIssued`                                       |
| S4  | `PrincipalBinding`    | Every ProofGraph node is bound to a principal         | `proofgraph.go:Append` — validates `node.Principal != ""`                                                                 | `proofgraph_test.go:TestPrincipalRequired`                                       |
| S5  | `HashChainIntegrity`  | Every non-genesis node references valid parent hashes | `proofgraph.go:Append` — computes `SHA-256(parent_hashes                                                                  |                                                                                  | content)` and verifies parent nodes exist | `proofgraph_test.go:TestHashChainIntegrity` |
| S6  | `TypeInvariant`       | All state variables satisfy their type constraints    | Enforced by Go type system (struct fields, `Verdict` enum, `uint64` Lamport)                                              | Compilation-time guarantee                                                       |

## Verification Strategy

### Model Checking (TLC)

TLC exhaustively explores all reachable states under the finite model
(2 principals × 2 tools × 3 actions × lamport < 6). Any invariant
violation surfaces as a CI failure with a counterexample trace.

### Implementation Testing

Each invariant maps to at least one Go test that:

1. Sets up the precondition
2. Exercises the state transition
3. Asserts the invariant holds

### Cross-Validation

The conformance vectors in `protocols/conformance/v1/test-vectors.json`
include golden receipts and decisions that exercise each invariant's
boundary conditions, enabling third-party implementations to verify
they satisfy the same properties.

## Adding New Invariants

1. Define the invariant in `HelmKernel.tla`
2. Add `INVARIANT <name>` to `HelmKernel.cfg`
3. Implement the enforcement in Go
4. Add a test that specifically exercises the invariant
5. Add a conformance vector for the invariant boundary case
6. Update this document
