# @mindburn/helm-autogen

HELM governance adapter for [AutoGen](https://microsoft.github.io/autogen).

## What it does

Wraps AutoGen agent tool functions with HELM governance:

1. Every tool call is evaluated against HELM policy before execution
2. Denied calls throw `HelmToolDenyError` (fail-closed by default)
3. Receipts with SHA-256 hashes are collected for every approved execution
4. Cross-platform: works in Node.js, Deno, and Bun

## Quick start

```typescript
import { HelmAutoGenGovernor } from "@mindburn/helm-autogen";

const governor = new HelmAutoGenGovernor({ helmUrl: "http://localhost:8080" });
const governed = governor.governTools([
  { name: "search", fn: searchWeb },
  { name: "calculate", fn: doMath },
]);
```

## License

Apache-2.0
