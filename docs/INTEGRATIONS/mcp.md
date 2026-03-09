# Integration: Model Context Protocol (MCP)

HELM provides an MCP gateway that governs tool execution via the MCP protocol.

## Capabilities

```bash
curl -s http://localhost:8080/mcp/v1/capabilities | jq '.tools[].name'
```

## Execute a Governed Tool

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

## What HELM Adds to MCP

- **Schema PEP** — input and output validation on every tool call
- **Receipts** — Ed25519-signed execution receipts
- **Budget enforcement** — ACID budget locks, fail-closed on ceiling breach
- **Approval ceremonies** — timelock + challenge/response for sensitive operations
- **ProofGraph** — append-only DAG of all tool executions

→ Full example: [examples/mcp_client/](../../examples/mcp_client/)
