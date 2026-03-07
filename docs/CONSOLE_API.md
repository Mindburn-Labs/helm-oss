# Operator Console API

The HELM Operator Console provides a comprehensive REST API for monitoring, managing, and operating HELM deployments. All endpoints require authentication unless explicitly listed as public.

## Authentication

Endpoints are protected by JWT authentication middleware. Public endpoints (health, demo, proof read) bypass auth.

## Endpoint Reference

### Core Operations

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/healthz` | Health check | Public |
| POST | `/v1/tools/execute` | Execute tool via PEP boundary | Public |
| POST | `/v1/chat/completions` | OpenAI-compatible proxy | API Key |

### Receipt & Proof APIs

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/receipts` | List recent receipts | Public |
| GET | `/api/v1/proofgraph` | Query ProofGraph DAG | Public |
| POST | `/api/v1/export` | Export evidence pack | Public |
| GET | `/api/verify/{id}` | Verify receipt by ID | Public |
| POST | `/api/v1/kernel/approve` | Cryptographic HITL approval (Ed25519) | Token |

### Governance & Policy

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/policies` | List active policies | Token |
| POST | `/api/policies/simulate` | Simulate policy evaluation | Token |
| GET | `/api/audit/events` | Query audit event log | Token |
| GET | `/api/vendors` | A2A vendor mesh status | Token |
| POST | `/api/ops/control` | Flight controls (pause/resume/scale) | Token |

### Monitoring & SRE

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/slo/definitions` | SLO definitions | Token |
| GET | `/api/slo/status` | Current SLO compliance status | Token |
| GET | `/api/alerts` | Active alerts | Token |
| POST | `/api/alerts/acknowledge` | Acknowledge alert | Token |
| GET | `/api/connectors/health` | Connector health status | Token |
| GET | `/api/metrics/overview` | Metrics dashboard data | Token |

### Cost & Budget

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/cost/summary` | Cost attribution summary | Token |
| GET | `/api/cost/alerts` | Cost threshold alerts | Token |

### Compliance & Tenants

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/compliance/obligations` | Compliance obligations (JSON) | Token |
| GET | `/api/compliance/sources` | Compliance rule sources | Token |
| GET | `/api/tenants` | Multi-tenant listing | Token |
| GET | `/api/compliance/report` | SOC2 compliance report | Token |

### Registry & Packs

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/registry/list` | List registered packs | Token |
| POST | `/api/registry/install` | Install pack for tenant | Token |
| POST | `/api/registry/publish` | Publish signed pack | Token |
| POST | `/api/registry/anchors` | Register trust anchor | Token |

### Operator Workflows

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET/POST | `/api/runs` | Operator run management | Token |
| GET/POST | `/api/intents` | Intent lifecycle management | Token |
| GET/POST | `/api/approvals` | Approval workflow management | Token |

### Admin (Restricted)

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/admin/tenants` | Tenant CRUD | Admin |
| GET | `/api/admin/roles` | Role management | Admin |
| GET | `/api/admin/audit-ui` | Audit log viewer | Admin |
| POST | `/api/ops/chaos/inject` | Chaos injection (burns error budget) | **Admin only** |

### Onboarding

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/signup` | User registration | Public |
| POST | `/api/onboarding/verify` | Email verification | Public |
| POST | `/api/resend-verification` | Resend verification email | Public |

### Builder & Factory

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/builder` | JIT business plan generation | Token |
| POST | `/api/factory` | Tenant provisioning | Token |

### Generative UI

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/ui/render` | Render generative UI component | Token |
| POST | `/api/ui/interact` | Handle UI interaction | Token |

## Error Responses

All error responses follow a consistent format:

```json
{
  "error": "error message",
  "code": 400
}
```

| Code | Meaning |
|------|---------|
| 400 | Bad Request — invalid parameters |
| 401 | Unauthorized — missing or invalid token |
| 403 | Forbidden — insufficient permissions |
| 404 | Not Found — resource does not exist |
| 405 | Method Not Allowed |
| 429 | Too Many Requests — error budget exhausted |
| 500 | Internal Server Error |

## Rate Limiting

The console uses an error budget model. When the error budget drops below 20%, the builder and promotion endpoints return `429 Too Many Requests` with a `Retry-After` header.
