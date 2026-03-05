# helm-langchain

HELM governance adapter for [LangChain](https://python.langchain.com).

## What it does

Wraps LangChain tools with HELM governance:

1. Every tool call is evaluated against HELM policy before execution
2. Denied calls raise `HelmToolDenyError` (fail-closed by default)
3. Receipts with SHA-256 hashes are collected for every approved execution

## Quick start

```python
from helm_langchain import HelmToolWrapper

wrapper = HelmToolWrapper(helm_url="http://localhost:8080")
governed_tools = wrapper.wrap_tools(my_tools)

# Use in a LangChain agent
agent = create_react_agent(llm, governed_tools)
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
