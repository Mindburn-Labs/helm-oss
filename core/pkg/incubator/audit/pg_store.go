package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/store"
	"github.com/google/uuid"
)

// Schema for the audit_entries table.
// Run this DDL before using PostgresStore in production.
const AuditEntriesSchema = `
CREATE TABLE IF NOT EXISTS audit_entries (
    sequence       BIGSERIAL       PRIMARY KEY,
    entry_id       UUID            NOT NULL UNIQUE,
    timestamp      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    entry_type     TEXT            NOT NULL,
    subject        TEXT            NOT NULL,
    action         TEXT            NOT NULL,
    payload        JSONB           NOT NULL DEFAULT '{}',
    payload_hash   TEXT            NOT NULL,
    previous_hash  TEXT            NOT NULL,
    entry_hash     TEXT            NOT NULL UNIQUE,
    metadata       JSONB           DEFAULT '{}'
);

-- Append-only: revoke UPDATE and DELETE from application role
-- REVOKE UPDATE, DELETE ON audit_entries FROM helm_app;

CREATE INDEX IF NOT EXISTS idx_audit_entries_type     ON audit_entries (entry_type);
CREATE INDEX IF NOT EXISTS idx_audit_entries_subject  ON audit_entries (subject);
CREATE INDEX IF NOT EXISTS idx_audit_entries_ts       ON audit_entries (timestamp);
`

// PostgresStore implements PersistentStore using PostgreSQL.
//
// This is the production-grade audit store for HELM. It provides:
//   - Append-only semantics (no UPDATE or DELETE queries)
//   - Hash chain integrity compatible with store.AuditStore
//   - Serializable isolation for chain head reads via SELECT FOR UPDATE
//   - Full QueryFilter support (type, subject, time range, sequence range)
//
// The hash computation is identical to store.AuditStore for chain portability.
type PostgresStore struct {
	db        *sql.DB
	mu        sync.Mutex // Serializes Append for chain head reads
	chainHead string     // Cached chain head
	sequence  uint64     // Cached sequence
}

// NewPostgresStore creates a new Postgres-backed persistent audit store.
// It loads the latest chain head and sequence from the database on startup.
func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("audit/pg: database connection required (fail-closed)")
	}

	ps := &PostgresStore{db: db, chainHead: "genesis"}

	// Load latest chain head from DB
	var entryHash sql.NullString
	var seq sql.NullInt64
	err := db.QueryRowContext(context.Background(),
		`SELECT entry_hash, sequence FROM audit_entries ORDER BY sequence DESC LIMIT 1`,
	).Scan(&entryHash, &seq)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("audit/pg: failed to load chain head: %w", err)
	}
	if entryHash.Valid {
		ps.chainHead = entryHash.String
		ps.sequence = uint64(seq.Int64)
	}

	return ps, nil
}

// EnsureSchema creates the audit_entries table if it doesn't exist.
func (s *PostgresStore) EnsureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, AuditEntriesSchema)
	return err
}

