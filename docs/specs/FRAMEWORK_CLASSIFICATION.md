# HELM Framework Integration Classification

> Every framework integration classified as Native / Bridge / Experimental.

## Classification Tiers

| Tier             | Definition                                                                   | Conformance          | Support Level           |
| ---------------- | ---------------------------------------------------------------------------- | -------------------- | ----------------------- |
| **Native**       | First-class integration built with HELM SDK. Full governance pipeline.       | Verified (Level 4)   | Production              |
| **Bridge**       | Wrapper/adapter connecting framework to HELM. Governance pipeline via proxy. | Compatible (Level 2) | Production with caveats |
| **Experimental** | Proof-of-concept or community contribution. May have gaps.                   | Self-certified       | Best-effort             |

## Framework Matrix

| Framework                     | Language   | Tier         | Package                                       | Governance Mode            |
| ----------------------------- | ---------- | ------------ | --------------------------------------------- | -------------------------- |
| **OpenAI Agents SDK**         | Python     | Native       | `sdk/python/helm_sdk/adapters/openai_agents/` | Inline effect interception |
| **OpenAI Agents SDK**         | JS/TS      | Bridge       | `sdk/ts/src/adapters/openai-agents/`          | MCP proxy                  |
| **OpenAI Responses API**      | JS/TS      | Experimental | —                                             | WebSocket `/v1/responses`  |
| **Microsoft AutoGen**         | Python     | Bridge       | `sdk/python/helm_sdk/adapters/autogen/`       | Tool wrapper               |
| **Microsoft Semantic Kernel** | .NET       | Experimental | —                                             | MCP client                 |
| **Google ADK**                | Python     | Native       | `sdk/python/helm_sdk/adapters/adk/`           | Built-in callbacks         |
| **LangGraph**                 | Python     | Native       | `sdk/python/helm_sdk/adapters/langgraph/`     | Node middleware            |
| **LangChain**                 | Python     | Bridge       | `sdk/python/helm_sdk/adapters/langchain/`     | Tool callback handler      |
| **Mastra**                    | TypeScript | Native       | `sdk/ts/src/adapters/mastra/`                 | Middleware integration     |
| **CrewAI**                    | Python     | Experimental | —                                             | Tool decorator             |
| **Haystack**                  | Python     | Experimental | —                                             | Pipeline component         |

## Native Integration Requirements

A Native integration MUST:

1. Call `EffectBoundary.Submit()` before every tool/effect execution
2. Enforce the verdict (DENY → block, ESCALATE → pause)
3. Call `EffectBoundary.Complete()` after execution
4. Collect receipts into an EvidencePack
5. Pass conformance Level 4 vectors
6. Be tested in CI via `sdk_gates.yml`

## Bridge Integration Requirements

A Bridge integration MUST:

1. Proxy effect requests to the HELM MCP server or REST API
2. Enforce verdicts from the proxy response
3. Pass conformance Level 2 vectors

## Experimental Notes

- **OpenAI JS WebSocket mode**: The `/v1/responses` streaming API uses WebSocket frames. HELM bridges this by intercepting tool calls from the stream and evaluating them against the EffectBoundary before forwarding execution.
- **CrewAI / Haystack**: Community contributions welcome. Wrap the HELM SDK `submit()` call in a tool decorator or pipeline component.

## per-Integration Proof Report

Each integration SHOULD be able to produce a Proof Report demonstrating:

1. Effect interception is working
2. DENY verdict blocks execution
3. Receipt chain is intact
4. EvidencePack is collectible
