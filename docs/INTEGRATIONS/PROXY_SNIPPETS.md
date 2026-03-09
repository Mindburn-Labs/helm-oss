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

## JavaScript — Current OSS Support

The OSS proxy currently supports the HTTP OpenAI-compatible surface at `/v1/chat/completions`.

Responses WebSocket mode for `/v1/responses` is not shipped in the OSS runtime yet.

---

## Starting the Proxy

```bash
# HTTP mode (default)
helm proxy --upstream https://api.openai.com/v1 --port 9090

# With budget enforcement
helm proxy --upstream https://api.openai.com/v1 --daily-limit 10000 --monthly-limit 100000

# With receipt signing
helm proxy --upstream https://api.openai.com/v1 --sign my-key-id
```
