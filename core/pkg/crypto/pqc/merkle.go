// Package pqc provides Post-Quantum Cryptography enhanced Merkle trees.
// This module extends the standard executor/merkle.go with PQC capabilities.
package pqc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// PQCMerkleProfileID is the profile identifier for PQC Merkle.
const PQCMerkleProfileID = "merkle-pqc-v1"

// HashAlgorithm represents the hash algorithm used.
type HashAlgorithm string

const (
	// SHA256PQC uses SHA-256 with PQC domain separators
	SHA256PQC HashAlgorithm = "SHA256-PQC"
	// SHA256Enhanced uses enhanced SHA-256 with additional mixing
	SHA256Enhanced HashAlgorithm = "SHA256-ENHANCED"
)

// LeafDomainSeparatorPQC is the prefix for PQC leaf hashes.
var LeafDomainSeparatorPQC = []byte{0x10}

// NodeDomainSeparatorPQC is the prefix for PQC internal node hashes.
var NodeDomainSeparatorPQC = []byte{0x11}

// PQCMerkleTree represents a Post-Quantum secure Merkle tree.
type PQCMerkleTree struct {
	Root      []byte          `json:"root"`
	Leaves    []PQCMerkleLeaf `json:"leaves"`
	Levels    [][][]byte      `json:"-"`
	Algorithm HashAlgorithm   `json:"algorithm"`
	Signature *Signature      `json:"signature,omitempty"`
	SignerID  string          `json:"signer_id,omitempty"`
}

// PQCMerkleLeaf represents a leaf in the PQC Merkle tree.
type PQCMerkleLeaf struct {
	Index    int    `json:"index"`
	Path     string `json:"path"`
	Hash     []byte `json:"hash"`
	Sealed   bool   `json:"sealed"`
	SealedAt int64  `json:"sealed_at,omitempty"`
}

// PQCMerkleBuilder builds a PQC-enhanced Merkle tree.
type PQCMerkleBuilder struct {
	leaves    []PQCMerkleLeaf
	algorithm HashAlgorithm
}

// NewPQCMerkleBuilder creates a new PQC Merkle builder.
func NewPQCMerkleBuilder(algorithm HashAlgorithm) *PQCMerkleBuilder {
	if algorithm == "" {
		algorithm = SHA256PQC
	}
	return &PQCMerkleBuilder{
		leaves:    make([]PQCMerkleLeaf, 0),
		algorithm: algorithm,
	}
}

// AddLeaf adds a leaf to the tree.
func (b *PQCMerkleBuilder) AddLeaf(path string, value any, sealed bool) error {
	canonical, err := canonicalJSONPQC(value)
	if err != nil {
		return fmt.Errorf("pqc-merkle: failed to serialize leaf at %s: %w", path, err)
	}

	hash := b.computeLeafHash(canonical)

	leaf := PQCMerkleLeaf{
		Index:  len(b.leaves),
		Path:   path,
		Hash:   hash,
		Sealed: sealed,
	}

	b.leaves = append(b.leaves, leaf)
	return nil
}

// AddLeafBytes adds a leaf from raw bytes.
func (b *PQCMerkleBuilder) AddLeafBytes(path string, data []byte, sealed bool) {
	hash := b.computeLeafHash(data)

	leaf := PQCMerkleLeaf{
		Index:  len(b.leaves),
		Path:   path,
		Hash:   hash,
		Sealed: sealed,
	}

	b.leaves = append(b.leaves, leaf)
}

// computeLeafHash computes the PQC-resistant hash for a leaf.
func (b *PQCMerkleBuilder) computeLeafHash(data []byte) []byte {
	h := sha256.New()
	h.Write(LeafDomainSeparatorPQC)
	h.Write(data)
	firstHash := h.Sum(nil)

	if b.algorithm == SHA256Enhanced {
		// Double hash for enhanced security
		h2 := sha256.New()
		h2.Write(firstHash)
		return h2.Sum(nil)
	}
	return firstHash
}

// computeNodeHash computes the PQC-resistant hash for an internal node.
func (b *PQCMerkleBuilder) computeNodeHash(left, right []byte) []byte {
	h := sha256.New()
	h.Write(NodeDomainSeparatorPQC)
	h.Write(left)
	h.Write(right)
	firstHash := h.Sum(nil)

	if b.algorithm == SHA256Enhanced {
		h2 := sha256.New()
		h2.Write(firstHash)
		return h2.Sum(nil)
	}
	return firstHash
}

// Build constructs the PQC Merkle tree and returns the root.
func (b *PQCMerkleBuilder) Build() (*PQCMerkleTree, error) {
	if len(b.leaves) == 0 {
		return nil, fmt.Errorf("pqc-merkle: cannot build tree with no leaves")
	}

	level := make([][]byte, len(b.leaves))
	for i, leaf := range b.leaves {
		level[i] = leaf.Hash
	}

	levels := [][][]byte{level}

	for len(level) > 1 {
		nextLevel := make([][]byte, 0, (len(level)+1)/2)

		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				hash := b.computeNodeHash(level[i], level[i+1])
				nextLevel = append(nextLevel, hash)
			} else {
				nextLevel = append(nextLevel, level[i])
			}
		}

		level = nextLevel
		levels = append(levels, level)
	}

	return &PQCMerkleTree{
		Root:      level[0],
		Leaves:    b.leaves,
		Levels:    levels,
		Algorithm: b.algorithm,
	}, nil
}

