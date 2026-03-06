# HELM × Microsoft Agent Framework (Python)

Route tool execution through HELM governance in the MS Agent Framework RC.

## Install

```bash
pip install helm-sdk
```

## Quick Integration

```python
from helm_ms_agent import HelmAgentToolWrapper

wrapper = HelmAgentToolWrapper(helm_url="http://localhost:8080")

# Govern tool execution
result = wrapper.execute("deploy_service", {"env": "production"})
print(f"Verdict: {result.receipt.verdict}")  # ALLOW or DENY

# Wrap tools with decorator
@wrapper.wrap_tool
def search_documents(query: str = "") -> str:
    return f"Found docs for: {query}"

# Export evidence
wrapper.export_evidence_pack("evidence.tar")
```

## With MS Agent Framework RC

```python
from microsoft.agents import Agent, ToolDefinition
from helm_ms_agent import HelmAgentToolWrapper

wrapper = HelmAgentToolWrapper(
    helm_url="http://localhost:8080",
    fail_closed=True,
    metadata={"org": "contoso", "env": "production"},
)

# Intercept tool execution boundary
@wrapper.wrap_tool
def execute_query(sql: str = "") -> str:
    # HELM governs this before execution
    return run_sql(sql)

agent = Agent(tools=[ToolDefinition.from_function(execute_query)])
```

## Tests

```bash
cd sdk/python && pytest microsoft_agents/ -v
```
