package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ProofNodeKind categorizes ProofGraph nodes.
type ProofNodeKind string

const (
	ProofNodeManifest    ProofNodeKind = "MANIFEST"
	ProofNodeMechanical  ProofNodeKind = "MECHANICAL_AUDIT"
	ProofNodeAIAudit     ProofNodeKind = "AI_AUDIT"
	ProofNodeCrossVerif  ProofNodeKind = "CROSS_VERIFICATION"
	ProofNodeMerge       ProofNodeKind = "MERGE"
	ProofNodeConform     ProofNodeKind = "CONFORM"
	ProofNodeConsensus   ProofNodeKind = "CONSENSUS"
	ProofNodeAttestation ProofNodeKind = "ATTESTATION"
)

// ProofNode is a single node in the tamper-evident audit DAG.
type ProofNode struct {
	ID           string            `json:"id"`
	Kind         ProofNodeKind     `json:"kind"`
	Hash         string            `json:"hash"`
	ParentHashes []string          `json:"parent_hashes,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ProofGraph is a typed, tamper-evident DAG of audit evidence.
//
// Each audit run produces nodes linked by parent hashes:
//
//	ManifestNode (sha256 of git ls-files)
//	  ├── MechanicalAuditNode (25 sections)
//	  ├── AIAuditNode (7 missions)
//	  │     └── CrossVerifyNode (Layer 2.5)
//	  ├── MergeNode (FINAL_AUDIT_REPORT)
//	  │     └── chains to MerkleAnchor root
//	  └── ConformNode (L1/L3 results)
//
// Thread-safe for concurrent node additions.
type ProofGraph struct {
	mu    sync.RWMutex
	nodes map[string]*ProofNode // id → node
	heads []string              // current leaf hashes (no children)
}

// NewProofGraph creates an empty ProofGraph.
func NewProofGraph() *ProofGraph {
	return &ProofGraph{
		nodes: make(map[string]*ProofNode),
	}
}

// AddNode adds a typed node to the graph.
// The node's hash is computed from its content + parent hashes.
func (g *ProofGraph) AddNode(kind ProofNodeKind, metadata map[string]string, parentIDs ...string) (*ProofNode, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Resolve parent hashes
	parentHashes := make([]string, 0, len(parentIDs))
	for _, pid := range parentIDs {
		parent, ok := g.nodes[pid]
		if !ok {
			return nil, fmt.Errorf("proofgraph: parent node %q not found", pid)
		}
		parentHashes = append(parentHashes, parent.Hash)
	}

	node := &ProofNode{
		Kind:         kind,
		ParentHashes: parentHashes,
		Timestamp:    time.Now().UTC(),
		Metadata:     metadata,
	}

	// Compute node hash
	hash, err := computeNodeHash(node)
	if err != nil {
		return nil, fmt.Errorf("proofgraph: hash computation failed: %w", err)
	}
	node.Hash = hash
	node.ID = hash[:16] // Short ID from hash

	g.nodes[node.ID] = node

	// Update heads: remove parents from heads, add this node
	g.heads = removeAll(g.heads, parentIDs...)
	g.heads = append(g.heads, node.ID)

	return node, nil
}

// GetNode retrieves a node by ID.
func (g *ProofGraph) GetNode(id string) (*ProofNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[id]
	return n, ok
}

// Heads returns the current leaf node IDs (nodes with no children).
func (g *ProofGraph) Heads() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]string, len(g.heads))
	copy(out, g.heads)
	return out
}

// Size returns the number of nodes.
func (g *ProofGraph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// Verify checks the integrity of the entire graph.
// Each node's stored hash must match its recomputed hash,
// and all parent references must be valid.
func (g *ProofGraph) Verify() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for id, node := range g.nodes {
		// Recompute hash
		computed, err := computeNodeHash(node)
		if err != nil {
			return fmt.Errorf("proofgraph: node %s hash computation failed: %w", id, err)
		}
		if computed != node.Hash {
			return fmt.Errorf("proofgraph: node %s hash mismatch (computed %s, stored %s)",
				id, computed[:16], node.Hash[:16])
		}

		// Verify parent references exist
		for _, ph := range node.ParentHashes {
			found := false
			for _, n := range g.nodes {
				if n.Hash == ph {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("proofgraph: node %s references unknown parent hash %s", id, ph[:16])
			}
		}
	}
	return nil
}

// Export serializes the graph to JSON.
func (g *ProofGraph) Export() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	export := struct {
		Version string       `json:"version"`
		Nodes   []*ProofNode `json:"nodes"`
		Heads   []string     `json:"heads"`
		Count   int          `json:"count"`
	}{
		Version: "1.0.0",
		Nodes:   make([]*ProofNode, 0, len(g.nodes)),
		Heads:   g.heads,
		Count:   len(g.nodes),
	}
	for _, n := range g.nodes {
		export.Nodes = append(export.Nodes, n)
	}
	return json.MarshalIndent(export, "", "  ")
}

func computeNodeHash(node *ProofNode) (string, error) {
	hashable := struct {
		Kind         ProofNodeKind     `json:"kind"`
		ParentHashes []string          `json:"parent_hashes"`
		Timestamp    time.Time         `json:"timestamp"`
		Metadata     map[string]string `json:"metadata"`
	}{
		Kind:         node.Kind,
		ParentHashes: node.ParentHashes,
		Timestamp:    node.Timestamp,
		Metadata:     node.Metadata,
	}

	data, err := json.Marshal(hashable)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

func removeAll(slice []string, items ...string) []string {
	result := make([]string, 0, len(slice))
	remove := make(map[string]bool, len(items))
	for _, item := range items {
		remove[item] = true
	}
	for _, s := range slice {
		if !remove[s] {
			result = append(result, s)
		}
	}
	return result
}
