---
title: AUTH_MATRIX
---

# HELM Auth Matrix

> Authentication and authorization modes for HELM integrations.

## Supported Auth Modes

| Mode                 | Use Case                | Config Key              | Status      |
| -------------------- | ----------------------- | ----------------------- | ----------- |
| API Key (provider)   | Direct LLM API calls    | `HELM_API_KEY`          | ✅ Shipping |
| Google OAuth         | Gemini CLI / ADK demos  | `HELM_GOOGLE_OAUTH`     | Planned     |
| Client-local session | Desktop/CLI passthrough | `HELM_SESSION_TOKEN`    | ✅ Shipping |
| MCP header auth      | MCP protocol auth       | `Authorization` header  | ✅ Shipping |
| MCP OAuth bearer     | Remote HTTP MCP         | `HELM_OAUTH_BEARER_TOKEN` | ⚠️ Preview |
| Enterprise token     | SSO/IdP integration     | `HELM_ENTERPRISE_TOKEN` | Planned     |
| mTLS                 | Service-to-service      | TLS config block        | Planned     |

## Configuration

### API Key Mode

```yaml
auth:
  mode: api_key
  provider_key_env: OPENAI_API_KEY # or ANTHROPIC_API_KEY, etc.
```

### Google OAuth Mode (Gemini CLI)

```yaml
auth:
  mode: google_oauth
  client_id: ${HELM_GOOGLE_CLIENT_ID}
  scopes:
    - https://www.googleapis.com/auth/generative-language
```

### MCP Header Auth

```yaml
auth:
  mode: mcp_header
  header_name: Authorization
  token_env: HELM_MCP_TOKEN
```

### MCP OAuth Bearer (OSS remote HTTP)

```yaml
auth:
  mode: mcp_oauth
  resource_metadata: http://localhost:9100/.well-known/oauth-protected-resource/mcp
  token_env: HELM_OAUTH_BEARER_TOKEN
```

### Enterprise Token

```yaml
auth:
  mode: enterprise
  idp_url: https://auth.example.com
  client_id: helm-agent
  scopes: [governance:read, governance:write]
```

## Auth Flow per Client

| Client         | Primary Auth | Fallback         |
| -------------- | ------------ | ---------------- |
| Claude Code    | MCP header   | API key          |
| Claude Desktop | MCP header   | API key          |
| Gemini CLI     | Google OAuth | API key          |
| Cursor         | MCP header / OAuth bearer | API key |
| VS Code        | MCP header / OAuth bearer | API key |
| Direct SDK     | API key      | Enterprise token |

## Security Requirements

1. Keys MUST NOT be logged or included in receipts
2. Auth errors MUST NOT leak key material
3. Token refresh MUST be automatic for OAuth/enterprise modes
4. All auth modes MUST work with fail-closed governance

## Auth Smoke Tests

Each auth mode MUST pass the following automated smoke tests:

### API Key Mode

```bash
# Verify: API key flows through → effect evaluated → receipt issued
HELM_API_KEY=test-key helm smoke-test --auth api_key --expect verdict:ALLOW
# Verify: Missing key → fail-closed DENY
helm smoke-test --auth api_key --expect verdict:DENY --expect reason:PDP_ERROR
```

### Google OAuth Mode

```bash
# One-command Gemini CLI demo via Google OAuth
helm demo gemini --auth google_oauth --client-id $HELM_GOOGLE_CLIENT_ID
# One-command Gemini CLI demo via API key
GEMINI_API_KEY=test helm demo gemini --auth api_key
```

### MCP Header Mode

```bash
# Verify: Valid token → ALLOW
helm smoke-test --auth mcp_header --token test-token --expect verdict:ALLOW
# Verify: Invalid token → DENY
helm smoke-test --auth mcp_header --token invalid --expect verdict:DENY
```

### Smoke Test Output

Each smoke test MUST produce:

- A valid receipt (verifiable offline)
- A pass/fail status
- Auth mode exercised in metadata

### CI Integration

```yaml
# .github/workflows/auth-smoke.yml
jobs:
  auth-smoke:
    steps:
      - run: make build
      - run: helm smoke-test --auth api_key
      - run: helm smoke-test --auth mcp_header
```
