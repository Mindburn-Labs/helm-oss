// Package graph provides a graph-backed knowledge store for HELM.
//
// It implements entity extraction from ProofGraph receipts and policies,
// semantic search with provenance tracking, and RAG pipeline integration.
// This is the core of the P6 Knowledge plane.
package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// EntityType classifies knowledge entities.
type EntityType string

const (
	EntityAgent    EntityType = "agent"
	EntityTool     EntityType = "tool"
	EntityPolicy   EntityType = "policy"
	EntityModel    EntityType = "model"
	EntityDocument EntityType = "document"
	EntityOrg      EntityType = "organization"
	EntityIncident EntityType = "incident"
	EntityFact     EntityType = "fact"
)

// Entity is a node in the knowledge graph.
type Entity struct {
	// ID is the unique entity identifier (content-addressed hash).
	ID string `json:"id"`

	// Type classifies the entity.
	Type EntityType `json:"type"`

	// Name is the human-readable entity name.
	Name string `json:"name"`

	// Properties are key-value attributes.
	Properties map[string]string `json:"properties,omitempty"`

	// Embedding is the vector embedding for semantic search (optional).
	Embedding []float32 `json:"embedding,omitempty"`

	// ProvenanceNodeID links to the ProofGraph node that created this entity.
	ProvenanceNodeID string `json:"provenance_node_id,omitempty"`

	// CreatedAt is when the entity was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the entity was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// TTL is the temporal decay duration. Zero means no expiry.
	TTL time.Duration `json:"ttl,omitempty"`

	// Pinned prevents temporal decay from expiring this entity.
	Pinned bool `json:"pinned,omitempty"`
}

// IsExpired checks if the entity has expired based on TTL.
func (e *Entity) IsExpired() bool {
	if e.Pinned || e.TTL == 0 {
		return false
	}
	return time.Since(e.UpdatedAt) > e.TTL
}

// Relation is an edge in the knowledge graph.
type Relation struct {
	// ID is a deterministic hash of (SourceID, TargetID, Type).
	ID string `json:"id"`

	// SourceID is the source entity ID.
	SourceID string `json:"source_id"`

	// TargetID is the target entity ID.
	TargetID string `json:"target_id"`

	// Type describes the relationship (e.g., "uses", "governs", "owns", "violates").
	Type string `json:"type"`

	// Properties are edge attributes.
	Properties map[string]string `json:"properties,omitempty"`

	// Weight is the relationship strength (0.0 to 1.0).
	Weight float64 `json:"weight,omitempty"`

	// CreatedAt is when the relation was created.
	CreatedAt time.Time `json:"created_at"`
}

// ComputeRelationID creates a deterministic ID for a relation.
func ComputeRelationID(sourceID, targetID, relType string) string {
	data := fmt.Sprintf("%s:%s:%s", sourceID, targetID, relType)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:16]) // 128-bit ID
}

// Store is the interface for knowledge graph persistence.
type Store interface {
	// PutEntity stores or updates an entity.
	PutEntity(ctx context.Context, entity *Entity) error

	// GetEntity retrieves an entity by ID.
	GetEntity(ctx context.Context, id string) (*Entity, error)

	// DeleteEntity removes an entity.
	DeleteEntity(ctx context.Context, id string) error

	// PutRelation stores or updates a relation.
	PutRelation(ctx context.Context, rel *Relation) error

	// GetRelations returns all relations for an entity (as source or target).
	GetRelations(ctx context.Context, entityID string) ([]*Relation, error)

	// Query executes a knowledge graph query.
	Query(ctx context.Context, q Query) (*QueryResult, error)

	// Search performs semantic/keyword search across entities.
	Search(ctx context.Context, query string, opts SearchOptions) ([]*Entity, error)
}

// Query represents a knowledge graph query.
type Query struct {
	// EntityTypes filters by entity type.
	EntityTypes []EntityType `json:"entity_types,omitempty"`

	// RelationType filters by relation type.
	RelationType string `json:"relation_type,omitempty"`

	// Properties filters entities by property values.
	Properties map[string]string `json:"properties,omitempty"`

	// TimeRange filters by creation time.
	TimeFrom *time.Time `json:"time_from,omitempty"`
	TimeTo   *time.Time `json:"time_to,omitempty"`

	// Limit is the maximum number of results.
	Limit int `json:"limit,omitempty"`

	// IncludeExpired includes entities past their TTL.
	IncludeExpired bool `json:"include_expired,omitempty"`
}

// QueryResult is the result of a knowledge graph query.
type QueryResult struct {
	Entities  []*Entity   `json:"entities"`
	Relations []*Relation `json:"relations,omitempty"`
	Count     int         `json:"count"`
}

// SearchOptions configures semantic search behavior.
type SearchOptions struct {
	// Types filters by entity type.
	Types []EntityType `json:"types,omitempty"`

	// Limit is the maximum number of results.
	Limit int `json:"limit,omitempty"`

	// MinScore is the minimum relevance score (0.0 to 1.0).
	MinScore float64 `json:"min_score,omitempty"`
}

