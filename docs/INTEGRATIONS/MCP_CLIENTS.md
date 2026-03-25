---
title: MCP_CLIENTS
---

# HELM MCP Clients — Integration Guide

Install the HELM MCP server in your AI coding tool for governed tool execution.
MCP remains a transport and capability layer; HELM adds authority, scope, and proof at execution time.

---

## Claude Code (Plugin)

```bash
# Generate the plugin
helm mcp install --client claude-code

# Install it
claude plugin install ./helm-mcp-plugin
```

The plugin bundles a `.mcp.json` that auto-starts the HELM MCP server when enabled. Tool calls are intercepted by the execution kernel, receipted, and can carry organizational scope metadata such as `organization_id`, `scope_id`, and `principal_id`.

---

## Claude Desktop (One-Click .mcpb)

```bash
# Generate the .mcpb bundle
helm mcp pack --client claude-desktop --out helm.mcpb
```

The bundle includes:

- `manifest.json` with `server.type="binary"`
- Cross-platform HELM binary under `server/`
- `platform_overrides` for Windows (`.exe`)

Double-click `helm.mcpb` or drag into Claude Desktop to install.

---

## Windsurf

```bash
helm mcp print-config --client windsurf
```

Output (add to Windsurf settings):

```json
{
  "mcpServers": {
    "helm-governance": {
      "command": "helm",
      "args": ["mcp", "serve", "--transport", "stdio"],
      "transport": "stdio"
    }
  }
}
```

Windsurf supports stdio and remote HTTP transports in the OSS runtime. For remote:

```bash
helm mcp serve --transport http --port 9100
# URL: http://localhost:9100/mcp
```

For bearer-gated remote HTTP:

```bash
HELM_OAUTH_BEARER_TOKEN=testtoken helm mcp serve --transport http --port 9100 --auth oauth
# Discovery: http://localhost:9100/.well-known/oauth-protected-resource/mcp
```

---

## Codex

```bash
helm mcp print-config --client codex
```

```bash
codex mcp add helm-governance -- helm mcp serve --transport stdio
```

Remote HTTP is available at `http://localhost:9100/mcp` when you run `helm mcp serve --transport http --port 9100`.

---

## VS Code

```bash
helm mcp print-config --client vscode
```

Add to `.vscode/settings.json`:

```json
{
  "mcp": {
    "servers": {
      "helm-governance": {
        "command": "helm",
        "args": ["mcp", "serve", "--transport", "stdio"]
      }
    }
  }
}
```

---

## Cursor

```bash
helm mcp print-config --client cursor
```

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "helm-governance": {
      "command": "helm",
      "args": ["mcp", "serve", "--transport", "stdio"]
    }
  }
}
```

---

## Auth

The HELM MCP server supports:

- **None** (default) — local stdio, no auth needed
- **Static headers** — `HELM_API_KEY` env var, `Authorization: Bearer ...` or `X-HELM-API-Key`
- **OAuth bearer discovery** — `HELM_OAUTH_BEARER_TOKEN` for OSS remote HTTP plus `/.well-known/oauth-protected-resource/mcp`

Supported MCP protocol versions are negotiated on `initialize`: `2025-11-25`, `2025-06-18`, `2025-03-26`.

For identity and delegation patterns above raw auth, see [IDENTITY_INTEROP.md](IDENTITY_INTEROP.md).
