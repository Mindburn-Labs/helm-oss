package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// validPropertyKey restricts JSON property keys to safe identifiers,
// preventing SQL injection via json_extract path interpolation.
var validPropertyKey = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// SQLiteStore implements the knowledge graph Store using SQLite and optionally FTS5.
type SQLiteStore struct {
	db      *sql.DB
	hasFTS5 bool // true if FTS5 is available
}

// NewSQLiteStore creates a new SQLiteStore and initializes the schema if needed.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Require foreign keys.
	dsn := fmt.Sprintf("file:%s?_fk=1", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("knowledge: open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("knowledge: ping sqlite: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	coreSchema := `
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		properties JSON,
		provenance_node_id TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		ttl INTEGER NOT NULL DEFAULT 0,
		pinned BOOLEAN NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);
	CREATE INDEX IF NOT EXISTS idx_entities_updated_at ON entities(updated_at);

	CREATE TABLE IF NOT EXISTS relations (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		type TEXT NOT NULL,
		properties JSON,
		weight REAL NOT NULL DEFAULT 1.0,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (source_id) REFERENCES entities(id) ON DELETE CASCADE,
		FOREIGN KEY (target_id) REFERENCES entities(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_relations_source ON relations(source_id);
	CREATE INDEX IF NOT EXISTS idx_relations_target ON relations(target_id);
	CREATE INDEX IF NOT EXISTS idx_relations_type ON relations(type);
	`

	_, err := s.db.Exec(coreSchema)
	if err != nil {
		return fmt.Errorf("knowledge: init schema: %w", err)
	}

	// Attempt FTS5 — gracefully degrade if not available
	ftsSchema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS entities_fts USING fts5(
		id UNINDEXED,
		name,
		properties,
		content='entities',
		content_rowid='rowid'
	);

	-- Triggers to automatically update FTS index
	CREATE TRIGGER IF NOT EXISTS entities_ai AFTER INSERT ON entities BEGIN
		INSERT INTO entities_fts(rowid, id, name, properties) VALUES (new.rowid, new.id, new.name, new.properties);
	END;
	CREATE TRIGGER IF NOT EXISTS entities_ad AFTER DELETE ON entities BEGIN
		INSERT INTO entities_fts(entities_fts, rowid, id, name, properties) VALUES('delete', old.rowid, old.id, old.name, old.properties);
	END;
	CREATE TRIGGER IF NOT EXISTS entities_au AFTER UPDATE ON entities BEGIN
		INSERT INTO entities_fts(entities_fts, rowid, id, name, properties) VALUES('delete', old.rowid, old.id, old.name, old.properties);
		INSERT INTO entities_fts(rowid, id, name, properties) VALUES (new.rowid, new.id, new.name, new.properties);
	END;
	`

	_, ftsErr := s.db.Exec(ftsSchema)
	s.hasFTS5 = ftsErr == nil

	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) PutEntity(ctx context.Context, entity *Entity) error {
	if entity.ID == "" {
		return fmt.Errorf("knowledge: entity ID required")
	}

	now := time.Now()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = now
	}
	entity.UpdatedAt = now

	propsJSON, err := json.Marshal(entity.Properties)
	if err != nil {
		return fmt.Errorf("knowledge: marshal properties: %w", err)
	}

	query := `
		INSERT INTO entities (id, type, name, properties, provenance_node_id, created_at, updated_at, ttl, pinned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type=excluded.type,
			name=excluded.name,
			properties=excluded.properties,
			provenance_node_id=excluded.provenance_node_id,
			updated_at=excluded.updated_at,
			ttl=excluded.ttl,
			pinned=excluded.pinned
	`
	_, err = s.db.ExecContext(ctx, query,
		entity.ID, string(entity.Type), entity.Name, string(propsJSON), entity.ProvenanceNodeID,
		entity.CreatedAt, entity.UpdatedAt, int64(entity.TTL), entity.Pinned,
	)
	if err != nil {
		return fmt.Errorf("knowledge: put entity: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetEntity(ctx context.Context, id string) (*Entity, error) {
	query := `SELECT id, type, name, properties, provenance_node_id, created_at, updated_at, ttl, pinned FROM entities WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	var e Entity
	var t string
	var propsJSON []byte
	var ttl int64
	var provNodeID sql.NullString

	err := row.Scan(&e.ID, &t, &e.Name, &propsJSON, &provNodeID, &e.CreatedAt, &e.UpdatedAt, &ttl, &e.Pinned)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("knowledge: entity %s not found", id)
		}
		return nil, fmt.Errorf("knowledge: get entity: %w", err)
	}

	e.Type = EntityType(t)
	e.TTL = time.Duration(ttl)
	if provNodeID.Valid {
		e.ProvenanceNodeID = provNodeID.String
	}

	if len(propsJSON) > 0 {
		if err := json.Unmarshal(propsJSON, &e.Properties); err != nil {
			return nil, fmt.Errorf("knowledge: unmarshal properties: %w", err)
		}
	} else {
		e.Properties = make(map[string]string)
	}

	return &e, nil
}

