# Chaos Engineering & Testing Scenarios (TEST-004)

## Overview

HELM chaos testing validates that the kernel degrades gracefully under adversarial conditions, maintaining security invariants (fail-closed, receipt generation, budget enforcement).

## Scenarios

### 1. Error Budget Exhaustion

**Goal**: Verify that builder/promotion endpoints return 429 when budget drops below 20%.

```bash
# Inject chaos (burns 30% budget per call)
curl -X POST http://localhost:8080/api/ops/chaos/inject \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Repeat until budget < 20%
curl -X POST http://localhost:8080/api/ops/chaos/inject \
  -H "Authorization: Bearer $ADMIN_TOKEN"
curl -X POST http://localhost:8080/api/ops/chaos/inject \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Verify builder is gated
curl -X POST http://localhost:8080/api/builder \
  -H "Content-Type: application/json" \
  -d '{"idea":"test","industry":"tech"}'
# Expected: 429 Too Many Requests with X-Gate-Enforced: BudgetExhausted
```

### 2. Signer Unavailable

**Goal**: Verify Guardian returns BLOCK verdict when signer is unavailable.

```go
// Test with nil signer
g := guardian.NewGuardian(nil, prg, registry)
decision, err := g.EvaluateDecision(ctx, req)
// Expected: err != nil (fail-closed)
```

### 3. Clock Drift

**Goal**: Verify that expired intents are rejected even with clock manipulation.

```go
// Inject a clock that returns past time
exec := NewSafeExecutor(dispatchers, guardian, proofgraph)
exec = exec.WithClock(func() time.Time {
    return time.Now().Add(-2 * time.Hour) // 2 hours in the past
})

// Execute with a valid intent — should detect expiration
result, err := exec.Execute(ctx, intent)
// Expected: err contains "expired"
```

### 4. Concurrent Storm

**Goal**: Verify kernel handles 1000 concurrent tool calls without race conditions.

```bash
# Run race detector with concurrent access
cd core && go test ./pkg/guardian/... -race -count=1 -timeout 120s
cd core && go test ./pkg/executor/... -race -count=1 -timeout 120s
cd core && go test ./pkg/firewall/... -race -count=1 -timeout 120s
```

### 5. Policy Graph Mutation

**Goal**: Verify that mutating the PRG after Guardian creation doesn't affect in-flight decisions.

```go
g := guardian.NewGuardian(signer, prg, registry)
// Modify PRG after creation
prg.AddRule("dangerous-tool", RequirementSet{...})
// Guardian should still use original snapshot
```

### 6. Malformed Input

**Goal**: Verify kernel rejects malformed JSON, oversized payloads, and invalid schemas.

```bash
# Malformed JSON
curl -X POST http://localhost:8080/v1/tools/execute \
  -H "Content-Type: application/json" \
  -d '{invalid json'
# Expected: 400 Bad Request

# Oversized payload (>1MB)
python3 -c "print('{\"data\":\"' + 'A'*2000000 + '\"}')" | \
  curl -X POST http://localhost:8080/v1/tools/execute \
  -H "Content-Type: application/json" \
  -d @-
# Expected: 400 or 413
```

## Automation

All chaos scenarios are automated via the race detector (`go test -race`) and the `TestPolicyFirewall_ConcurrentAccess` test in the firewall package.
