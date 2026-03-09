# Observability Templates

## OBS-001: Structured Logging with Trace Correlation

HELM uses Go's `log/slog` for structured logging with OpenTelemetry trace ID correlation.

### Configuration

```go
import (
    "log/slog"
    "go.opentelemetry.io/otel/trace"
)

// Create a handler that adds trace_id to every log entry
type TracedHandler struct {
    slog.Handler
}

func (h *TracedHandler) Handle(ctx context.Context, r slog.Record) error {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        r.AddAttrs(
            slog.String("trace_id", span.SpanContext().TraceID().String()),
            slog.String("span_id", span.SpanContext().SpanID().String()),
        )
    }
    return h.Handler.Handle(ctx, r)
}
```

### Log Fields Convention

| Field | Type | Description |
|-------|------|-------------|
| `trace_id` | string | OpenTelemetry trace ID (auto-injected) |
| `span_id` | string | Span ID within trace |
| `tenant_id` | string | Tenant isolation boundary |
| `decision_id` | string | Guardian decision identifier |
| `receipt_id` | string | Execution receipt reference |
| `tool_id` | string | Tool being executed |
| `latency_ms` | float64 | Operation latency in milliseconds |
| `error` | string | Error message if applicable |

---

## OBS-002: Grafana / Prometheus Dashboard Templates

### Prometheus Metrics (exposed at `:9090/metrics`)

```yaml
# Guardian decision metrics
helm_guardian_decisions_total{verdict="ALLOW|DENY|ESCALATE"}
helm_guardian_decision_duration_seconds{quantile="0.5|0.9|0.99"}

# Executor metrics
helm_executor_tool_calls_total{tool_id, status="ok|error"}
helm_executor_tool_duration_seconds{tool_id}

# ProofGraph metrics
helm_proofgraph_nodes_total{type="DECISION|EXECUTION|EVIDENCE"}
helm_proofgraph_chain_length

# Budget metrics
helm_budget_remaining{tenant_id}
helm_budget_consumed_total{tenant_id, tool_id}

# Evidence metrics
helm_evidence_packs_exported_total
helm_evidence_verification_total{result="pass|fail"}
```

### Grafana Dashboard JSON (import via Grafana UI)

Save as `grafana-helm-dashboard.json` and import:

```json
{
  "dashboard": {
    "title": "HELM Kernel Overview",
    "panels": [
      {
        "title": "Decision Throughput",
        "type": "timeseries",
        "targets": [{"expr": "rate(helm_guardian_decisions_total[5m])"}]
      },
      {
        "title": "Decision Latency (p99)",
        "type": "gauge",
        "targets": [{"expr": "histogram_quantile(0.99, helm_guardian_decision_duration_seconds)"}]
      },
      {
        "title": "Tool Call Rate by Tool",
        "type": "timeseries",
        "targets": [{"expr": "rate(helm_executor_tool_calls_total[5m])"}]
      },
      {
        "title": "Error Budget Remaining",
        "type": "gauge",
        "targets": [{"expr": "helm_budget_remaining"}]
      },
      {
        "title": "ProofGraph Growth",
        "type": "timeseries",
        "targets": [{"expr": "helm_proofgraph_nodes_total"}]
      },
      {
        "title": "Evidence Verification Success Rate",
        "type": "stat",
        "targets": [{"expr": "helm_evidence_verification_total{result='pass'} / helm_evidence_verification_total"}]
      }
    ]
  }
}
```

### AlertManager Rules

```yaml
groups:
  - name: helm-kernel
    rules:
      - alert: GuardianLatencyHigh
        expr: histogram_quantile(0.99, helm_guardian_decision_duration_seconds) > 0.005
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Guardian p99 latency exceeds 5ms SLA"

      - alert: BudgetExhausted
        expr: helm_budget_remaining < 20
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Error budget below 20% — builder/promotion gates activated"

      - alert: EvidenceVerificationFailure
        expr: rate(helm_evidence_verification_total{result="fail"}[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Evidence pack verification failures detected — potential tampering"
```