// RootHex returns the root hash as a hex string.
func (t *PQCMerkleTree) RootHex() string {
	return hex.EncodeToString(t.Root)
}

// SignRoot signs the Merkle root using a PQC signer.
func (t *PQCMerkleTree) SignRoot(signer *PQCSigner) error {
	signature, err := signer.Sign(t.Root)
	if err != nil {
		return fmt.Errorf("pqc-merkle: failed to sign root: %w", err)
	}

	t.Signature = signature
	t.SignerID = signer.KeyID()
	return nil
}

// VerifyRootSignature verifies the Merkle root signature.
func (t *PQCMerkleTree) VerifyRootSignature(signer *PQCSigner) (bool, error) {
	if t.Signature == nil {
		return false, fmt.Errorf("pqc-merkle: no signature present")
	}

	return signer.Verify(t.Root, t.Signature)
}

// PQCMerkleProof represents an inclusion proof with PQC hashes.
type PQCMerkleProof struct {
	LeafIndex int                `json:"leaf_index"`
	LeafPath  string             `json:"leaf_path"`
	LeafHash  string             `json:"leaf_hash"`
	Siblings  []PQCMerkleSibling `json:"siblings"`
	Root      string             `json:"root"`
	Algorithm HashAlgorithm      `json:"algorithm"`
}

// PQCMerkleSibling represents a sibling in the proof path.
type PQCMerkleSibling struct {
	Hash     string `json:"hash"`
	Position string `json:"position"`
}

// GenerateProof generates an inclusion proof for a leaf.
func (t *PQCMerkleTree) GenerateProof(leafIndex int) (*PQCMerkleProof, error) {
	if leafIndex < 0 || leafIndex >= len(t.Leaves) {
		return nil, fmt.Errorf("pqc-merkle: leaf index %d out of range [0, %d)", leafIndex, len(t.Leaves))
	}

	proof := &PQCMerkleProof{
		LeafIndex: leafIndex,
		LeafPath:  t.Leaves[leafIndex].Path,
		LeafHash:  hex.EncodeToString(t.Leaves[leafIndex].Hash),
		Siblings:  make([]PQCMerkleSibling, 0),
		Root:      t.RootHex(),
		Algorithm: t.Algorithm,
	}

	idx := leafIndex
	for levelNum := 0; levelNum < len(t.Levels)-1; levelNum++ {
		level := t.Levels[levelNum]

		var siblingIdx int
		var position string

		if idx%2 == 0 {
			siblingIdx = idx + 1
			position = "right"
		} else {
			siblingIdx = idx - 1
			position = "left"
		}

		if siblingIdx < len(level) {
			proof.Siblings = append(proof.Siblings, PQCMerkleSibling{
				Hash:     hex.EncodeToString(level[siblingIdx]),
				Position: position,
			})
		}

		idx /= 2
	}

	return proof, nil
}

// VerifyPQCProof verifies a PQC Merkle inclusion proof.
func VerifyPQCProof(proof *PQCMerkleProof) (bool, error) {
	leafHash, err := hex.DecodeString(proof.LeafHash)
	if err != nil {
		return false, fmt.Errorf("pqc-merkle: invalid leaf hash: %w", err)
	}

	builder := NewPQCMerkleBuilder(proof.Algorithm)
	currentHash := leafHash

	for _, sibling := range proof.Siblings {
		siblingHash, err := hex.DecodeString(sibling.Hash)
		if err != nil {
			return false, fmt.Errorf("pqc-merkle: invalid sibling hash: %w", err)
		}

		if sibling.Position == "left" {
			currentHash = builder.computeNodeHash(siblingHash, currentHash)
		} else {
			currentHash = builder.computeNodeHash(currentHash, siblingHash)
		}
	}

	expectedRoot, err := hex.DecodeString(proof.Root)
	if err != nil {
		return false, fmt.Errorf("pqc-merkle: invalid root hash: %w", err)
	}

	return bytes.Equal(currentHash, expectedRoot), nil
}

// canonicalJSONPQC produces deterministic JSON with sorted keys.
func canonicalJSONPQC(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return marshalCanonicalPQC(raw)
}

func marshalCanonicalPQC(v any) ([]byte, error) {
	switch val := v.(type) {
	case map[string]any:
		return marshalCanonicalObjectPQC(val)
	case []any:
		return marshalCanonicalArrayPQC(val)
	default:
		return json.Marshal(v)
	}
}

func marshalCanonicalObjectPQC(m map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		keyJSON, _ := json.Marshal(k)
		buf.Write(keyJSON)
		buf.WriteByte(':')

		valJSON, err := marshalCanonicalPQC(m[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalCanonicalArrayPQC(arr []any) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')

	for i, elem := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}

		elemJSON, err := marshalCanonicalPQC(elem)
		if err != nil {
			return nil, err
		}
		buf.Write(elemJSON)
	}

	buf.WriteByte(']')
	return buf.Bytes(), nil
}
