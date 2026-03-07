# helm-google-adk

HELM governance adapter for [Google Agent Development Kit (ADK)](https://github.com/google/adk-python).

## What it does

Wraps Google ADK tools with HELM governance:

1. Every tool call is evaluated against HELM policy before execution
2. Denied calls raise `HelmToolDenyError` (fail-closed by default)
3. Receipts with SHA-256 hashes are collected for every approved execution

## Quick start

```python
from helm_google_adk import HelmADKGovernor

governor = HelmADKGovernor(helm_url="http://localhost:8080")
governed_tools = governor.govern_tools(my_tools)

agent = Agent(name="assistant", tools=governed_tools)
```

## License

Apache-2.0
