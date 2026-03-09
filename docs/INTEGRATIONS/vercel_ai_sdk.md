# Integration: Vercel AI SDK

Use HELM as the base URL for the Vercel AI SDK.

## Setup

```typescript
import { openai } from "@ai-sdk/openai";
import { generateText } from "ai";

const model = openai("gpt-4", {
  baseURL: "http://localhost:8080/v1",
});

const { text } = await generateText({
  model,
  prompt: "List files in /tmp",
});

console.log(text);
```

## Streaming

```typescript
import { openai } from "@ai-sdk/openai";
import { streamText } from "ai";

const model = openai("gpt-4", {
  baseURL: "http://localhost:8080/v1",
});

const result = streamText({
  model,
  prompt: "Explain HELM in 3 sentences",
});

for await (const chunk of result.textStream) {
  process.stdout.write(chunk);
}
```

## What Changes

- Vercel AI SDK sends all requests through HELM proxy
- Tool calls are receipted and schema-validated
- Streaming works via SSE passthrough
- No additional packages needed

→ Full example: [examples/ts_vercel_baseurl/](../../examples/ts_vercel_baseurl/)
