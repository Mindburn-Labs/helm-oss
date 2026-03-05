package graph

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_PutGetEntity(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	entity := &Entity{
		ID:   "agent:test-1",
		Type: EntityAgent,
		Name: "test-agent",
		Properties: map[string]string{
			"model": "gpt-4",
		},
	}

	err := store.PutEntity(ctx, entity)
	require.NoError(t, err)

	got, err := store.GetEntity(ctx, "agent:test-1")
	require.NoError(t, err)
	assert.Equal(t, "test-agent", got.Name)
	assert.Equal(t, EntityAgent, got.Type)
	assert.Equal(t, "gpt-4", got.Properties["model"])
	assert.False(t, got.CreatedAt.IsZero())
}

func TestInMemoryStore_GetEntity_NotFound(t *testing.T) {
	store := NewInMemoryStore()
	_, err := store.GetEntity(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestInMemoryStore_PutEntity_RequiresID(t *testing.T) {
	store := NewInMemoryStore()
	err := store.PutEntity(context.Background(), &Entity{Name: "no-id"})
	assert.Error(t, err)
}

func TestInMemoryStore_DeleteEntity(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_ = store.PutEntity(ctx, &Entity{ID: "del-1", Name: "to-delete"})
	err := store.DeleteEntity(ctx, "del-1")
	require.NoError(t, err)

	_, err = store.GetEntity(ctx, "del-1")
	assert.Error(t, err)
}

func TestInMemoryStore_Relations(t *testing.T) {
	store := NewInMemoryStore()
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

	// Get relations for tool (as target).
	rels, err = store.GetRelations(ctx, "tool:t1")
	require.NoError(t, err)
	assert.Len(t, rels, 1)
}

func TestInMemoryStore_Query(t *testing.T) {
	store := NewInMemoryStore()
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
}

func TestInMemoryStore_Search(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_ = store.PutEntity(ctx, &Entity{ID: "a1", Type: EntityAgent, Name: "CodeReview Agent"})
	_ = store.PutEntity(ctx, &Entity{ID: "a2", Type: EntityAgent, Name: "DataPipeline Agent"})
	_ = store.PutEntity(ctx, &Entity{ID: "t1", Type: EntityTool, Name: "GitHub API"})

	// Search by name.
	results, err := store.Search(ctx, "code", SearchOptions{})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "CodeReview Agent", results[0].Name)

	// Search with type filter.
	results, err = store.Search(ctx, "agent", SearchOptions{Types: []EntityType{EntityAgent}})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestEntity_IsExpired(t *testing.T) {
	// Non-expiring entity.
	e1 := &Entity{TTL: 0, UpdatedAt: time.Now().Add(-24 * time.Hour)}
	assert.False(t, e1.IsExpired())

	// Pinned entity (never expires).
	e2 := &Entity{TTL: 1 * time.Minute, Pinned: true, UpdatedAt: time.Now().Add(-24 * time.Hour)}
	assert.False(t, e2.IsExpired())

	// Expired entity.
	e3 := &Entity{TTL: 1 * time.Hour, UpdatedAt: time.Now().Add(-2 * time.Hour)}
	assert.True(t, e3.IsExpired())

	// Not yet expired.
	e4 := &Entity{TTL: 1 * time.Hour, UpdatedAt: time.Now().Add(-30 * time.Minute)}
	assert.False(t, e4.IsExpired())
}

func TestComputeRelationID(t *testing.T) {
	id1 := ComputeRelationID("a", "b", "uses")
	id2 := ComputeRelationID("a", "b", "uses")
	id3 := ComputeRelationID("a", "c", "uses")

	assert.Equal(t, id1, id2, "same inputs should produce same ID")
	assert.NotEqual(t, id1, id3, "different inputs should produce different ID")
	assert.Len(t, id1, 32, "ID should be 128-bit hex")
}

func TestEntityExtractor_FromReceipt(t *testing.T) {
	extractor := NewEntityExtractor()

	payload, _ := json.Marshal(map[string]string{
		"tool":   "web_search",
		"agent":  "research-agent",
		"policy": "allow-web-search",
	})

	entities, relations := extractor.ExtractFromReceipt(payload)
	assert.Len(t, entities, 3, "should extract tool, agent, and policy entities")
	assert.Len(t, relations, 1, "should create agent→tool 'uses' relation")

	// Verify entities.
	names := make(map[string]bool)
	for _, e := range entities {
		names[e.Name] = true
	}
	assert.True(t, names["web_search"])
	assert.True(t, names["research-agent"])
	assert.True(t, names["allow-web-search"])

	// Verify relation.
	assert.Equal(t, "agent:research-agent", relations[0].SourceID)
	assert.Equal(t, "tool:web_search", relations[0].TargetID)
	assert.Equal(t, "uses", relations[0].Type)
}

func TestEntityExtractor_EmptyPayload(t *testing.T) {
	extractor := NewEntityExtractor()
	entities, relations := extractor.ExtractFromReceipt(json.RawMessage("{}"))
	assert.Empty(t, entities)
	assert.Empty(t, relations)
}

func TestQuery_TemporalDecay(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Add an expired entity.
	expired := &Entity{
		ID:        "expired-1",
		Type:      EntityFact,
		Name:      "Old Fact",
		TTL:       1 * time.Millisecond,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	store.mu.Lock()
	store.entities[expired.ID] = expired
	store.mu.Unlock()

	// Fresh entity.
	_ = store.PutEntity(ctx, &Entity{ID: "fresh-1", Type: EntityFact, Name: "Fresh Fact"})

	// Default query should exclude expired.
	result, err := store.Query(ctx, Query{EntityTypes: []EntityType{EntityFact}})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count)
	assert.Equal(t, "Fresh Fact", result.Entities[0].Name)

	// IncludeExpired should return both.
	result, err = store.Query(ctx, Query{EntityTypes: []EntityType{EntityFact}, IncludeExpired: true})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Count)
}