func (s *SQLiteStore) DeleteEntity(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM entities WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) PutRelation(ctx context.Context, rel *Relation) error {
	if rel.ID == "" {
		rel.ID = ComputeRelationID(rel.SourceID, rel.TargetID, rel.Type)
	}
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = time.Now()
	}

	propsJSON, err := json.Marshal(rel.Properties)
	if err != nil {
		return fmt.Errorf("knowledge: marshal relation properties: %w", err)
	}

	query := `
		INSERT INTO relations (id, source_id, target_id, type, properties, weight, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			properties=excluded.properties,
			weight=excluded.weight
	`
	_, err = s.db.ExecContext(ctx, query,
		rel.ID, rel.SourceID, rel.TargetID, rel.Type, string(propsJSON), rel.Weight, rel.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("knowledge: put relation: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetRelations(ctx context.Context, entityID string) ([]*Relation, error) {
	query := `SELECT id, source_id, target_id, type, properties, weight, created_at FROM relations WHERE source_id = ? OR target_id = ?`
	rows, err := s.db.QueryContext(ctx, query, entityID, entityID)
	if err != nil {
		return nil, fmt.Errorf("knowledge: get relations: %w", err)
	}
	defer rows.Close()

	var results []*Relation
	for rows.Next() {
		var r Relation
		var propsJSON []byte
		if err := rows.Scan(&r.ID, &r.SourceID, &r.TargetID, &r.Type, &propsJSON, &r.Weight, &r.CreatedAt); err != nil {
			return nil, err
		}
		if len(propsJSON) > 0 {
			_ = json.Unmarshal(propsJSON, &r.Properties)
		} else {
			r.Properties = make(map[string]string)
		}
		results = append(results, &r)
	}
	return results, nil
}

func (s *SQLiteStore) Query(ctx context.Context, q Query) (*QueryResult, error) {
	var conditions []string
	var args []interface{}

	if len(q.EntityTypes) > 0 {
		placeholders := make([]string, len(q.EntityTypes))
		for i, typ := range q.EntityTypes {
			placeholders[i] = "?"
			args = append(args, string(typ))
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(placeholders, ",")))
	}

	if q.TimeFrom != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *q.TimeFrom)
	}
	if q.TimeTo != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *q.TimeTo)
	}

	if len(q.Properties) > 0 {
		for k, v := range q.Properties {
			if !validPropertyKey.MatchString(k) {
				return nil, fmt.Errorf("knowledge: invalid property key %q", k)
			}
			// json_extract requires SQLite 3.38.0+
			conditions = append(conditions, fmt.Sprintf("json_extract(properties, '$.%s') = ?", k))
			args = append(args, v)
		}
	}

	// Handle temporal decay filtering by default
	if !q.IncludeExpired {
		// A record is somewhat active if Pinned=1, TTL=0, OR updated_at > Now - TTL
		// Since we cannot easily do duration arithmetic in standard SQLite simply without generating complex SQL,
		// we fetch all that pass other filters and filter in memory, or we can use SQLite's datetime mapping.
		// For robustness, we will fetch and filter in-memory for TTL if not pinned and TTL > 0.
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`SELECT id, type, name, properties, provenance_node_id, created_at, updated_at, ttl, pinned FROM entities %s`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("knowledge: query: %w", err)
	}
	defer rows.Close()

	result := &QueryResult{}

	for rows.Next() {
		var e Entity
		var t string
		var propsJSON []byte
		var ttl int64
		var provNodeID sql.NullString

		if err := rows.Scan(&e.ID, &t, &e.Name, &propsJSON, &provNodeID, &e.CreatedAt, &e.UpdatedAt, &ttl, &e.Pinned); err != nil {
			return nil, err
		}

		e.Type = EntityType(t)
		e.TTL = time.Duration(ttl)
		if provNodeID.Valid {
			e.ProvenanceNodeID = provNodeID.String
		}
		if len(propsJSON) > 0 {
			_ = json.Unmarshal(propsJSON, &e.Properties)
		} else {
			e.Properties = make(map[string]string)
		}

		if !q.IncludeExpired && e.IsExpired() {
			continue
		}

		result.Entities = append(result.Entities, &e)
		if q.Limit > 0 && len(result.Entities) >= q.Limit {
			break
		}
	}

	result.Count = len(result.Entities)
	return result, nil
}

