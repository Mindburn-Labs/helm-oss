# Integration: OpenAI base_url Swap

The simplest way to integrate HELM. Change one line, get receipts for every tool call.

## Python

```diff
- client = openai.OpenAI()
+ client = openai.OpenAI(base_url="http://localhost:8080/v1")
```

Full example:

```python
import openai

client = openai.OpenAI(base_url="http://localhost:8080/v1")

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "List files in /tmp"}],
)
print(response.choices[0].message.content)
# Response headers include:
#   X-Helm-Receipt-ID: rec_a1b2c3...
#   X-Helm-Output-Hash: sha256:7f83b1...
#   X-Helm-Lamport-Clock: 42
```

## TypeScript / Node.js

```diff
- const openai = new OpenAI();
+ const openai = new OpenAI({ baseURL: "http://localhost:8080/v1" });
```

## Go

```go
config := openai.DefaultConfig("your-api-key")
config.BaseURL = "http://localhost:8080/v1"
client := openai.NewClientWithConfig(config)
```

## What Happens

1. Your app sends requests to HELM proxy instead of OpenAI directly
2. HELM forwards to the configured upstream (OpenAI, Anthropic, or any compatible API)
3. Tool calls are intercepted, schema-validated, and receipted
4. Response includes `X-Helm-Receipt-ID`, `X-Helm-Output-Hash`, `X-Helm-Lamport-Clock` headers
5. Receipts are stored in the ProofGraph DAG

## Configuring Upstream

Set via environment variable or config:

```bash
# Default: OpenAI
HELM_UPSTREAM_URL=https://api.openai.com/v1

# Anthropic (via OpenAI-compat proxy)
HELM_UPSTREAM_URL=https://your-anthropic-proxy/v1

# Local model (Ollama, vLLM, etc.)
HELM_UPSTREAM_URL=http://localhost:11434/v1
```

→ Full example: [examples/python_openai_baseurl/](../../examples/python_openai_baseurl/)
