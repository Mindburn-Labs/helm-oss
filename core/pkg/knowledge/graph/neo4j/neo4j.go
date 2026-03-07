// Package neo4j provides a Neo4j-backed adapter for the HELM Knowledge Graph Store.
package neo4j

import (
	"context"
	"fmt"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/knowledge/graph"
)

// Store implements the graph.Store interface using Neo4j as the backend.
type Store struct {
	uri      string
	username string
	password string
}

// Config configures the Neo4j adapter.
type Config struct {
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewStore creates a new Neo4j-backed knowledge graph store.
func NewStore(cfg Config) (*Store, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("neo4j: URI is required")
	}
	return &Store{uri: cfg.URI, username: cfg.Username, password: cfg.Password}, nil
}

func (s *Store) PutEntity(_ context.Context, _ *graph.Entity) error {
	return fmt.Errorf("neo4j: PutEntity not yet implemented (requires neo4j-go-driver)")
}

func (s *Store) GetEntity(_ context.Context, _ string) (*graph.Entity, error) {
	return nil, fmt.Errorf("neo4j: GetEntity not yet implemented")
}

func (s *Store) DeleteEntity(_ context.Context, _ string) error {
	return fmt.Errorf("neo4j: DeleteEntity not yet implemented")
}

func (s *Store) PutRelation(_ context.Context, _ *graph.Relation) error {
	return fmt.Errorf("neo4j: PutRelation not yet implemented")
}

func (s *Store) GetRelations(_ context.Context, _ string) ([]*graph.Relation, error) {
	return nil, fmt.Errorf("neo4j: GetRelations not yet implemented")
}

func (s *Store) Query(_ context.Context, _ graph.Query) (*graph.QueryResult, error) {
	return nil, fmt.Errorf("neo4j: Query not yet implemented")
}

func (s *Store) Search(_ context.Context, _ string, _ graph.SearchOptions) ([]*graph.Entity, error) {
	return nil, fmt.Errorf("neo4j: Search not yet implemented")
}

// Compile-time interface satisfaction check.
var _ graph.Store = (*Store)(nil)
