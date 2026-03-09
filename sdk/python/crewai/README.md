# helm-crewai

HELM governance adapter for [CrewAI](https://www.crewai.com).

## What it does

Wraps CrewAI agent tools with HELM governance:

1. Every tool call is evaluated against HELM policy before execution
2. Denied calls raise `HelmToolDenyError` (fail-closed by default)
3. Per-agent principal tracking for multi-agent crew governance
4. Receipts with SHA-256 hashes are collected for every approved execution

## Quick start

```python
from helm_crewai import HelmCrewGovernor

governor = HelmCrewGovernor(helm_url="http://localhost:8080")
governed_tools = governor.govern_tools(my_tools, agent_role="researcher")

# Use in a CrewAI agent
agent = Agent(role="researcher", tools=governed_tools)
```

## Configuration

| Parameter          | Default                 | Description          |
| ------------------ | ----------------------- | -------------------- |
| `helm_url`         | `http://localhost:8080` | HELM kernel URL      |
| `api_key`          | `None`                  | HELM API key         |
| `fail_closed`      | `True`                  | Deny on HELM errors  |
| `collect_receipts` | `True`                  | Keep receipt chain   |
| `timeout`          | `30.0`                  | HTTP timeout seconds |

## License

Apache-2.0
