---
title: UC-008_replay_verify.sh
slug: use-cases/uc-008
---

```bash
#!/bin/bash
# UC-008: Replay verify offline
# Expected: evidence pack export and verify round-trips
set -euo pipefail

echo "=== UC-008: Replay Verify Offline ==="
cd "$(dirname "$0")/../../core"

go test -run TestExportAndVerify_RoundTrip ./cmd/helm/ -v -count=1
go test -run TestExportPack_Deterministic ./cmd/helm/ -v -count=1

echo "UC-008: PASS"

```

[View on GitHub](https://github.com/Mindburn-Labs/helm-oss/tree/main/docs/use_cases/UC-008_replay_verify.sh)
