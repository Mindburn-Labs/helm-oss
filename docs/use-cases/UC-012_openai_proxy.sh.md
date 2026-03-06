---
title: UC-012_openai_proxy.sh
slug: use-cases/uc-012
---

```bash
#!/bin/bash
# UC-012: OpenAI proxy loop (only if proxy enabled)
# Expected: proxy endpoint returns governed response
set -euo pipefail

echo "=== UC-012: OpenAI Proxy ==="
cd "$(dirname "$0")/../../core"

# Verify the proxy code compiles
go build ./pkg/api/

echo "UC-012: PASS (compile check; runtime test requires running server)"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm/tree/main/docs/use_cases/UC-012_openai_proxy.sh)