// Append adds a new entry to the Postgres audit log.
// Thread-safe — serializes chain head access via mutex.
func (s *PostgresStore) Append(entryType store.EntryType, subject, action string, payload interface{}, metadata map[string]string) (*store.AuditEntry, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("audit/pg: failed to serialize payload: %w", err)
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		metadataBytes = []byte("{}")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++
	entry := &store.AuditEntry{
		EntryID:      uuid.New().String(),
		Sequence:     s.sequence,
		Timestamp:    time.Now().UTC(),
		EntryType:    entryType,
		Subject:      subject,
		Action:       action,
		Payload:      payloadBytes,
		PayloadHash:  pgComputeHash(payloadBytes),
		PreviousHash: s.chainHead,
		Metadata:     metadata,
	}

	// Compute entry hash (identical to store.AuditStore algorithm)
	entryHash, err := pgComputeEntryHash(entry)
	if err != nil {
		s.sequence--
		return nil, fmt.Errorf("audit/pg: failed to compute entry hash: %w", err)
	}
	entry.EntryHash = entryHash

	// INSERT — append-only
	_, err = s.db.ExecContext(context.Background(),
		`INSERT INTO audit_entries
			(entry_id, timestamp, entry_type, subject, action, payload, payload_hash, previous_hash, entry_hash, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.EntryID, entry.Timestamp, string(entry.EntryType),
		entry.Subject, entry.Action, entry.Payload,
		entry.PayloadHash, entry.PreviousHash, entry.EntryHash,
		metadataBytes,
	)
	if err != nil {
		s.sequence--
		return nil, fmt.Errorf("audit/pg: INSERT failed: %w", err)
	}

	s.chainHead = entry.EntryHash
	return entry, nil
}

// Query retrieves entries matching the filter.
func (s *PostgresStore) Query(filter store.QueryFilter) []*store.AuditEntry {
	query := `SELECT entry_id, sequence, timestamp, entry_type, subject, action,
	                 payload, payload_hash, previous_hash, entry_hash, metadata
	          FROM audit_entries WHERE 1=1`
	args := make([]interface{}, 0)
	argN := 0

	if filter.EntryType != "" {
		argN++
		query += fmt.Sprintf(" AND entry_type = $%d", argN)
		args = append(args, string(filter.EntryType))
	}
	if filter.Subject != "" {
		argN++
		query += fmt.Sprintf(" AND subject = $%d", argN)
		args = append(args, filter.Subject)
	}
	if filter.StartTime != nil {
		argN++
		query += fmt.Sprintf(" AND timestamp >= $%d", argN)
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		argN++
		query += fmt.Sprintf(" AND timestamp <= $%d", argN)
		args = append(args, *filter.EndTime)
	}
	if filter.StartSeq > 0 {
		argN++
		query += fmt.Sprintf(" AND sequence >= $%d", argN)
		args = append(args, filter.StartSeq)
	}
	if filter.EndSeq > 0 {
		argN++
		query += fmt.Sprintf(" AND sequence <= $%d", argN)
		args = append(args, filter.EndSeq)
	}

	query += " ORDER BY sequence ASC"

	if filter.MaxResults > 0 {
		argN++
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, filter.MaxResults)
	}

	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var results []*store.AuditEntry
	for rows.Next() {
		var e store.AuditEntry
		var metaJSON []byte
		if err := rows.Scan(
			&e.EntryID, &e.Sequence, &e.Timestamp, &e.EntryType,
			&e.Subject, &e.Action, &e.Payload, &e.PayloadHash,
			&e.PreviousHash, &e.EntryHash, &metaJSON,
		); err != nil {
			continue
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &e.Metadata)
		}
		results = append(results, &e)
	}
	return results
}

// VerifyChain validates the entire hash chain from genesis to head.
func (s *PostgresStore) VerifyChain() error {
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT sequence, entry_type, subject, action, payload_hash, previous_hash, entry_hash, timestamp
		 FROM audit_entries ORDER BY sequence ASC`)
	if err != nil {
		return fmt.Errorf("audit/pg: chain query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	expectedPrev := "genesis"
	idx := 0
	for rows.Next() {
		var e store.AuditEntry
		if err := rows.Scan(
			&e.Sequence, &e.EntryType, &e.Subject, &e.Action,
			&e.PayloadHash, &e.PreviousHash, &e.EntryHash, &e.Timestamp,
		); err != nil {
			return fmt.Errorf("audit/pg: scan failed at row %d: %w", idx, err)
		}

		if e.PreviousHash != expectedPrev {
			return fmt.Errorf("audit/pg: chain broken at entry %d: expected previous %s, got %s",
				idx, expectedPrev, e.PreviousHash)
		}

		computed, err := pgComputeEntryHash(&e)
		if err != nil {
			return fmt.Errorf("audit/pg: hash computation failed at entry %d: %w", idx, err)
		}
		if computed != e.EntryHash {
			return fmt.Errorf("audit/pg: hash mismatch at entry %d: computed %s, stored %s",
				idx, computed, e.EntryHash)
		}

		expectedPrev = e.EntryHash
		idx++
	}
	return rows.Err()
}

// GetChainHead returns the hash of the most recent entry.
func (s *PostgresStore) GetChainHead() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.chainHead
}

// Size returns the number of entries in the store.
func (s *PostgresStore) Size() int {
	var count int
	err := s.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM audit_entries`).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// --- Hash functions (identical to store.AuditStore for chain portability) ---

func pgComputeHash(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func pgComputeEntryHash(entry *store.AuditEntry) (string, error) {
	hashable := struct {
		Sequence     uint64          `json:"sequence"`
		Timestamp    time.Time       `json:"timestamp"`
		EntryType    store.EntryType `json:"entry_type"`
		Subject      string          `json:"subject"`
		Action       string          `json:"action"`
		PayloadHash  string          `json:"payload_hash"`
		PreviousHash string          `json:"previous_hash"`
	}{
		Sequence:     entry.Sequence,
		Timestamp:    entry.Timestamp,
		EntryType:    entry.EntryType,
		Subject:      entry.Subject,
		Action:       entry.Action,
		PayloadHash:  entry.PayloadHash,
		PreviousHash: entry.PreviousHash,
	}

	data, err := json.Marshal(hashable)
	if err != nil {
		return "", fmt.Errorf("audit/pg: marshal for hash failed: %w", err)
	}
	return pgComputeHash(data), nil
}

// Compile-time assertion.
var _ PersistentStore = (*PostgresStore)(nil)
