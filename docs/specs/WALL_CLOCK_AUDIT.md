---
title: WALL_CLOCK_AUDIT
---

# HELM Wall-Clock Contamination Audit

> Audit of `time.Now()` usage in proof-bearing paths.

## Scope

Proof-bearing paths are code paths that produce receipts, decisions, or
evidence that form part of the verifiable governance chain. These MUST use
deterministic or injected time sources — never `time.Now()` directly.

## Audit Results

### `core/pkg/kernel/` — Critical Path

| File                              | Line                                   | Usage                                                                        | Verdict |
| --------------------------------- | -------------------------------------- | ---------------------------------------------------------------------------- | ------- |
| `event_log.go:89`                 | `event.CommittedAt = time.Now().UTC()` | **CONTAMINATED** — event timestamps in proof chain should use injected clock |
| `concurrency_test.go:26,33`       | `time.Now().UnixMilli()`               | OK — test-only, synthetic data                                               |
| `concurrency_test.go:127`         | `time.Now()`                           | OK — test-only, measuring elapsed time                                       |
| `swarm_orchestrator_test.go:75`   | `time.Now()`                           | OK — test-only                                                               |
| `io_capture_test.go:17`           | `time.Now()`                           | OK — test-only                                                               |
| `error_ir_test.go:284`            | `time.Now()`                           | OK — test-only                                                               |
| `cybernetics_test.go:190,206,207` | `time.Now()`                           | OK — test-only                                                               |

### `core/pkg/guardian/` — Governance Path

```bash
# Scan for time.Now in guardian package
grep -rn 'time\.Now()' core/pkg/guardian/ --include='*.go'
```

Results pending — requires deeper audit of guardian, governance, and receipt generation code.

## Remediation Strategy

### Option 1: Clock Interface (Recommended)

```go
type Clock interface {
    Now() time.Time
}

type SystemClock struct{}
func (SystemClock) Now() time.Time { return time.Now().UTC() }

type FixedClock struct{ T time.Time }
func (c FixedClock) Now() time.Time { return c.T }
```

Inject `Clock` into all proof-bearing components:

- `EventLog`
- `ReceiptGenerator`
- `ProofGraph`
- `DecisionRecord` builders

### Option 2: Package-Level Override

```go
var nowFunc = time.Now

func setNowFunc(f func() time.Time) { nowFunc = f }
```

Less clean but faster to implement.

## Priority

- **event_log.go:89** — P0. This contaminates every event timestamp.
- Guardian/governance paths — P1. Audit and fix individually.
- All test files — OK. No action needed.

## Related

- `INVARIANTS.md` — MonotonicLamport invariant depends on logical clocks, not wall clocks
- `receipt-format-v1.md` — Receipt `produced_at` MUST be deterministic in replay
