# OpenAI Agents SDK + HELM Integration Example

Shows how to wrap OpenAI's Agents SDK with HELM governance for receipt-bound agent execution.

## Architecture

```
User → OpenAI Agent → HELM Proxy → OpenAI API
                          │
                      Guardian
                          │
                 EvidencePack Export
```

## Setup

```bash
pip install openai helm-sdk
```

## Usage

```bash
python openai_agents_helm.py
```

## What It Does

1. Creates an OpenAI agent with tool definitions
2. Routes all API calls through the HELM proxy
3. Every tool call and response gets a governance receipt
4. Exports a verifiable EvidencePack
