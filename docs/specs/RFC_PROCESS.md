# HELM RFC Process

> How to propose, review, and finalize changes to the HELM standard.

## 1. RFC States

```
DRAFT → REVIEW → FINAL → DEPRECATED
```

| State        | Meaning                                      |
| ------------ | -------------------------------------------- |
| `draft`      | Proposal under active development            |
| `review`     | Open for community review (30-day minimum)   |
| `final`      | Normative, locked. Changes require a new RFC |
| `deprecated` | Superseded by a newer RFC                    |

## 2. Creating an RFC

### 2.1 File Naming

```
protocols/specs/rfc/{topic}-v{N}.md
```

Examples: `receipt-format-v1.md`, `artifact-versioning-v1.md`

### 2.2 Frontmatter

```yaml
---
title: "RFC Title"
status: draft
version: "1.0.0"
created: 2026-MM-DD
authors:
  - Author Name
---
```

### 2.3 Required Sections

1. **Abstract** — one-paragraph summary
2. **Status** — current lifecycle state
3. **Introduction** — problem statement and scope
4. **Specification** — normative requirements (use RFC 2119 language)
5. **Conformance** — how to test compliance
6. **References** — related RFCs and external standards

## 3. Review Process

1. Author opens PR with `status: draft`
2. Maintainers label PR as `rfc-review`
3. 30-day comment period opens
4. Author addresses feedback, updates RFC
5. Maintainers move to `status: review` in the frontmatter
6. After review period with no blocking comments → `status: final`
7. PR merged. RFC is normative.

## 4. Versioning RFCs

- Additive changes within same major → update version (e.g., `1.0.0` → `1.1.0`)
- Breaking changes → new RFC file (e.g., `receipt-format-v2.md`)
- Old RFC updated to `status: deprecated` with pointer to successor

## 5. Standards Evolution

The HELM standard evolves through:

1. **RFCs** — formal proposals for protocol changes
2. **Conformance vectors** — machine-testable requirements
3. **Compatibility registry** — tracks who implements what
4. **Certification** — automated verification of conformance
