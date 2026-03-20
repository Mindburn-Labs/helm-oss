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
| PyPI | `helm` | `ACTIVE` | Yes | Published by `publish-pypi` in release workflow |
| crates.io | `helm` | `ACTIVE` | Yes | Published by `publish-crates` in release workflow |
| Maven Central | `ai.mindburn.helm:helm` | `ACTIVE` | Yes | Published by `publish-maven` in release workflow |
| Go module proxy | `core` and `sdk/go` tags | `ACTIVE` | Yes | Tag-driven; no registry upload job |
| NuGet | `.NET SDK package` | `BLOCKED` | No | No `sdk/dotnet` package exists yet |

## Claim policy

Allowed language:

- "GitHub release artifacts are available"
- "GHCR images are available"
- "npm packages are published for the TypeScript SDK, CLI, and all four adapters"
- "PyPI, crates.io, and Maven Central packages are published"
- "Go module is available via git tags"

Forbidden language while this table remains unchanged:

- ".NET/NuGet distribution is available"
- "Every channel in the repo is publicly distributed"

## Required update rule

Any PR that changes public release channels must update:

1. this file
2. `docs/PUBLISHING.md`
3. `.github/workflows/release.yml`
4. `scripts/ci/release_truth_check.sh`
