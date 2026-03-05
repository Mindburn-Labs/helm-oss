package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ChainSource provides the current head hash of an audit chain.
// Each independent audit log implementation should satisfy this.
type ChainSource interface {
	GetChainHead() string
}

// AnchorPoint is the result of anchoring multiple chain heads
// into a single Merkle root.
type AnchorPoint struct {
	// RootHash is the SHA-256 Merkle root of all chain heads.
	RootHash string `json:"root_hash"`
	// Chains maps chain name → head hash at anchor time.
	Chains map[string]string `json:"chains"`
	// Timestamp when the anchor was computed.
	Timestamp time.Time `json:"timestamp"`
	// ChainCount is the number of chains included.
	ChainCount int `json:"chain_count"`
}

// MerkleAnchor computes a single root hash from multiple independent
// audit chain heads.
//
// This prevents an attacker who compromises one chain from going
// undetected by the others. If any chain head changes independently,
// the root hash changes, alerting the system.
//
// Usage:
//
//	anchor := audit.NewMerkleAnchor()
//	anchor.RegisterChain("store", auditStore)
//	anchor.RegisterChain("guardian", guardianLog)
//	anchor.RegisterChain("crypto", cryptoLog)
//	point := anchor.Anchor()
//	// point.RootHash is the combined trust root
type MerkleAnchor struct {
	chains map[string]ChainSource
}

// NewMerkleAnchor creates an empty anchor with no chains registered.
func NewMerkleAnchor() *MerkleAnchor {
	return &MerkleAnchor{chains: make(map[string]ChainSource)}
}

// RegisterChain adds a named chain source to the anchor.
func (a *MerkleAnchor) RegisterChain(name string, source ChainSource) {
	if source == nil || name == "" {
		return
	}
	a.chains[name] = source
}

// ChainCount returns the number of registered chains.
func (a *MerkleAnchor) ChainCount() int {
	return len(a.chains)
}

// Anchor computes the Merkle root from all registered chain heads.
//
// The computation is deterministic:
//  1. Collect all chain heads
//  2. Sort by chain name (deterministic ordering)
//  3. Compute leaf hashes: SHA-256(name + ":" + head)
//  4. Build Merkle tree from leaves
//  5. Return root hash
func (a *MerkleAnchor) Anchor() AnchorPoint {
	if len(a.chains) == 0 {
		return AnchorPoint{
			RootHash:   "empty",
			Chains:     map[string]string{},
			Timestamp:  time.Now(),
			ChainCount: 0,
		}
	}

	// 1. Collect and sort chain names for deterministic ordering
	names := make([]string, 0, len(a.chains))
	for name := range a.chains {
		names = append(names, name)
	}
	sort.Strings(names)

	// 2. Compute leaf hashes
	heads := make(map[string]string, len(names))
	leaves := make([]string, 0, len(names))
	for _, name := range names {
		head := a.chains[name].GetChainHead()
		heads[name] = head
		leaf := computeSHA256(name + ":" + head)
		leaves = append(leaves, leaf)
	}

	// 3. Build Merkle tree
	root := merkleRoot(leaves)

	return AnchorPoint{
		RootHash:   root,
		Chains:     heads,
		Timestamp:  time.Now(),
		ChainCount: len(names),
	}
}

// Verify checks that the current chain heads produce the same root hash
// as the given anchor point. Returns nil if they match, error otherwise.
func (a *MerkleAnchor) Verify(expected AnchorPoint) error {
	current := a.Anchor()
	if current.RootHash != expected.RootHash {
		// Find which chains diverged
		var diverged []string
		for name, expectedHead := range expected.Chains {
			if currentHead, ok := current.Chains[name]; ok {
				if currentHead != expectedHead {
					diverged = append(diverged, fmt.Sprintf("%s (expected %s, got %s)", name, expectedHead[:8], currentHead[:8]))
				}
			} else {
				diverged = append(diverged, fmt.Sprintf("%s (missing)", name))
			}
		}
		return fmt.Errorf("merkle anchor mismatch: root %s != %s; diverged chains: %s",
			current.RootHash[:16], expected.RootHash[:16], strings.Join(diverged, ", "))
	}
	return nil
}

func computeSHA256(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// merkleRoot computes the Merkle root of a list of hex-encoded hashes.
func merkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return computeSHA256("empty")
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	// Pair and hash upward
	var next []string
	for i := 0; i < len(hashes); i += 2 {
		if i+1 < len(hashes) {
			combined := hashes[i] + hashes[i+1]
			next = append(next, computeSHA256(combined))
		} else {
			// Odd element: promote as-is
			next = append(next, hashes[i])
		}
	}
	return merkleRoot(next)
}
