# HELM OpenAPI Contract

## Overview

`helm.openapi.yaml` is the single source of truth for the HELM kernel HTTP API.
All SDKs are generated from this spec. Manual drift is caught by CI.

## Endpoints

| Group | Endpoints | Tag |
|-------|-----------|-----|
| OpenAI Proxy | `POST /v1/chat/completions` | `proxy` |
| Approval | `POST /api/v1/kernel/approve` | `approval` |
| ProofGraph | `GET /api/v1/proofgraph/sessions`, `GET .../receipts` | `proofgraph` |
| Evidence | `POST /api/v1/evidence/export`, `POST .../verify`, `POST /api/v1/replay/verify` | `evidence` |
| Conformance | `POST /api/v1/conformance/run`, `GET .../reports/{id}` | `conformance` |
| System | `GET /healthz`, `GET /version` | `system` |

## Error Envelope

Every error uses the same shape:
```json
{
  "error": {
    "message": "Human-readable message",
    "type": "invalid_request | authentication_error | permission_denied | not_found | internal_error",
    "code": "machine_readable_code",
    "reason_code": "DENY_TOOL_NOT_FOUND",
    "details": {}
  }
}
```

## Reason Codes

See `x-helm-reason-codes` in the spec. All codes are deterministic and stable.

## Versioning

- Contract version matches `info.version` in the YAML
- Breaking changes require a major version bump
- SDK generation runs on every PR via `scripts/ci/sdk_drift_check.sh`
- Drift = CI failure

## Generation

```bash
# Regenerate all SDK types from this spec
bash scripts/sdk/gen.sh

# Check for drift (CI gate)
bash scripts/ci/sdk_drift_check.sh
```
