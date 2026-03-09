# HELM SDK — Python

Typed Python client for the HELM kernel API. One dependency: `httpx`.

## Install

```bash
pip install helm-sdk
```

## Quick Example

```python
from helm_sdk import HelmClient, HelmApiError, ChatCompletionRequest, ChatMessage

helm = HelmClient(base_url="http://localhost:8080")

# OpenAI-compatible chat (tool calls governed by HELM)
try:
    res = helm.chat_completions(ChatCompletionRequest(
        model="gpt-4",
        messages=[ChatMessage(role="user", content="List files in /tmp")],
    ))
    print(res.choices[0].message.content)
except HelmApiError as e:
    print(f"Denied: {e.reason_code}")  # e.g. DENY_TOOL_NOT_FOUND

# Export + verify evidence pack
pack = helm.export_evidence()
result = helm.verify_evidence(pack)
print(result.verdict)  # PASS

# Conformance
from helm_sdk import ConformanceRequest
conf = helm.conformance_run(ConformanceRequest(level="L2"))
print(conf.verdict, conf.gates, "gates")
```

## API

| Method | Endpoint |
|--------|----------|
| `chat_completions(req)` | `POST /v1/chat/completions` |
| `approve_intent(req)` | `POST /api/v1/kernel/approve` |
| `list_sessions()` | `GET /api/v1/proofgraph/sessions` |
| `get_receipts(session_id)` | `GET /api/v1/proofgraph/sessions/{id}/receipts` |
| `export_evidence(session_id?)` | `POST /api/v1/evidence/export` |
| `verify_evidence(bundle)` | `POST /api/v1/evidence/verify` |
| `replay_verify(bundle)` | `POST /api/v1/replay/verify` |
| `conformance_run(req)` | `POST /api/v1/conformance/run` |
| `health()` | `GET /healthz` |
| `version()` | `GET /version` |

## Error Handling

All errors raise `HelmApiError` with a typed `reason_code`:
```python
try: helm.chat_completions(req)
except HelmApiError as e: print(e.reason_code)
```