// InMemoryStore is an in-memory knowledge graph implementation for development and testing.
type InMemoryStore struct {
	mu        sync.RWMutex
	entities  map[string]*Entity
	relations map[string]*Relation
}

// NewInMemoryStore creates a new in-memory knowledge store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entities:  make(map[string]*Entity),
		relations: make(map[string]*Relation),
	}
}

func (s *InMemoryStore) PutEntity(_ context.Context, entity *Entity) error {
	if entity.ID == "" {
		return fmt.Errorf("knowledge: entity ID required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = time.Now()
	}
	entity.UpdatedAt = time.Now()
	s.entities[entity.ID] = entity
	return nil
}

func (s *InMemoryStore) GetEntity(_ context.Context, id string) (*Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entities[id]
	if !ok {
		return nil, fmt.Errorf("knowledge: entity %s not found", id)
	}
	return e, nil
}

func (s *InMemoryStore) DeleteEntity(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entities, id)
	return nil
}

func (s *InMemoryStore) PutRelation(_ context.Context, rel *Relation) error {
	if rel.ID == "" {
		rel.ID = ComputeRelationID(rel.SourceID, rel.TargetID, rel.Type)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = time.Now()
	}
	s.relations[rel.ID] = rel
	return nil
}

func (s *InMemoryStore) GetRelations(_ context.Context, entityID string) ([]*Relation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*Relation
	for _, r := range s.relations {
		if r.SourceID == entityID || r.TargetID == entityID {
			results = append(results, r)
		}
	}
	return results, nil
}

func (s *InMemoryStore) Query(_ context.Context, q Query) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &QueryResult{}

	for _, e := range s.entities {
		if !q.IncludeExpired && e.IsExpired() {
			continue
		}
		if len(q.EntityTypes) > 0 && !containsType(q.EntityTypes, e.Type) {
			continue
		}
		if q.TimeFrom != nil && e.CreatedAt.Before(*q.TimeFrom) {
			continue
		}
		if q.TimeTo != nil && e.CreatedAt.After(*q.TimeTo) {
			continue
		}
		if q.Properties != nil {
			match := true
			for k, v := range q.Properties {
				if e.Properties[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		result.Entities = append(result.Entities, e)
		if q.Limit > 0 && len(result.Entities) >= q.Limit {
			break
		}
	}

	result.Count = len(result.Entities)
	return result, nil
}

func (s *InMemoryStore) Search(_ context.Context, query string, opts SearchOptions) ([]*Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*Entity
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}

	for _, e := range s.entities {
		if e.IsExpired() {
			continue
		}
		if len(opts.Types) > 0 && !containsType(opts.Types, e.Type) {
			continue
		}
		// Simple keyword matching for in-memory; production uses embeddings.
		if strings.Contains(strings.ToLower(e.Name), query) {
			results = append(results, e)
		} else {
			for _, v := range e.Properties {
				if strings.Contains(strings.ToLower(v), query) {
					results = append(results, e)
					break
				}
			}
		}
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func containsType(types []EntityType, t EntityType) bool {
	for _, tt := range types {
		if tt == t {
			return true
		}
	}
	return false
}

// EntityExtractor extracts knowledge entities from ProofGraph-related data.
type EntityExtractor struct{}

// NewEntityExtractor creates a new entity extractor.
func NewEntityExtractor() *EntityExtractor { return &EntityExtractor{} }

// ExtractFromReceipt parses a receipt payload and extracts knowledge entities.
func (e *EntityExtractor) ExtractFromReceipt(payload json.RawMessage) ([]*Entity, []*Relation) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, nil
	}

	var entities []*Entity
	var relations []*Relation
	now := time.Now()

	// Extract tool entity.
	if tool, ok := data["tool"].(string); ok && tool != "" {
		entities = append(entities, &Entity{
			ID:   "tool:" + tool,
			Type: EntityTool,
			Name: tool,
			Properties: map[string]string{
				"source": "receipt",
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Extract agent entity.
	if agent, ok := data["agent"].(string); ok && agent != "" {
		entities = append(entities, &Entity{
			ID:   "agent:" + agent,
			Type: EntityAgent,
			Name: agent,
			Properties: map[string]string{
				"source": "receipt",
			},
			CreatedAt: now,
			UpdatedAt: now,
		})

		// Create uses relation if both tool and agent exist.
		if tool, ok := data["tool"].(string); ok && tool != "" {
			relations = append(relations, &Relation{
				SourceID:  "agent:" + agent,
				TargetID:  "tool:" + tool,
				Type:      "uses",
				CreatedAt: now,
			})
		}
	}

	// Extract policy entity.
	if policy, ok := data["policy"].(string); ok && policy != "" {
		entities = append(entities, &Entity{
			ID:   "policy:" + policy,
			Type: EntityPolicy,
			Name: policy,
			Properties: map[string]string{
				"source": "receipt",
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	return entities, relations
}
