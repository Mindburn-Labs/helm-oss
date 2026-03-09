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
| npm | `@mindburn/helm-sdk` | `ACTIVE` | Yes | Core TS SDK only |
| npm | `mindburn-helm-cli` | `ACTIVE` | Yes | CLI package only |
| npm | `@mindburn/helm-openai-agents` | `WITHHELD` | No | In repo, but not a clean standalone publish target yet |
| npm | `@mindburn/helm-mastra` | `WITHHELD` | No | In repo, but not a clean standalone publish target yet |
| PyPI | `helm-sdk` | `ACTIVE` | Yes | Published through release workflow |
| crates.io | `helm-sdk` | `ACTIVE` | Yes | Rust SDK publish path exists |
| Maven Central | Java SDK | `ACTIVE` | Yes | Release workflow contains publish job |
| Go module proxy | `core` and `sdk/go` tags | `ACTIVE` | Yes | Tag-driven; no registry upload job |
| NuGet | `.NET SDK package` | `BLOCKED` | No | No `sdk/dotnet` package exists yet |

## Claim policy

Allowed language:

- “GitHub release artifacts are available”
- “GHCR images are available”
- “Core SDKs for TypeScript, Python, Rust, Java, and Go are in scope”
- “Adapter packages may exist in-repo but are not yet all published as standalone channels”

Forbidden language while this table remains unchanged:

- “All adapters are published”
- “.NET/NuGet distribution is available”
- “Every package in the repo is publicly distributed”
- “Broad package/channel distribution is complete”

## Required update rule

Any PR that changes public release channels must update:

1. this file
2. `docs/PUBLISHING.md`
3. `.github/workflows/release.yml`
4. `docs/master_scope/final_standard_implementation_report.md`
