# HELM Orchestrator Integrations

Route tool execution through the HELM execution kernel in your orchestrator framework.
Orchestrators decide sequence. HELM decides whether a proposed side effect is authorized under policy, scope, and proof requirements.

---

## OpenAI Agents SDK

### Python

```bash
pip install helm
```

```python
from helm_openai_agents import HelmToolExecutor

executor = HelmToolExecutor(helm_url="http://localhost:8080")
result = executor.execute("search_web", {"query": "HELM"})
print(result.receipt.verdict)  # ALLOW or DENY

# Optional organization-scoped metadata
result = executor.execute(
    "search_web",
    {"query": "HELM"},
    metadata={
        "organization_id": "northstar-research",
        "scope_id": "lab.discovery.search",
        "principal_id": "research_lead",
    },
)

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
import { HelmToolProxy } from "@mindburn/helm-openai-agents";

const proxy = new HelmToolProxy({ baseUrl: "http://localhost:8080" });
const governedTools = proxy.wrapTools(myTools);
```

Responses WebSocket mode is not shipped in the OSS proxy runtime yet. Use the HTTP proxy surface for current OSS deployments.

See [sdk/ts/openai-agents/](../../sdk/ts/openai-agents/) for full docs.

---

## Microsoft Agent Framework (Python)

```bash
pip install helm
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
// Source example only. No public NuGet package is currently published.
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
// Install from npm for the published OSS adapter.
// npm install @mindburn/helm-mastra
import { HelmMastraAdapter } from "@mindburn/helm-mastra";

const adapter = new HelmMastraAdapter({
  helmUrl: "http://localhost:8080",
  sandboxProvider: "opensandbox", // or 'e2b', 'daytona'
});
```

See [sdk/ts/mastra/](../../sdk/ts/mastra/) for full docs.

For a dry-run policy simulation flow, see [../POLICY_SIMULATION.md](../POLICY_SIMULATION.md).
