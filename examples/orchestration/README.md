# Orchestration Shims

This directory contains thin adapter shims used by orchestration frameworks:

- `autogen/shim.py`
- `langgraph/shim.py`
- `openai_agents/shim.py`

Each shim demonstrates how to route tool execution through HELM while keeping
framework-specific orchestration code minimal.
