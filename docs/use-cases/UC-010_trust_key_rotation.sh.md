---
title: UC-010_trust_key_rotation.sh
slug: use-cases/uc-010
---

```bash
#!/bin/bash
# UC-010: Trust key rotation replay correctness
# Expected: event-sourced registry replays correctly at any Lamport height
set -euo pipefail

echo "=== UC-010: Trust Key Rotation ==="
cd "$(dirname "$0")/../../core"

go test -run TestTrustRegistry ./pkg/trust/registry/ -v -count=1

echo "UC-010: PASS"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm-oss/tree/main/docs/use_cases/UC-010_trust_key_rotation.sh)
