# LangGraph + HELM Integration Example

Shows how to wrap a LangGraph agentic pipeline with HELM governance receipts.

## Architecture

```
User Request → LangGraph Agent → HELM Proxy → LLM Provider
                                     │
                                 Guardian
                                     │
                           EvidencePack Export
```

## Setup

```bash
pip install langgraph langchain-openai helm-sdk
```

## Usage

```bash
python langgraph_helm.py
```

## What It Does

1. Creates a simple LangGraph agent with a tool node
2. Routes requests through a HELM-governed proxy
3. Exports the EvidencePack after execution
4. Verifies the EvidencePack offline
