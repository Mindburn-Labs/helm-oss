---
title: UC-002_schema_mismatch.sh
slug: use-cases/uc-002
---

```bash
#!/bin/bash
# UC-002: Schema mismatch fail-closed
# Expected: DENY — unknown fields, type mismatch, missing required all rejected
set -euo pipefail

echo "=== UC-002: Schema Mismatch Fail-Closed ==="
cd "$(dirname "$0")/../../core"

go test -run TestValidateAndCanonicalizeToolArgs_UnknownField ./pkg/manifest/ -v -count=1
go test -run TestValidateAndCanonicalizeToolArgs_TypeMismatch ./pkg/manifest/ -v -count=1
go test -run TestValidateAndCanonicalizeToolArgs_MissingRequired ./pkg/manifest/ -v -count=1

echo "UC-002: PASS"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm/tree/main/docs/use_cases/UC-002_schema_mismatch.sh)
