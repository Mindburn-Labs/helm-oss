# HELM Proxy — Copy-Paste Snippets

Drop-in integration with your existing OpenAI-compatible app.

---

## Python (HTTP)

```python
from openai import OpenAI

# Point at HELM proxy instead of OpenAI directly
client = OpenAI(base_url="http://localhost:9090/v1")

# Everything works the same — tool calls are now governed
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Search for HELM governance"}],
    tools=[{"type": "function", "function": {"name": "search", "parameters": {"type": "object", "properties": {"query": {"type": "string"}}}}}],
)

# Check governance headers
print(response.headers.get("X-Helm-Receipt-ID"))
print(response.headers.get("X-Helm-Status"))
```

---

## JavaScript / TypeScript (HTTP)

```typescript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:9090/v1",
});

const response = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role: "user", content: "Search for HELM governance" }],
});
```

---

## JavaScript — Responses WebSocket Mode

For the OpenAI Agents SDK JS websocket transport:

```bash
# Start HELM proxy with Responses WebSocket mode
helm proxy --websocket --upstream https://api.openai.com/v1
```

```typescript
import { Agent, run } from "@openai/agents";

// HELM serves /v1/responses over WebSocket
process.env.OPENAI_WEBSOCKET_BASE_URL = "ws://localhost:9090";

const agent = new Agent({ name: "researcher", model: "gpt-4o" });
const result = await run(agent, "Search for HELM governance");
```

HELM intercepts Responses API events (`response.create`) over WebSocket, preserves `previous_response_id` chaining, and generates receipts per event.

---

## Starting the Proxy

```bash
# HTTP mode (default)
helm proxy --upstream https://api.openai.com/v1 --port 9090

# With Responses WebSocket
helm proxy --websocket --upstream https://api.openai.com/v1

# With budget enforcement
helm proxy --upstream https://api.openai.com/v1 --daily-limit 10000 --monthly-limit 100000

# With receipt signing
helm proxy --upstream https://api.openai.com/v1 --sign my-key-id
```
