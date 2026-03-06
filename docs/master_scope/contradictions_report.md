# HELM OSS Contradictions Report

> Systematic identification of contradictions, duplicates, stale content, and inconsistencies.

## 1. Verdict Vocabulary Inconsistency

### Status: PARTIALLY FIXED

The kernel (`guardian.go`, `contracts/verdict.go`) now uses canonical `ALLOW/DENY/ESCALATE`.
However, **31 files** in `core/pkg/` still contain the string `"PASS"`:

| Package            | Files | Context                                                           |
| ------------------ | ----- | ----------------------------------------------------------------- |
| `conform/`         | 5     | Uses `PASS`/`FAIL` for conformance gate results (separate domain) |
| `incubator/audit/` | 8     | Uses `PASS`/`FAIL` for audit finding severity                     |
| `executor/`        | 2     | Uses `PASS` in test assertions                                    |
| `verifier/`        | 1     | Uses `PASS` in verification outcomes                              |
| `governance/`      | 1     | Uses `PASS` in governance test                                    |
| `compliance/`      | 2     | Uses `PASS` in compliance checks                                  |
| `pack/`            | 2     | Uses `PASS` in pack verification                                  |
| `proofgraph/`      | 1     | Uses `PASS` in graph tests                                        |
| `bridge/`          | 1     | Uses `PASS` in bridge tests                                       |
| `contracts/`       | 2     | Uses `PASS` in proof/decision contexts                            |
| `crypto/`          | 2     | Uses `PASS` in hardening tests                                    |
| Other              | 4     | Various test/doc contexts                                         |

> [!IMPORTANT]
> Not all of these are kernel verdict vocabulary violations. `PASS/FAIL` is valid
> in audit, conformance, and test domains. But the codebase should clearly
> distinguish **kernel verdicts** (ALLOW/DENY/ESCALATE) from **domain outcomes**
> (PASS/FAIL). Currently this distinction is implicit.

### Recommendation

Add `// NOTE: This uses audit-domain PASS/FAIL, not kernel Verdict` comments
or create typed audit outcomes analogous to `contracts.Verdict`.

---

## 2. Duplicate Schema Locations

Two separate schema directories exist with overlapping concerns:

| Location                  | Schemas   | Status                                  |
| ------------------------- | --------- | --------------------------------------- |
| `protocols/json-schemas/` | 124 files | Primary — indexed in SCHEMA_INDEX.md    |
| `schemas/` (root)         | 4 files   | **Duplicate location** — must be merged |

Specific duplicates/overlaps:

- `schemas/orgdna.schema.json` may overlap with `protocols/json-schemas/orgdna/`

### Recommendation

Move all root `schemas/` files into `protocols/json-schemas/` and delete `schemas/`.

---

## 3. Stale Examples

| Example                                                                  | Issue                                                             |
| ------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| `examples/golden/`                                                       | Contains golden fixtures but may not match current receipt format |
| No example verifies against `receipt-format-v1.md` (now `status: final`) |
| No example uses canonical `contracts.Verdict` types                      |

### Recommendation

Regenerate golden fixtures from current code. Add conformance check that
golden fixtures validate against the finalized receipt RFC.

---

## 4. Dead Commands / Flags

| Location                  | Issue                                                                |
| ------------------------- | -------------------------------------------------------------------- |
| OpenAI proxy              | Previously returned stub responses; now requires `HELM_UPSTREAM_URL` |
| `helm server` mode        | May still reference old stub behavior in docs                        |
| `docker-compose.demo.yml` | May reference old binary paths                                       |

### Recommendation

Audit all CLI help text, docker-compose files, and docs for stale command references.

---

## 5. Outdated Documentation

| File                                   | Issue                                              |
| -------------------------------------- | -------------------------------------------------- |
| `docs/v1_adoption/legacy_inventory.md` | References WebSocket mode, old packaging           |
| `docs/ROADMAP.md`                      | Aspirational — doesn't reflect actual master scope |
| `docs/INTEGRATIONS/PROXY_SNIPPETS.md`  | May reference old proxy behavior                   |
| `docs/INTEGRATIONS/ORCHESTRATORS.md`   | May reference WebSocket endpoints                  |
| `README.md`                            | Heavy product pitch, lacks standard identity       |
| `RELEASE.md`                           | May not match actual goreleaser + CI flow          |

---

## 6. Obsolete Install Paths

