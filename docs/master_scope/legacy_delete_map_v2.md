# HELM OSS Legacy Delete Map v2

> Exact list of files and directories to remove or replace.
> Ordered by severity. Execute top-to-bottom.

## 🚨 Priority 0: Security (Immediate)

```
DELETE  data/root.key           # Private key in public repo
DELETE  data/helm.db            # Database with potential secrets
DELETE  data/keys/              # Key material directory
MOVE    data/root.pub → CI secret or config  # Public key material
```

## Priority 1: Compiled Binaries (530MB+ bloat)

```
DELETE  helm                    # 40MB compiled binary at repo root
DELETE  helm-node               # 39MB compiled binary at repo root
DELETE  bin/helm                # Compiled binary
DELETE  bin/helm-node           # Compiled binary
DELETE  bin/helm-linux-amd64    # Cross-compiled binary
DELETE  bin/helm-node-linux-amd64  # Cross-compiled binary
```

## Priority 2: Committed node_modules (287MB)

```
DELETE  packages/mindburn-helm-cli/node_modules/   # 91MB
DELETE  packages/helm-lab-runner/node_modules/      # 109MB
DELETE  sdk/ts/node_modules/                        # 87MB
```

## Priority 3: Duplicate Schema Location

```
MOVE    schemas/compatibility-registry.schema.json → protocols/json-schemas/registry/
MOVE    schemas/lab_mission.schema.json            → protocols/json-schemas/lab/
MOVE    schemas/lab_receipts.schema.json           → protocols/json-schemas/lab/
MOVE    schemas/orgdna.schema.json                 → protocols/json-schemas/orgdna/
DELETE  schemas/                                    # Remove directory after moves
```

## Priority 4: Structural Cleanup

```
MOVE    apps/helm-node/ → tools/helm-node/   # apps/ dir has single child
DELETE  apps/                                 # Remove empty parent
```

## Priority 5: .gitignore Update

Add these entries to `.gitignore`:

```
# Compiled binaries
bin/
helm
helm-node

# Node modules
node_modules/

# Sensitive data
data/*.key
data/*.db
data/keys/

# Build artifacts
*.wasm
```

## Priority 6: Stale/Deprecated Content

```
DEPRECATE  docs/v1_adoption/         # Stale adoption content
REWRITE    docs/ROADMAP.md           # Must reflect master scope
REWRITE    docs/INTEGRATIONS/        # Must match actual integration status
REWRITE    README.md                 # Must reflect standard identity
REWRITE    RELEASE.md                # Must match actual release process
REWRITE    .gitignore                # Must exclude all generated content
```

## Post-Delete Verification

After executing deletions:

```bash
# Verify no private keys remain
grep -r "PRIVATE KEY" . --include="*.key" --include="*.pem"

# Verify no node_modules
find . -name node_modules -type d

# Verify no compiled binaries at root
file helm helm-node 2>/dev/null

# Verify repo size decreased
du -sh .
```

## Expected Size Reduction

| Category          | Size Removed |
| ----------------- | ------------ |
| Compiled binaries | ~240MB       |
| node_modules      | ~287MB       |
| Database/keys     | ~1MB         |
| **Total**         | **~528MB**   |
