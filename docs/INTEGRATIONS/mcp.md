# Integration: Model Context Protocol (MCP)

HELM provides an MCP gateway that governs tool execution via the MCP protocol.
The OSS runtime now exposes two surfaces:

- `/mcp` for modern streamable HTTP / JSON-RPC clients with protocol negotiation
- `/mcp/v1/*` as a legacy HTTP compatibility layer for scripts and smoke tests

## Modern MCP Endpoint

Initialize a remote MCP session:

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}' | jq .
```

List tools with negotiated protocol headers:

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'MCP-Protocol-Version: 2025-11-25' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | jq '.result.tools[] | {name, title, outputSchema, annotations}'
```

Call a governed tool:

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'MCP-Protocol-Version: 2025-11-25' \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"file_read","arguments":{"path":"/tmp/test.txt"}}}' | jq .
```

**Expected:** `result.content[]`, `result.structuredContent`, and `result.receipt_id` when the call is allowed.

## Legacy Capabilities

```bash
curl -s http://localhost:8080/mcp/v1/capabilities | jq '.tools[].name'
```

## Legacy Execute a Governed Tool

```bash
curl -s -X POST http://localhost:8080/mcp/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"method":"file_read","params":{"path":"/tmp/test.txt"}}' | jq .
```

**Expected:**
```json
{
  "result": "...",
  "receipt_id": "rec_...",
  "reason_code": "ALLOW"
}
```

## Denied Tool Call

```bash
curl -s -X POST http://localhost:8080/mcp/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"method":"unknown_tool","params":{}}' | jq .
```

**Expected:**
```json
{
  "error": {
    "message": "Tool not found: unknown_tool",
    "reason_code": "DENY_TOOL_NOT_FOUND"
  }
}
```

## OAuth Discovery for Remote HTTP

```bash
HELM_OAUTH_BEARER_TOKEN=testtoken helm mcp serve --transport http --port 9100 --auth oauth
curl -s http://localhost:9100/.well-known/oauth-protected-resource/mcp | jq .
```

The OSS runtime uses a local bearer-token gate for remote HTTP and publishes protected-resource metadata for clients that understand MCP OAuth discovery.

## What HELM Adds to MCP

- **Schema PEP** — input and output validation on every tool call
- **Protocol negotiation** — supports `2025-11-25`, `2025-06-18`, and `2025-03-26`
- **Structured tool results** — `structuredContent` + text content for backwards compatibility
- **Receipts** — Ed25519-signed execution receipts
- **Budget enforcement** — ACID budget locks, fail-closed on ceiling breach
- **Approval ceremonies** — timelock + challenge/response for sensitive operations
- **ProofGraph** — append-only DAG of all tool executions

→ Full example: [examples/mcp_client/](../../examples/mcp_client/)
