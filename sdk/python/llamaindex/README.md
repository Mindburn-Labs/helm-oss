# helm-llamaindex

HELM governance adapter for [LlamaIndex](https://www.llamaindex.ai).

## What it does

Wraps LlamaIndex tools with HELM governance:

1. Every tool call is evaluated against HELM policy before execution
2. Denied calls raise `HelmToolDenyError` (fail-closed by default)
3. Works with `QueryEngineTool`, `FunctionTool`, and custom tools
4. Receipts with SHA-256 hashes are collected for every approved execution

## Quick start

```python
from helm_llamaindex import HelmToolGovernor

governor = HelmToolGovernor(helm_url="http://localhost:8080")
governed_tools = governor.govern_tools(my_tools)

agent = ReActAgent.from_tools(governed_tools, llm=llm)
```

## License

Apache-2.0