| Path                                  | Issue                                                     |
| ------------------------------------- | --------------------------------------------------------- |
| `install.sh`                          | Now hardened with checksums ✅                            |
| Root `helm` binary                    | 40MB compiled binary should not exist in source           |
| `bin/` directory                      | Contains pre-built binaries — install should use releases |
| `scripts/release/homebrew_formula.rb` | May be stale if tap is separate                           |

---

## 7. Obsolete Release Scripts

| Script                               | Issue                                          |
| ------------------------------------ | ---------------------------------------------- |
| `scripts/release/distribute.sh`      | May overlap with `.goreleaser.yml`             |
| `scripts/release/maven-settings.xml` | Java SDK release — verify if active            |
| `scripts/ci/repo_audit.sh`           | 44KB shell script — verify if still functional |

---

## 8. WebSocket Endpoint References

Found in:

- `core/pkg/incubator/audit/ws_sink.go` — WebSocket audit sink
- `core/pkg/console/server.go` — Console WebSocket server
- `docs/v1_adoption/legacy_inventory.md` — Legacy WebSocket docs
- `docs/INTEGRATIONS/ORCHESTRATORS.md` — WebSocket integration docs

### Recommendation

If WebSocket mode is no longer primary, document its status explicitly.
If deprecated, remove the code and docs.

---

## 9. Old Packaging References

| Package                       | Issue                                |
| ----------------------------- | ------------------------------------ |
| `packages/mindburn-helm-cli/` | Has committed `node_modules` (91MB)  |
| `packages/helm-lab-runner/`   | Has committed `node_modules` (109MB) |
| `sdk/ts/`                     | Has committed `node_modules` (87MB)  |

These should use `.npmrc` and CI install, not committed dependencies.

---

## 10. Compiled-Only Compliance Logic

These compliance engines are Go-only and should become loadable bundles:

| File                   | Current Status | Bundle Status           |
| ---------------------- | -------------- | ----------------------- |
| `core/pkg/compliance/` | Go package     | Should be bundle-driven |
| DORA rules             | Go-only        | No bundle equivalent    |
| GDPR logic             | Go-only        | No bundle equivalent    |
| SOX logic              | Go-only        | No bundle equivalent    |
| FCA logic              | Go-only        | No bundle equivalent    |
| MiCA logic             | Go-only        | No bundle equivalent    |

### Recommendation

Phase 4 of the master scope addresses this. The `bundles/loader.go` is now
implemented. Compliance engines should be migrated to YAML policy bundles
that the loader can process.

---

## 11. Manual Type Definitions vs Codegen

These types are manually defined in multiple languages and should become codegen outputs:

| Type           | Go        | TypeScript | Python    | Java      | Rust      |
| -------------- | --------- | ---------- | --------- | --------- | --------- |
| Receipt        | ✅ manual | ✅ manual  | ✅ manual | ✅ manual | ✅ manual |
| DecisionRecord | ✅ manual | ✅ manual  | ✅ manual | ✅ manual | ✅ manual |
| Verdict enum   | ✅ manual | ✅ manual  | ✅ manual | ✅ manual | ✅ manual |
| EvidencePack   | ✅ manual | ✅ manual  | ✅ manual | ❌        | ❌        |

Now that `protocols/proto/helm/kernel/v1/helm.proto` exists, these should
be generated from the proto IDL.

---

## 12. Summary of Contradictions

| ID  | Severity     | Finding                                            | Fix Phase |
| --- | ------------ | -------------------------------------------------- | --------- |
| C1  | MEDIUM       | Verdict vocabulary split (kernel vs audit domains) | Phase 1   |
| C2  | LOW          | Duplicate `schemas/` directory                     | Phase 0   |
| C3  | LOW          | Stale golden fixtures                              | Phase 3   |
| C4  | MEDIUM       | Stub proxy behavior in docs                        | Phase 2   |
| C5  | LOW          | Stale roadmap and adoption docs                    | Phase 0   |
| C6  | HIGH         | Compiled binaries in source                        | Phase 0   |
| C7  | LOW          | Potential overlap in release scripts               | Phase 0   |
| C8  | MEDIUM       | Old WebSocket references                           | Phase 0   |
| C9  | HIGH         | Committed node_modules                             | Phase 0   |
| C10 | MEDIUM       | Go-only compliance logic                           | Phase 4   |
| C11 | HIGH         | Manual SDK types vs codegen                        | Phase 7   |
| C12 | **CRITICAL** | Private key in public repo                         | Phase 0   |
