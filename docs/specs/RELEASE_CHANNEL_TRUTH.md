# HELM Release Channel Truth

Status: `authoritative`

This document defines which channels are currently real, which are intentionally withheld, and what can be publicly claimed.

## Rules

- A channel is `ACTIVE` only if the package/artifact exists, the release workflow publishes it, and CI covers it.
- A channel is `WITHHELD` if the repo has partial code but no honest end-to-end publish path.
- A channel is `BLOCKED` if the package or runtime surface does not exist yet.

## Current channel matrix

| Channel | Artifact / package | Status | Public claim allowed | Notes |
| --- | --- | --- | --- | --- |
| GitHub Releases | Core binaries + release assets | `ACTIVE` | Yes | Governed by `.github/workflows/release.yml` |
| GHCR | `ghcr.io/<repo>` images | `ACTIVE` | Yes | Images are built and signed in release workflow |
| npm | `@mindburn/helm` | `ACTIVE` | Yes | Core TS SDK (v1.0.1) |
| npm | `@mindburn/helm-cli` | `ACTIVE` | Yes | CLI verifier package (v1.0.1) |
| npm | `@mindburn/helm-openai-agents` | `ACTIVE` | Yes | OpenAI Agents adapter (v1.0.2) |
| npm | `@mindburn/helm-mastra` | `ACTIVE` | Yes | Mastra adapter (v1.0.2) |
| npm | `@mindburn/helm-autogen` | `ACTIVE` | Yes | AutoGen adapter (v1.0.2) |
| npm | `@mindburn/helm-semantic-kernel` | `ACTIVE` | Yes | Semantic Kernel adapter (v1.0.2) |
| PyPI | `helm` | `WITHHELD` | No | SDK exists in repo but not yet published to PyPI |
| crates.io | `helm` | `WITHHELD` | No | SDK exists in repo but not yet published to crates.io |
| Maven Central | Java SDK | `WITHHELD` | No | SDK exists in repo but not yet published |
| Go module proxy | `core` and `sdk/go` tags | `ACTIVE` | Yes | Tag-driven; no registry upload job |
| NuGet | `.NET SDK package` | `BLOCKED` | No | No `sdk/dotnet` package exists yet |

## Claim policy

Allowed language:

- "GitHub release artifacts are available"
- "GHCR images are available"
- "npm packages are published for TypeScript SDK, CLI, and all four adapters"
- "Go module is available via git tags"

Forbidden language while this table remains unchanged:

- "PyPI / crates.io / Maven packages are published"
- ".NET/NuGet distribution is available"
- "Every channel in the repo is publicly distributed"

## Required update rule

Any PR that changes public release channels must update:

1. this file
2. `docs/PUBLISHING.md`
3. `.github/workflows/release.yml`
4. `docs/master_scope/final_standard_implementation_report.md`
