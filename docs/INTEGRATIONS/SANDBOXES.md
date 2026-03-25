---
title: SANDBOXES
---

# HELM Sandbox Governance — OpenSandbox / E2B / Daytona

HELM enforces strict governance on sandbox execution with preflight posture checks and receipt preimage binding.

---

## Quick Start

```bash
# Mock provider (zero deps, for demos)
helm sandbox exec --provider mock -- echo "Hello"

# Real providers
helm sandbox exec --provider opensandbox -- npm test
helm sandbox exec --provider e2b -- python script.py
helm sandbox exec --provider daytona -- cargo build
```

---

## Strict Preflight

Before any sandbox command executes, HELM performs a strict posture check:

| Check            | Description                        |
| ---------------- | ---------------------------------- |
| Provider version | Pinned to known-good version       |
| Image digest     | SHA-256 of container/sandbox image |
| Egress policy    | Hash of network egress rules       |
| Mount policy     | Read-only workspace mounts         |
| Resource limits  | CPU, memory, time bounds           |

**Degraded posture → automatic DENY.** No fallthrough.

---

## Receipt Preimage Binding

Every sandbox execution produces a receipt whose preimage binds:

```
provider_version | sandbox_spec_hash | image_digest | mounts_hash | egress_policy_hash | resource_limits_hash
```

This means the receipt cryptographically proves _which exact sandbox configuration_ was used.

---

## Conformance Tiers

```bash
# Check provider conformance
helm sandbox conform --provider opensandbox --tier compatible
helm sandbox conform --provider e2b --tier verified
helm sandbox conform --provider daytona --tier sovereign --json
```

| Tier           | Requirements                                        |
| -------------- | --------------------------------------------------- |
| **Compatible** | L1 pass (preflight, receipt binding)                |
| **Verified**   | L1 + L2 + strict preflight + receipt binding checks |
| **Sovereign**  | Verified + L3 degraded-path adversarial tests       |

---

## Provider Setup

### OpenSandbox

```bash
export OPENSANDBOX_API_KEY=your-key
helm sandbox exec --provider opensandbox -- echo "governed"
```

### E2B

```bash
export E2B_API_KEY=your-key
helm sandbox exec --provider e2b -- echo "governed"
```

### Daytona

```bash
export DAYTONA_API_KEY=your-key
helm sandbox exec --provider daytona -- echo "governed"
```

---

## Switching Providers

One flag:

```bash
helm sandbox exec --provider opensandbox -- npm test
helm sandbox exec --provider e2b -- npm test        # same command, different provider
helm sandbox exec --provider daytona -- npm test
```

Receipts bind to the specific provider configuration used.
