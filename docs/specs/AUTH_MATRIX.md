# HELM Auth Matrix

> Authentication and authorization modes for HELM integrations.

## Supported Auth Modes

| Mode                 | Use Case                | Config Key              | Status      |
| -------------------- | ----------------------- | ----------------------- | ----------- |
| API Key (provider)   | Direct LLM API calls    | `HELM_API_KEY`          | ✅ Shipping |
| Google OAuth         | Gemini CLI / ADK demos  | `HELM_GOOGLE_OAUTH`     | Planned     |
| Client-local session | Desktop/CLI passthrough | `HELM_SESSION_TOKEN`    | ✅ Shipping |
| MCP header auth      | MCP protocol auth       | `Authorization` header  | ✅ Shipping |
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
| Cursor         | MCP header   | API key          |
| VS Code        | MCP header   | API key          |
| Direct SDK     | API key      | Enterprise token |

## Security Requirements

1. Keys MUST NOT be logged or included in receipts
2. Auth errors MUST NOT leak key material
3. Token refresh MUST be automatic for OAuth/enterprise modes
4. All auth modes MUST work with fail-closed governance
