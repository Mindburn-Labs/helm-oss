# HELM Benchmark Harness

Measures the incremental latency HELM adds versus direct execution (the `< 5ms p99 overhead` claim).

## What is measured

The hot path is: `Guardian.EvaluateDecision → crypto.SignReceipt → store.Append`

| Scenario | Description |
|----------|-------------|
| `baseline_no_helm` | Mock tool call with no governance — JSON marshal only |
| `helm_allow` | Full governed allow: Guardian eval + Ed25519 sign + SQLite persist |
| `helm_deny` | Undeclared tool → fail-closed deny with signed decision |
| `helm_allow_parallel` | Same as allow under goroutine concurrency |

## Running

```bash
# Full overhead report (10K iterations, JSON output)
make bench-report

# Standard Go benchmarks only
make bench

# Individual component benchmarks
cd core && go test -bench=. -benchmem ./pkg/crypto/
cd core && go test -bench=. -benchmem ./pkg/store/
cd core && go test -bench=. -benchmem ./pkg/guardian/
cd core && go test -bench=. -benchmem ./benchmarks/
```

## Output

`benchmarks/results/latest.json` — machine-readable report:

```json
{
  "helm_version": "0.3.0",
  "go_version": "go1.24.0",
  "go_os": "darwin",
  "go_arch": "arm64",
  "num_cpu": 10,
  "hot_path_p99_us": 245.0,
  "baseline_p99_us": 1.2,
  "overhead_p99_us": 243.8,
  "overhead_under_5ms": true,
  "scenarios": [...]
}
```

## Metrics per scenario

- p50, p95, p99
- mean, standard deviation
- min, max
- environment metadata (Go version, OS, arch, CPU count, commit, timestamp)

## Hard rule

If measured overhead exceeds 5ms p99, the README claim is updated to match reality. Reality wins.
