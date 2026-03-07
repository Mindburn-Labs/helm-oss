package audit_test

import (
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/incubator/audit"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- PersistentStore Adapter Tests ---

func TestInMemoryAdapter_Satisfies_Interface(t *testing.T) {
	var _ audit.PersistentStore = audit.NewInMemoryAdapter()
}

func TestInMemoryAdapter_AppendAndQuery(t *testing.T) {
	adapter := audit.NewInMemoryAdapter()
	entry, err := adapter.Append(store.EntryTypeAudit, "test", "created", nil, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.EntryID)
	assert.Equal(t, 1, adapter.Size())

	results := adapter.Query(store.QueryFilter{Subject: "test"})
	assert.Len(t, results, 1)
}

func TestInMemoryAdapter_VerifyChain(t *testing.T) {
	adapter := audit.NewInMemoryAdapter()
	_, _ = adapter.Append(store.EntryTypeAudit, "a", "x", nil, nil)
	_, _ = adapter.Append(store.EntryTypeAudit, "b", "y", nil, nil)

	err := adapter.VerifyChain()
	assert.NoError(t, err)
	assert.NotEmpty(t, adapter.GetChainHead())
}

func TestRequirePersistentStore_ProductionPanics(t *testing.T) {
	adapter := audit.NewInMemoryAdapter()
	assert.Panics(t, func() {
		audit.RequirePersistentStore(adapter, "production")
	})
}

func TestRequirePersistentStore_DevOK(t *testing.T) {
	adapter := audit.NewInMemoryAdapter()
	assert.NotPanics(t, func() {
		audit.RequirePersistentStore(adapter, "development")
	})
}

// --- MerkleAnchor Tests ---

type mockChain struct{ head string }

func (c *mockChain) GetChainHead() string { return c.head }

func TestMerkleAnchor_SingleChain(t *testing.T) {
	anchor := audit.NewMerkleAnchor()
	anchor.RegisterChain("store", &mockChain{head: "abc123"})

	point := anchor.Anchor()
	assert.NotEmpty(t, point.RootHash)
	assert.Equal(t, 1, point.ChainCount)
	assert.Equal(t, "abc123", point.Chains["store"])
}

func TestMerkleAnchor_ThreeChains_Deterministic(t *testing.T) {
	anchor := audit.NewMerkleAnchor()
	anchor.RegisterChain("store", &mockChain{head: "aaa"})
	anchor.RegisterChain("guardian", &mockChain{head: "bbb"})
	anchor.RegisterChain("crypto", &mockChain{head: "ccc"})

	p1 := anchor.Anchor()
	p2 := anchor.Anchor()
	assert.Equal(t, p1.RootHash, p2.RootHash, "Merkle root must be deterministic")
	assert.Equal(t, 3, p1.ChainCount)
}

func TestMerkleAnchor_TamperDetection(t *testing.T) {
	anchor := audit.NewMerkleAnchor()
	chain := &mockChain{head: "original"}
	anchor.RegisterChain("store", chain)

	original := anchor.Anchor()

	// Tamper with chain head
	chain.head = "tampered"
	err := anchor.Verify(original)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestMerkleAnchor_Verify_OK(t *testing.T) {
	anchor := audit.NewMerkleAnchor()
	anchor.RegisterChain("a", &mockChain{head: "111"})
	anchor.RegisterChain("b", &mockChain{head: "222"})

	point := anchor.Anchor()
	err := anchor.Verify(point)
	assert.NoError(t, err)
}

func TestMerkleAnchor_Empty(t *testing.T) {
	anchor := audit.NewMerkleAnchor()
	point := anchor.Anchor()
	assert.Equal(t, "empty", point.RootHash)
	assert.Equal(t, 0, point.ChainCount)
}

func TestMerkleAnchor_NilChain(t *testing.T) {
	anchor := audit.NewMerkleAnchor()
	anchor.RegisterChain("test", nil)
	assert.Equal(t, 0, anchor.ChainCount())
}
