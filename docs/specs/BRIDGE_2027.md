# HELM 2027 Bridge Layer

> Forward compatibility for next-generation agent platforms.

## 1. Purpose

The 2027 bridge layer provides forward-looking integration points for
emerging agent platforms that are expected to reach production maturity
in 2027. These are classified as **Experimental** integrations with
bridge-level governance.

## 2. Target Platforms

| Platform              | Vendor          | Integration Type | Status       |
| --------------------- | --------------- | ---------------- | ------------ |
| **Antigravity**       | Google DeepMind | MCP server       | Experimental |
| **Kiro**              | Amazon / AWS    | MCP bridge       | Experimental |
| **Q Developer Agent** | AWS             | MCP bridge       | Experimental |
| **Devin v2**          | Cognition       | REST bridge      | Planned      |
| **Replit Agent v2**   | Replit          | MCP server       | Planned      |
| **Windsurf Next**     | Codeium         | MCP server       | Experimental |

## 3. Bridge Architecture

```
┌─────────────────┐
│  2027 Platform  │
│  (Antigravity,  │
│   Kiro, etc.)   │
└────────┬────────┘
         │ MCP or REST
    ┌────▼────┐
    │  HELM   │
    │  Bridge │  ← Translates platform-native calls
    │  Layer  │     to HELM EffectBoundary
    └────┬────┘
         │
    ┌────▼────┐
    │  HELM   │
    │  Kernel │
    └─────────┘
```

### Bridge Responsibilities

1. **Protocol Translation**: Convert platform-native effect format to HELM `EffectRequest`
2. **Auth Mapping**: Map platform auth to HELM auth modes
3. **Receipt Forwarding**: Return receipts back to platform in native format
4. **Metadata Enrichment**: Add `platform_id` and `bridge_version` to receipts

## 4. Integration Patterns

### 4.1 MCP-Native (Antigravity, Windsurf Next)

Platforms that natively support MCP use HELM as a standard MCP server:

```json
{
  "mcpServers": {
    "helm": {
      "command": "helm",
      "args": ["mcp-server", "--mode=governance", "--bridge=antigravity"]
    }
  }
}
```

### 4.2 REST Bridge (Kiro, Q Developer, Devin)

Platforms without MCP support use a REST bridge:

```yaml
bridge:
  platform: kiro
  upstream: http://localhost:4001/v1/effects
  auth_forward: true
  receipt_format: json # or protobuf
```

### 4.3 SDK Embedding

For platforms with extension SDKs:

```python
from helm_sdk import HelmClient
from helm_sdk.bridges import register_bridge

# Register platform-specific bridge
register_bridge("antigravity", {
    "effect_mapping": "auto",
    "receipt_format": "compact",
})
```

## 5. Forward Compatibility Guarantees

1. Bridge API surface is **experimental** — may change between minor versions
2. Bridges MUST NOT override kernel governance decisions
3. Bridge-generated receipts carry `bridge_metadata` field for traceability
4. Bridges inherit the jurisdiction/industry/business preset of the parent config

## 6. Deprecation Timeline

| Phase        | Timeline | Action                                                      |
| ------------ | -------- | ----------------------------------------------------------- |
| Experimental | 2026 H2  | Bridge layer available, APIs unstable                       |
| Beta         | 2027 H1  | APIs stabilize, conformance vectors added                   |
| GA           | 2027 H2  | Bridge promoted to Native if platform supports MCP natively |
