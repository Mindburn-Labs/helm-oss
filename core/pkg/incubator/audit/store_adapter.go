package audit

import (
	"github.com/Mindburn-Labs/helm/core/pkg/store"
)

// PersistentStore is the interface for durable audit storage backends.
//
// Production deployments MUST use a persistent implementation (Postgres, etc.).
// The in-memory store.AuditStore is suitable for testing and development only.
//
// Implementations must guarantee:
//   - Append-only semantics (no UPDATE or DELETE)
//   - Hash chain integrity (each entry links to the previous via PreviousHash)
//   - Serializable isolation for chain head reads
type PersistentStore interface {
	// Append adds a new entry to the persistent audit log.
	Append(entryType store.EntryType, subject, action string, payload interface{}, metadata map[string]string) (*store.AuditEntry, error)

	// Query retrieves entries matching the filter.
	Query(filter store.QueryFilter) []*store.AuditEntry

	// VerifyChain validates the integrity of the entire hash chain.
	VerifyChain() error

	// GetChainHead returns the hash of the most recent entry.
	GetChainHead() string

	// Size returns the number of entries in the store.
	Size() int
}

// InMemoryAdapter wraps store.AuditStore to satisfy PersistentStore.
// WARNING: This adapter is for testing/development only.
// Data is lost on process restart.
type InMemoryAdapter struct {
	store *store.AuditStore
}

// NewInMemoryAdapter creates a PersistentStore backed by an in-memory AuditStore.
func NewInMemoryAdapter() *InMemoryAdapter {
	return &InMemoryAdapter{store: store.NewAuditStore()}
}

// NewInMemoryAdapterFrom wraps an existing AuditStore.
func NewInMemoryAdapterFrom(s *store.AuditStore) *InMemoryAdapter {
	return &InMemoryAdapter{store: s}
}

func (a *InMemoryAdapter) Append(entryType store.EntryType, subject, action string, payload interface{}, metadata map[string]string) (*store.AuditEntry, error) {
	return a.store.Append(entryType, subject, action, payload, metadata)
}

func (a *InMemoryAdapter) Query(filter store.QueryFilter) []*store.AuditEntry {
	return a.store.Query(filter)
}

func (a *InMemoryAdapter) VerifyChain() error {
	return a.store.VerifyChain()
}

func (a *InMemoryAdapter) GetChainHead() string {
	return a.store.GetChainHead()
}

func (a *InMemoryAdapter) Size() int {
	return a.store.Size()
}

// Compile-time assertion.
var _ PersistentStore = (*InMemoryAdapter)(nil)

// RequirePersistentStore panics if HELM_ENV=production and the store is in-memory.
// This ensures production deployments can never silently use volatile storage.
func RequirePersistentStore(s PersistentStore, env string) {
	if env == "production" {
		if _, ok := s.(*InMemoryAdapter); ok {
			panic("audit: FATAL — in-memory audit store used in production (HELM_ENV=production). " +
				"Configure a persistent backend (Postgres) to prevent audit data loss.")
		}
	}
}
