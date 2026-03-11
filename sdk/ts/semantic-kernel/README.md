# @mindburn/helm-semantic-kernel

HELM governance adapter for [Microsoft Semantic Kernel](https://github.com/microsoft/semantic-kernel).

## What it does

Wraps Semantic Kernel plugin functions with HELM governance:

1. Every function call is evaluated against HELM policy before execution
2. Denied calls throw `HelmToolDenyError` (fail-closed by default)
3. Per-plugin principal tracking (e.g. `MyPlugin.MyFunction`)
4. Receipts with SHA-256 hashes are collected for every approved execution

## Quick start

```typescript
import { HelmSKGovernor } from "@mindburn/helm-semantic-kernel";

const governor = new HelmSKGovernor({ helmUrl: "http://localhost:8080" });
const governed = governor.governPlugin("MathPlugin", {
  add: (a: number, b: number) => a + b,
  multiply: (a: number, b: number) => a * b,
});
```

## License

Apache-2.0
