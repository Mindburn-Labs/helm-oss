---
title: UC-007_proofgraph_export.sh
slug: use-cases/uc-007
---

```bash
#!/bin/bash
# UC-007: ProofGraph export
# Expected: graph nodes are created, chained, and validated
set -euo pipefail

echo "=== UC-007: ProofGraph Export ==="
cd "$(dirname "$0")/../../core"

go test -run TestGraph_AppendAndValidate ./pkg/proofgraph/ -v -count=1
go test -run TestGraph_LamportMonotonicity ./pkg/proofgraph/ -v -count=1

echo "UC-007: PASS"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm-oss/tree/main/docs/use_cases/UC-007_proofgraph_export.sh)
