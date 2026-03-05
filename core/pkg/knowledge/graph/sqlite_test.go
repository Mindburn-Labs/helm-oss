package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareSQLiteStore(t *testing.T) (*SQLiteStore, func()) {
	dir, err := os.MkdirTemp("", "knowledge-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(dir, "knowledge.db")
	store, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)

	cleanup := func() {
		store.Close()
		os.RemoveAll(dir)
	}
	return store, cleanup
}

func TestSQLiteStore_PutGetEntity(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	ctx := context.Background()

	entity := &Entity{
		ID:   "agent:test-1",
		Type: EntityAgent,
		Name: "test-agent",
		Properties: map[string]string{
			"model": "gpt-4",
		},
		TTL:    24 * time.Hour,
		Pinned: true,
	}

	err := store.PutEntity(ctx, entity)
	require.NoError(t, err)

	got, err := store.GetEntity(ctx, "agent:test-1")
	require.NoError(t, err)
	assert.Equal(t, "test-agent", got.Name)
	assert.Equal(t, EntityAgent, got.Type)
	assert.Equal(t, "gpt-4", got.Properties["model"])
	assert.Equal(t, 24*time.Hour, got.TTL)
	assert.True(t, got.Pinned)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestSQLiteStore_GetEntity_NotFound(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	_, err := store.GetEntity(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestSQLiteStore_PutEntity_RequiresID(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	err := store.PutEntity(context.Background(), &Entity{Name: "no-id"})
	assert.Error(t, err)
}

func TestSQLiteStore_DeleteEntity(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	ctx := context.Background()

	_ = store.PutEntity(ctx, &Entity{ID: "del-1", Name: "to-delete"})
	err := store.DeleteEntity(ctx, "del-1")
	require.NoError(t, err)

	_, err = store.GetEntity(ctx, "del-1")
	assert.Error(t, err)
}

func TestSQLiteStore_Relations(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	ctx := context.Background()

	_ = store.PutEntity(ctx, &Entity{ID: "agent:a1", Type: EntityAgent, Name: "Agent A"})
	_ = store.PutEntity(ctx, &Entity{ID: "tool:t1", Type: EntityTool, Name: "Tool T"})

	rel := &Relation{
		SourceID: "agent:a1",
		TargetID: "tool:t1",
		Type:     "uses",
		Weight:   0.8,
	}
	err := store.PutRelation(ctx, rel)
	require.NoError(t, err)
	assert.NotEmpty(t, rel.ID, "ID should be auto-generated")

	// Get relations for agent.
	rels, err := store.GetRelations(ctx, "agent:a1")
	require.NoError(t, err)
	assert.Len(t, rels, 1)
	assert.Equal(t, "uses", rels[0].Type)
	assert.Equal(t, 0.8, rels[0].Weight)

	// Get relations for tool (as target).
	rels, err = store.GetRelations(ctx, "tool:t1")
	require.NoError(t, err)
	assert.Len(t, rels, 1)
}

func TestSQLiteStore_Query(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	ctx := context.Background()

	_ = store.PutEntity(ctx, &Entity{ID: "a1", Type: EntityAgent, Name: "Agent 1"})
	_ = store.PutEntity(ctx, &Entity{ID: "a2", Type: EntityAgent, Name: "Agent 2"})
	_ = store.PutEntity(ctx, &Entity{ID: "t1", Type: EntityTool, Name: "Tool 1"})

	// Query by type.
	result, err := store.Query(ctx, Query{EntityTypes: []EntityType{EntityAgent}})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Count)

	// Query with limit.
	result, err = store.Query(ctx, Query{EntityTypes: []EntityType{EntityAgent}, Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count)

	// Query by properties.
	_ = store.PutEntity(ctx, &Entity{
		ID: "p1", Type: EntityPolicy, Name: "Policy 1",
		Properties: map[string]string{"framework": "eu-ai-act"},
	})
	result, err = store.Query(ctx, Query{
		Properties: map[string]string{"framework": "eu-ai-act"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count)
	assert.Equal(t, "p1", result.Entities[0].ID)
}

func TestSQLiteStore_Search(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	ctx := context.Background()

	_ = store.PutEntity(ctx, &Entity{ID: "a1", Type: EntityAgent, Name: "CodeReview Agent", Properties: map[string]string{"desc": "reviews PRs"}})
	_ = store.PutEntity(ctx, &Entity{ID: "a2", Type: EntityAgent, Name: "Data Pipeline Agent", Properties: map[string]string{"desc": "moves data"}})
	_ = store.PutEntity(ctx, &Entity{ID: "t1", Type: EntityTool, Name: "GitHub API", Properties: map[string]string{"desc": "code hosting"}})

	// Search by name using FTS5 match (prefix match logic in Search adds wildcard)
	results, err := store.Search(ctx, "code", SearchOptions{})
	require.NoError(t, err)
	assert.Len(t, results, 2, "CodeReview Agent and GitHub API (desc: code hosting) should both match")

	// Verify both exist somewhat
	foundNames := make(map[string]bool)
	for _, res := range results {
		foundNames[res.Name] = true
	}
	assert.True(t, foundNames["CodeReview Agent"])
	assert.True(t, foundNames["GitHub API"])

	// Search by name word
	results, err = store.Search(ctx, "review", SearchOptions{})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "CodeReview Agent", results[0].Name)

	// Search with type filter
	results, err = store.Search(ctx, "pipeline", SearchOptions{Types: []EntityType{EntityAgent}})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSQLiteStore_TemporalDecay(t *testing.T) {
	store, cleanup := prepareSQLiteStore(t)
	defer cleanup()

	ctx := context.Background()

	expired := &Entity{
		ID:   "expired-1",
		Type: EntityFact,
		Name: "Old Fact",
		TTL:  1 * time.Millisecond, // very short TTL
	}
	err := store.PutEntity(ctx, expired)
	require.NoError(t, err)

	// Backdate the updated_at timestamp directly in the database to simulate expiration
	// since PutEntity intentionally sets it to time.Now().
	_, err = store.db.ExecContext(ctx, "UPDATE entities SET updated_at = ? WHERE id = ?", time.Now().Add(-1*time.Hour), "expired-1")
	require.NoError(t, err)

	// Add a fresh entity
	_ = store.PutEntity(ctx, &Entity{ID: "fresh-1", Type: EntityFact, Name: "Fresh Fact", TTL: 24 * time.Hour})

	// Default query should exclude expired.
	result, err := store.Query(ctx, Query{EntityTypes: []EntityType{EntityFact}})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count)
	assert.Equal(t, "Fresh Fact", result.Entities[0].Name)

	// IncludeExpired should return both.
	result, err = store.Query(ctx, Query{EntityTypes: []EntityType{EntityFact}, IncludeExpired: true})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Count)

	// FTS Search should also exclude expired
	results, err := store.Search(ctx, "Fact", SearchOptions{})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Fresh Fact", results[0].Name)
}
