# HELM Orchestrator Integrations

Route tool execution through HELM governance in your orchestrator framework.

---

## OpenAI Agents SDK

### Python

```bash
pip install helm-sdk
```

```python
from helm_openai_agents import HelmToolExecutor

executor = HelmToolExecutor(helm_url="http://localhost:8080")
result = executor.execute("search_web", {"query": "HELM"})
print(result.receipt.verdict)  # ALLOW or DENY

# Export evidence
executor.export_evidence_pack("evidence.tar")
```

**Proxy mode (zero code changes):**

```bash
helm proxy --upstream https://api.openai.com/v1
export OPENAI_BASE_URL=http://localhost:9090/v1
python your_app.py
```

### TypeScript / JavaScript

```bash
npm install @mindburn/helm-openai-agents
```

```typescript
import { HelmGovernanceAdapter } from "@mindburn/helm-openai-agents";

const adapter = new HelmGovernanceAdapter({ helmUrl: "http://localhost:8080" });
const result = await adapter.execute("search_web", { query: "HELM" });
```

**Responses WebSocket mode:**

```bash
helm proxy --websocket --upstream https://api.openai.com/v1
export OPENAI_WEBSOCKET_BASE_URL=ws://localhost:9090
# HELM serves /v1/responses over WebSocket
```

See [sdk/ts/openai-agents/](../../sdk/ts/openai-agents/) for full docs.

---

## Microsoft Agent Framework (Python)

```bash
pip install helm-sdk
```

```python
from helm_ms_agent import HelmAgentToolWrapper

wrapper = HelmAgentToolWrapper(helm_url="http://localhost:8080")

@wrapper.wrap_tool
def execute_query(sql: str = "") -> str:
    return run_sql(sql)
```

See [sdk/python/microsoft_agents/](../../sdk/python/microsoft_agents/) for full docs.

### .NET (Minimal Example)

```csharp
// Add NuGet: Mindburn.Helm.Governance
using Mindburn.Helm;

var helm = new HelmGovernance("http://localhost:8080");
var result = await helm.EvaluateAsync("deploy", new { env = "prod" });
if (result.Verdict == "DENY") throw new Exception(result.ReasonCode);
```

See [examples/ms_agent_framework/dotnet/](../../examples/ms_agent_framework/dotnet/).

---

## LangChain

```python
from helm_langchain import HelmToolExecutor

executor = HelmToolExecutor(helm_url="http://localhost:8080")

# Wrap any LangChain tool
governed_tool = executor.wrap(search_tool)
```

See [sdk/python/langchain/](../../sdk/python/langchain/) for full docs.

---

## Mastra

```typescript
import { HelmMastraAdapter } from "@mindburn/helm-mastra";

const adapter = new HelmMastraAdapter({
  helmUrl: "http://localhost:8080",
  sandboxProvider: "opensandbox", // or 'e2b', 'daytona'
});
```

See [sdk/ts/mastra/](../../sdk/ts/mastra/) for full docs.
