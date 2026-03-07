# HELM Client Ecosystem Packaging

> MCP server configuration and installation for every major AI client.

## Supported Clients

| Client            | Install Method | CLI Command                                             | Auth Mode    |
| ----------------- | -------------- | ------------------------------------------------------- | ------------ |
| Claude Code       | Plugin install | `helm mcp install --client claude-code`                 | MCP header   |
| Claude Desktop    | .mcpb bundle   | `helm mcp pack --client claude-desktop --out helm.mcpb` | MCP header   |
| Cursor            | Print config   | `helm mcp print-config --client cursor`                 | MCP header   |
| VS Code (Copilot) | Print config   | `helm mcp print-config --client vscode`                 | MCP header   |
| Windsurf          | Print config   | `helm mcp print-config --client windsurf`               | MCP header   |
| Codex CLI         | Print config   | `helm mcp print-config --client codex`                  | MCP header   |
| Gemini CLI        | Print config   | (manual — see below)                                    | Google OAuth |

## Quick Start

```bash
# Claude Code — generates plugin directory with plugin.json + .mcp.json
helm mcp install --client claude-code

# Claude Desktop — generates .mcpb zip bundle
helm mcp pack --client claude-desktop --out helm.mcpb

# Cursor / VS Code / Windsurf / Codex — prints config snippet to stdout
helm mcp print-config --client cursor
helm mcp print-config --client vscode
helm mcp print-config --client windsurf
helm mcp print-config --client codex
```

---

## Client Configurations

### Claude Code

```bash
helm mcp install --client claude-code
# Generates: helm-mcp-plugin/{plugin.json, .mcp.json}
# Install:   claude plugin install ./helm-mcp-plugin
```

The MCP server auto-starts when the plugin is enabled:

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

### Claude Desktop

```bash
helm mcp pack --client claude-desktop --out helm.mcpb
# Double-click or drag helm.mcpb into Claude Desktop to install.
```

---

### Cursor

```bash
helm mcp print-config --client cursor >> .cursor/mcp.json
```

Output:

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

### VS Code with GitHub Copilot

```bash
helm mcp print-config --client vscode
# Add the output to .vscode/settings.json
```

Output:

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

### Windsurf

```bash
helm mcp print-config --client windsurf
```

Output:

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

For remote HTTP mode:

```bash
helm mcp serve --transport http --port 9100
# URL: http://localhost:9100/mcp
```

---

### Codex CLI

```bash
helm mcp print-config --client codex
# Output: codex mcp add helm-governance -- helm mcp serve --transport stdio
```

---

### Gemini CLI

Manual configuration — add to `~/.gemini/settings.json`:

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

### Generic MCP Client

For any MCP-compatible client not listed above, the HELM MCP server
uses standard stdio transport:

```bash
# Start server directly:
helm mcp serve --transport stdio

# Or start as HTTP server:
helm mcp serve --transport http --port 9100
```

All clients use the same underlying command: `helm mcp serve --transport stdio`.
The `print-config` and `install` commands are convenience wrappers that
generate the correct config format for each client.

---

## .mcpb Package Format

HELM ships as a `.mcpb` (MCP Bundle) for easy distribution:

```
helm.mcpb (zip archive)
├── manifest.json       # Package metadata (manifest_version, server config)
└── server/
    └── helm            # Binary (platform-specific)
```

Generate with:

```bash
helm mcp pack --client claude-desktop --out helm.mcpb
```

For cross-platform bundles, build for each target OS/arch and include
all binaries in `server/` with `platform_overrides` in the manifest.
