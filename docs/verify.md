# Verify HELM Conformance

Verify any HELM release in one command. No configuration needed.

## Quick Start

```bash
npx @mindburn/helm
```

This launches the **interactive flow**:
1. Choose artifact source (latest release or local bundle)
2. Select conformance level (L1 or L2)
3. View verification results
4. Optionally drill down into gate details or generate an HTML report

## CI / Automation

```bash
# Verify local bundle — JSON on stdout, human summary on stderr
npx @mindburn/helm --ci --bundle ./evidence 2>/dev/null

# Exit code: 0 = pass, 1 = fail, 2 = error, 3 = no bundle
echo $?

# Pipe to jq for specific fields
npx @mindburn/helm --ci --bundle ./evidence 2>/dev/null | jq .verdict
# "PASS"
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `--bundle PATH` | Local evidence bundle directory | — |
| `--level L1\|L2` | Conformance level | L2 |
| `--ci` | CI mode (JSON stdout, exit code) | — |
| `--json` | Alias for `--ci` | — |
| `--depth 0-3` | Output verbosity | 1 |
| `--report PATH` | Generate HTML evidence report | — |
| `--no-cache` | Skip download cache | — |
| `--cache-dir DIR` | Custom cache directory | `~/.helm/cache` |

## Output Depth

| Depth | Content |
|-------|---------|
| 0 | Badge + short hash |
| 1 | Summary table (structure, hash chain, signature, gates, roots) |
| 2 | Per-gate details with failure reasons |
| 3 | Full tree stats with leaf counts |

## What Gets Verified

1. **Structure** — §3.1 mandatory directories and files
2. **Hash chain** — every INDEX entry hash matches file contents
3. **Manifest root hash** — `sha256(00_INDEX.json)` for bundle identity
4. **Merkle root** — real Merkle tree over entry hashes (domain-separated)
5. **Signature** — conformance report signature (when present)
6. **Gates** — L1/L2 gate pass/fail against 01_SCORE.json
7. **Attestation** — Ed25519 signature over release attestation (when downloading)

## Examples

```bash
# Verify a specific bundle with detailed gate output
npx @mindburn/helm --bundle ./artifacts/conformance --level L2 --depth 2

# Generate an HTML evidence report
npx @mindburn/helm --bundle ./evidence --report ./report.html

# Verify at minimal level
npx @mindburn/helm --bundle ./evidence --level L1 --depth 0
```

## HTML Report

The `--report` flag generates a single-file HTML evidence report suitable for embedding in audit documentation:

```bash
npx @mindburn/helm --bundle ./evidence --report ./helm-report.html
open ./helm-report.html
```

## Programmatic API

```typescript
import { verifyBundle, computeMerkleRoot, LEVELS } from "@mindburn/helm";

const result = await verifyBundle("./evidence", "L2");
console.log(result.verdict);          // "PASS" or "FAIL"
console.log(result.roots.merkle_root); // real Merkle root
```
