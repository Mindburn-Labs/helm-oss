# HELM MCP Clients — Integration Guide

Install the HELM MCP server in your AI coding tool for governed tool execution.

---

## Claude Code (Plugin)

```bash
# Generate the plugin
helm mcp install --client claude-code

# Install it
claude plugin install ./helm-mcp-plugin
```

The plugin bundles a `.mcp.json` that auto-starts the HELM MCP server when enabled. Tool calls are intercepted by the GovernanceFirewall and receipted.

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

---

## Codex

```bash
helm mcp print-config --client codex
```

```bash
codex mcp add helm-governance -- helm mcp serve --transport stdio
```

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
- **Static headers** — `HELM_API_KEY` env var
- **OAuth** — not implemented in the OSS runtime
