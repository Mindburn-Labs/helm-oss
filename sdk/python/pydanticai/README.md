# helm-pydanticai

HELM governance adapter for [PydanticAI](https://ai.pydantic.dev).

## What it does

Provides a `@helm_tool` decorator that wraps PydanticAI tools with HELM governance:

1. Every tool call is evaluated against HELM policy before execution
2. Denied calls raise `HelmToolDenyError` (fail-closed by default)
3. Works with PydanticAI's function-based tool definition pattern

## Quick start

```python
from helm_pydanticai import helm_tool

@helm_tool(helm_url="http://localhost:8080")
def search_web(ctx: RunContext, query: str) -> str:
    return do_search(query)
```

## License

Apache-2.0
