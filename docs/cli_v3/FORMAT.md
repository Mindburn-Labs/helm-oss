# HELM v3 Evidence Bundle Format

## Canonicalization

`00_INDEX.json` MUST be canonical JSON:

- Keys sorted lexicographically (recursive)
- No trailing whitespace
- UTF-8 encoded
- No BOM
- Single trailing newline

This matches [JCS (RFC 8785)](https://www.rfc-editor.org/rfc/rfc8785).

## Manifest Root Hash

```
manifest_root_hash = sha256(canonical_bytes(00_INDEX.json))
```

Identity of the bundle. Cache key. Single hash that pins the entire evidence tree.

## Merkle Tree

Leaves are the `sha256` hex strings from each `00_INDEX.json` entry, **sorted ascending by `path` string**. Each hex string is decoded to 32 bytes before hashing.

### Construction

```
leaf_hash    = sha256(0x00 || entry_sha256_bytes)     # domain separator: leaf
internal     = sha256(0x01 || left_hash || right_hash) # domain separator: node
odd_leaf     → duplicate last leaf
merkle_root  = root hash (hex)
```

### Ordering

Leaves are sorted by canonical `path` string (ascending, lexicographic). This prevents order drift across implementations.

### Verification Algorithm

```
1. Read 00_INDEX.json
2. Sort entries by path ascending
3. Decode each entry.sha256 from hex → 32 bytes
4. Hash each: leaf = sha256(0x00 || bytes)
5. Build tree bottom-up:
   a. If odd number of leaves, duplicate last
   b. Parent = sha256(0x01 || left || right)
6. Root = final hash (hex)
7. Compare to attestation merkle_root
```

## Attestation

```json
{
  "format": "helm-attestation-v3",
  "release_tag": "v0.9.1",
  "asset_name": "helm-evidence-v0.9.1.tar",
  "asset_sha256": "abc123...",
  "manifest_root_hash": "def456...",
  "merkle_root": "789abc...",
  "created_at": "2026-02-21T12:00:00Z",
  "profiles_version": "1.0.0"
}
```

Signed with Ed25519. Signature is over `sha256(canonical_bytes(attestation_json))`.

## Public Key

Shipped in CLI as pinned constant. Key rotation via versioned key list.
