# try.mindburn.run — Hosted Governed Agent (Free Tier)

**Use a HELM-governed AI assistant for free. No signup, no install.**

Every action is gated by HELM policy enforcement. Every decision produces a cryptographic receipt (Ed25519). Export your receipts and verify offline.

## What You Get

| Feature | Limit |
|---------|-------|
| AI model | Llama 3.1 8B (via OpenRouter free tier) |
| Requests | 50/day per session |
| Tokens | 10,000 per request |
| Receipts | ✅ Exportable, offline-verifiable |
| Governance | ✅ Fail-closed HELM policy |
| Cost | Free |

## Architecture

```
User Browser
     │
     ▼
  Caddy (TLS + rate limit)
     │
     ├─→ OpenClaw Agent (chat UI)
     │        │
     │        ▼
     │   HELM Proxy (base_url rewrite)
     │        │
     │   Guardian (policy: allow/deny)
     │        │
     │   Ed25519 Receipt
     │        │
     │        ▼
     └── OpenRouter (free models)
```

## Deploy

```bash
# Prerequisites: Docker, OpenRouter API key (free)
export OPENROUTER_API_KEY="sk-or-..."

# Deploy
docker compose -f docker-compose.try.yml up -d

# Verify
curl -s https://try.mindburn.run/health | jq .
```

## DNS

Point `try.mindburn.run` to your VPS IP. Caddy auto-provisions TLS via Let's Encrypt.

## Upgrade Path

| Need | Solution |
|------|----------|
| More requests / tokens | → HELM Managed Hosting on `helm.mindburn.run` |
| Bring your own models | → Self-hosted with helm-oss Agent Hardening Kit |
| Private tenant | → HELM Hardening Sprint (managed engagement) |

## Security

- **HELM budget enforcement** — not application-level rate limiting. Every DENY is a kernel-level guarantee with a signed receipt.
- **Caddy rate limiting** — 20 req/min per IP at the edge.
- **No persistent storage** — sessions are ephemeral. Evidence is exportable but not stored long-term.
- **Free models only** — policy enforces free-tier model attestation.

## Files

| File | Purpose |
|------|---------|
| `docker-compose.try.yml` | Full stack: Caddy + HELM + OpenClaw |
| `Caddyfile` | TLS + rate limiting + CORS |
| `policy-free-tier.yml` | HELM free-tier policy (50 req/day, 10K tokens/req) |
