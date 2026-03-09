# Integration: LangChain

Use HELM as a drop-in proxy for LangChain's OpenAI integration.

## Python

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4",
    openai_api_base="http://localhost:8080/v1",
)

response = llm.invoke("List files in /tmp")
print(response.content)
```

## With Tools

```python
from langchain_openai import ChatOpenAI
from langchain_core.tools import tool

@tool
def list_files(path: str) -> str:
    """List files at path."""
    import os
    return "\n".join(os.listdir(path))

llm = ChatOpenAI(
    model="gpt-4",
    openai_api_base="http://localhost:8080/v1",
).bind_tools([list_files])

response = llm.invoke("List files in /tmp")
# Tool calls are intercepted by HELM, schema-validated, and receipted
```

## What Changes

- LangChain sends requests to HELM proxy
- HELM applies schema PEP, budget enforcement, and approval ceremonies
- Tool call results get receipted with Ed25519 signatures
- All other LangChain features (chains, agents, memory) work unchanged

## Notes

- HELM is transparent to LangChain — no special adapter needed
- Streaming is supported via SSE passthrough
- Set `OPENAI_API_KEY` as usual; HELM forwards it to upstream
