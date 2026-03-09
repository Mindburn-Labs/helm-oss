---
title: UC-009_connector_drift.sh
slug: use-cases/uc-009
---

```bash
#!/bin/bash
# UC-009: Connector output drift fail-closed
# Expected: drift detection rejects unexpected/missing/wrong-type fields
set -euo pipefail

echo "=== UC-009: Connector Output Drift ==="
cd "$(dirname "$0")/../../core"

go test -run TestValidateToolOutput_DriftDetected ./pkg/manifest/ -v -count=1

echo "UC-009: PASS"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm-oss/tree/main/docs/use_cases/UC-009_connector_drift.sh)
