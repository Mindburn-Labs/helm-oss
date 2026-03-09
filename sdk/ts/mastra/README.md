# @mindburn/helm-mastra

HELM governance adapter for [Mastra](https://mastra.ai) agent framework with native Daytona sandbox integration.

## What it does

Wraps Mastra's Daytona sandbox with HELM governance:

1. Every sandbox `exec` is evaluated against HELM policy first
2. Denied commands never reach the sandbox
3. Receipts form a deterministic proof chain

## Quick start

```typescript
import { HelmMastraSandbox } from "@mindburn/helm-mastra";

const sandbox = new HelmMastraSandbox({
  baseUrl: "http://localhost:8080",
  daytonaApiKey: process.env.DAYTONA_API_KEY,
});

const result = await sandbox.exec({
  command: ["python3", "-c", 'print("governed exec")'],
});

console.log(result.stdout); // "governed exec\n"
console.log(result.receipt.requestHash); // "sha256:..."
```

## Configuration

| Option            | Default                  | Description              |
| ----------------- | ------------------------ | ------------------------ |
| `baseUrl`         | required                 | HELM kernel URL          |
| `daytonaApiKey`   | required                 | Daytona API key          |
| `daytonaUrl`      | `https://api.daytona.io` | Daytona API URL          |
| `failClosed`      | `true`                   | Deny on HELM errors      |
| `defaultLanguage` | `python3`                | Default exec language    |
| `execTimeout`     | `30000`                  | Per-command timeout (ms) |

## License

Apache-2.0
