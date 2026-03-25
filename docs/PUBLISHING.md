---
title: PUBLISHING
---

# HELM Publishing

`docs/specs/RELEASE_CHANNEL_TRUTH.md` is the authoritative shipping matrix for HELM OSS.

This document explains how public install claims map to release automation.

## Rules

- A channel may be publicly documented only if the release workflow publishes it.
- A published channel must appear in the release truth matrix.
- Package identity is not the same as public availability. In-repo package metadata alone is not enough.
- When a channel changes, update the truth matrix, this document, and the release workflow in the same PR.

## Current public channels

- GitHub Releases: core binaries and release assets
- GHCR: container images
- npm: `@mindburn/helm`, `@mindburn/helm-cli`, and the TypeScript adapters
- PyPI: `helm`
- crates.io: `helm`
- Maven Central: `ai.mindburn.helm:helm`
- Go module proxy: git tag based distribution

## Install claims allowed today

```bash
npm install @mindburn/helm
npx @mindburn/helm-cli
pip install helm
cargo add helm
```

```xml
<dependency>
  <groupId>ai.mindburn.helm</groupId>
  <artifactId>helm</artifactId>
</dependency>
```

## Not yet public

- NuGet / `.NET` distribution
- Homebrew formula

## CI enforcement

CI runs `scripts/ci/release_truth_check.sh` to validate:

- release workflow publish jobs match the truth matrix
- adapter publish steps are present when adapters are claimed as public
- install docs and integration docs do not drift from the published matrix
