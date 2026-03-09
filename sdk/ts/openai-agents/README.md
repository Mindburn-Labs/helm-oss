# @mindburn/helm-openai-agents

HELM governance adapter for the [OpenAI Agents SDK](https://github.com/openai/openai-agents-sdk).

## What it does

Wraps tool definitions with HELM governance so every tool call:

1. Is evaluated against HELM policy **before** execution
2. Is denied if the policy rejects it (fail-closed by default)
3. Produces a receipt for the execution proof chain

## Quick start

```typescript
import { HelmToolProxy } from "@mindburn/helm-openai-agents";

const proxy = new HelmToolProxy({
  baseUrl: "http://localhost:8080",
  apiKey: process.env.HELM_API_KEY,
});

// Wrap your tools
const governedTools = proxy.wrapTools(myTools);

// Use in an agent — every call now goes through HELM
const agent = new Agent({ tools: governedTools });
```

## Configuration

| Option            | Default  | Description          |
| ----------------- | -------- | -------------------- |
| `baseUrl`         | required | HELM kernel URL      |
| `apiKey`          | optional | HELM API key         |
| `failClosed`      | `true`   | Deny on HELM errors  |
| `collectReceipts` | `true`   | Keep receipt chain   |
| `onReceipt`       | —        | Callback per receipt |
| `onDeny`          | —        | Callback per denial  |

## License

Apache-2.0
