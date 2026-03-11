# HELM × OpenAI Agents SDK (Python)

Route tool execution through HELM governance in under 5 minutes.

## Install

```bash
pip install helm  # or: pip install -e sdk/python/
```

## Quick Integration

```python
from helm_openai_agents import HelmToolExecutor, wrap_openai_tool

# 1. Create executor pointing to your HELM instance
executor = HelmToolExecutor(helm_url="http://localhost:8080")

# 2. Execute tools through governance
result = executor.execute("search_web", {"query": "HELM governance"})
print(f"Verdict: {result.receipt.verdict}")
print(f"Receipt: {result.receipt.receipt_id}")

# 3. Export EvidencePack
pack_hash = executor.export_evidence_pack("evidence.tar")
print(f"Pack SHA-256: {pack_hash}")

# 4. Verify offline
# helm verify --bundle evidence.tar
```

## With OpenAI Agents SDK

```python
from agents import Agent, Runner
from helm_openai_agents import HelmToolExecutor

executor = HelmToolExecutor(
    helm_url="http://localhost:8080",
    fail_closed=True,  # Deny on HELM unreachable
    metadata={"org": "acme", "env": "production"},
)

# Wrap your tool functions
@executor.wrap_openai_tool
def search_web(query: str) -> str:
    return f"Results for: {query}"

agent = Agent(name="researcher", tools=[search_web])
result = Runner.run_sync(agent, "Search for HELM governance")
```

## Proxy Mode (Zero Code Changes)

```bash
# Start HELM proxy
helm proxy --upstream https://api.openai.com/v1

# Point your app at the proxy
export OPENAI_BASE_URL=http://localhost:9090/v1
python your_app.py  # Every tool call is now governed
```

## Tests

```bash
cd sdk/python && pytest openai_agents/ -v
```