func (s *SQLiteStore) Search(ctx context.Context, query string, opts SearchOptions) ([]*Entity, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}

	var sqlQuery string
	var args []interface{}

	if s.hasFTS5 {
		// FTS5 path
		matchQuery := strings.ReplaceAll(query, `"`, `""`)
		terms := strings.Fields(matchQuery)
		for i, t := range terms {
			terms[i] = t + "*"
		}
		ftsExpr := strings.Join(terms, " ")

		var conditions []string
		conditions = append(conditions, "entities_fts MATCH ?")
		args = append(args, ftsExpr)

		if len(opts.Types) > 0 {
			placeholders := make([]string, len(opts.Types))
			for i, typ := range opts.Types {
				placeholders[i] = "?"
				args = append(args, string(typ))
			}
			conditions = append(conditions, fmt.Sprintf("e.type IN (%s)", strings.Join(placeholders, ",")))
		}

		whereClause := "WHERE " + strings.Join(conditions, " AND ")
		args = append(args, limit)

		sqlQuery = fmt.Sprintf(`
			SELECT e.id, e.type, e.name, e.properties, e.provenance_node_id, e.created_at, e.updated_at, e.ttl, e.pinned
			FROM entities_fts f
			JOIN entities e ON e.rowid = f.rowid
			%s
			ORDER BY rank
			LIMIT ?
		`, whereClause)
	} else {
		// LIKE fallback — search name and properties
		likePattern := "%" + query + "%"
		var conditions []string
		conditions = append(conditions, "(name LIKE ? OR properties LIKE ?)")
		args = append(args, likePattern, likePattern)

		if len(opts.Types) > 0 {
			placeholders := make([]string, len(opts.Types))
			for i, typ := range opts.Types {
				placeholders[i] = "?"
				args = append(args, string(typ))
			}
			conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(placeholders, ",")))
		}

		whereClause := "WHERE " + strings.Join(conditions, " AND ")
		args = append(args, limit)

		sqlQuery = fmt.Sprintf(`
			SELECT id, type, name, properties, provenance_node_id, created_at, updated_at, ttl, pinned
			FROM entities
			%s
			LIMIT ?
		`, whereClause)
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("knowledge: search: %w", err)
	}
	defer rows.Close()

	var results []*Entity
	for rows.Next() {
		var e Entity
		var t string
		var propsJSON []byte
		var ttl int64
		var provNodeID sql.NullString

		if err := rows.Scan(&e.ID, &t, &e.Name, &propsJSON, &provNodeID, &e.CreatedAt, &e.UpdatedAt, &ttl, &e.Pinned); err != nil {
			return nil, err
		}

		e.Type = EntityType(t)
		e.TTL = time.Duration(ttl)
		if provNodeID.Valid {
			e.ProvenanceNodeID = provNodeID.String
		}
		if len(propsJSON) > 0 {
			_ = json.Unmarshal(propsJSON, &e.Properties)
		} else {
			e.Properties = make(map[string]string)
		}

		if e.IsExpired() {
			continue
		}
		results = append(results, &e)
	}

	return results, nil
}
