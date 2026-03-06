---
title: UC-001_pep_allow_safe.sh
slug: use-cases/uc-001
---

```bash
#!/bin/bash
# UC-001: PEP allows a safe tool call with valid schema
# Expected: PASS — valid args are canonicalized and hashed
set -euo pipefail

echo "=== UC-001: PEP Allow Safe Tool Call ==="
cd "$(dirname "$0")/../../core"

go test -run TestValidateAndCanonicalizeToolArgs_NoSchema ./pkg/manifest/ -v -count=1
go test -run TestValidateAndCanonicalizeToolArgs_StableHash ./pkg/manifest/ -v -count=1
go test -run TestValidateAndCanonicalizeToolArgs_AllowExtra ./pkg/manifest/ -v -count=1

echo "UC-001: PASS"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm/tree/main/docs/use_cases/UC-001_pep_allow_safe.sh)
