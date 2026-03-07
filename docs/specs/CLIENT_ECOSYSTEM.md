# HELM Client Ecosystem Packaging

> Configuration generators and installation guides for every major AI client.

## Supported Clients

| Client            | Install Method | Config Format | Auth Mode    |
| ----------------- | -------------- | ------------- | ------------ |
| Claude Code       | MCP config     | JSON          | MCP header   |
| Claude Desktop    | MCP config     | JSON          | MCP header   |
| Gemini CLI        | Extension      | YAML          | Google OAuth |
| Cursor            | MCP config     | JSON          | MCP header   |
| Windsurf          | MCP config     | JSON          | MCP header   |
| VS Code (Copilot) | Extension      | JSON          | MCP header   |
| Continue.dev      | MCP config     | JSON          | MCP header   |
| Zed               | Extension      | JSON          | MCP header   |

## Quick Config Generator

```bash
# Generate config for a specific client
helm config generate --client claude-code
helm config generate --client cursor
helm config generate --client gemini-cli

# Generate .mcpb package
helm config generate --format mcpb --output helm.mcpb
```

---

## Client Configurations

### Claude Code / Claude Desktop

```json
{
  "mcpServers": {
    "helm": {
      "command": "helm",
      "args": ["mcp-server", "--mode=governance"],
      "env": {
        "HELM_POLICY_DIR": "~/.helm/policies",
        "HELM_LOG_LEVEL": "info"
      }
    }
  }
}
```

**Install**:

```bash
# Install HELM
curl -fsSL https://helm.sh/install.sh | sh

# Generate Claude config
helm config generate --client claude-code >> ~/.claude/mcp.json
```

---

### Gemini CLI

```yaml
extensions:
  helm:
    command: helm
    args: [mcp-server, --mode=governance]
    auth:
      type: google_oauth
```

**Install**:

```bash
helm config generate --client gemini-cli >> ~/.gemini/extensions.yaml
```

---

### Cursor

```json
{
  "mcpServers": {
    "helm": {
      "command": "helm",
      "args": ["mcp-server"],
      "env": {
        "HELM_POLICY_DIR": "./policies"
      }
    }
  }
}
```

**Install**:

```bash
helm config generate --client cursor >> .cursor/mcp.json
```

---

### VS Code with GitHub Copilot

```json
{
  "mcp": {
    "servers": {
      "helm": {
        "command": "helm",
        "args": ["mcp-server"]
      }
    }
  }
}
```

---

### Windsurf

```json
{
  "mcpServers": {
    "helm": {
      "command": "helm",
      "args": ["mcp-server"]
    }
  }
}
```

---

## .mcpb Package Format

HELM ships as a `.mcpb` (MCP Bundle) for easy distribution:

```
helm-governance.mcpb
├── manifest.json       # Package metadata
├── helm                # Binary (platform-specific)
├── default-policies/   # Default policy bundle
└── README.md           # Quick start guide
```

## Config Generator Implementation

The `helm config generate` command:

1. Detects the client from `--client` flag
2. Generates the appropriate config format
3. Uses sensible defaults (local mode, default policies)
4. Supports `--preset` flag to embed a business preset
5. Outputs to stdout (pipe to config file)

```bash
# Generate with a preset
helm config generate --client claude-code --preset engineering

# Generate with jurisdiction
helm config generate --client cursor --preset engineering --jurisdiction eu-gdpr
```
