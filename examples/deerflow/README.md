# DeerFlow Integration — HELM Hardening Quickstart

Harden a [DeerFlow](https://github.com/bytedance/deer-flow) research pipeline with fail-closed HELM governance in under 5 minutes.

## What You Get

- Every DeerFlow tool call is gated through HELM's policy engine
- Cryptographic receipts (Ed25519) for every ALLOW/DENY decision
- Offline-verifiable EvidencePacks for the entire research session
- Budget enforcement — prevent runaway LLM spend

## Architecture

```
DeerFlow ReAct Loop
       │
       ▼
  HELM Proxy (base_url rewrite)
       │
       ├─→ Guardian (policy: allow/deny)
       │        │
       │   Ed25519 Receipt
       │
       ▼
  Original LLM Provider
```

## Prerequisites

- Docker & Docker Compose
- An LLM API key (OpenAI, Anthropic, etc.)

## Quick Start

```bash
# 1. Clone helm-oss (if you haven't)
git clone https://github.com/Mindburn-Labs/helm-oss.git
cd helm-oss/examples/deerflow

# 2. Set your API key
cp .env.example .env
# Edit .env with your LLM provider key

# 3. Start HELM + DeerFlow
docker compose up -d

# 4. Run a governed research query
curl -s http://localhost:3000/api/research \
  -H 'Content-Type: application/json' \
  -d '{"query": "Latest advances in transformer architecture"}' | jq .

# 5. Verify receipts
cd ../.. && helm verify --bundle ./data/evidence
```

## How It Works

The adapter rewrites DeerFlow's LLM `base_url` to point at the HELM proxy. No code changes to DeerFlow itself — just a config override:

```yaml
# DeerFlow config override
llm:
  base_url: "http://helm:8080/v1"   # ← HELM proxy, not direct LLM
  api_key: "${LLM_API_KEY}"
```

HELM intercepts every tool call, applies policy, and issues a signed receipt before forwarding to the real LLM provider.

## Files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | One-command stack: HELM + DeerFlow |
| `deerflow-config.yml` | DeerFlow config with HELM proxy override |
| `.env.example` | Environment template |
| `README.md` | This file |

## Next Steps

- View live proof → [mindburn.org/lab/status](https://mindburn.org/lab/status)
- Compatibility matrix → [mindburn.org/lab/compatibility](https://mindburn.org/lab/compatibility)
- Full HELM docs → [README](../../README.md)
