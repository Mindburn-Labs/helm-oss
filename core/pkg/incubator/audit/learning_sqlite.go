package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ── SQLite Learning Store ───────────────────────────────────────────────────
//
// High-performance audit store backed by SQLite (modernc.org/sqlite, pure Go).
//
// Schema:
//   - audit_runs:  run metadata + JSON summary
//   - findings:    individual findings with fingerprint index
//   - signals:     human outcome signals (append-only)
//
// Replaces JSON-per-run file storage for O(1) queries at 1000+ runs.

// SQLiteLearningStore implements AuditStore with SQLite backing.
type SQLiteLearningStore struct {
	mu sync.RWMutex
	db *sql.DB
}

// Compile-time check.
var _ AuditStore = (*SQLiteLearningStore)(nil)

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS audit_runs (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id    TEXT UNIQUE NOT NULL,
    git_sha   TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    summary   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS findings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      TEXT NOT NULL REFERENCES audit_runs(run_id),
    file        TEXT NOT NULL,
    category    TEXT NOT NULL,
    severity    TEXT NOT NULL,
    verdict     TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT DEFAULT '',
    fingerprint TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS signals (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finding_id TEXT NOT NULL,
    file       TEXT DEFAULT '',
    category   TEXT DEFAULT '',
    outcome    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_findings_fingerprint ON findings(fingerprint);
CREATE INDEX IF NOT EXISTS idx_findings_file ON findings(file);
CREATE INDEX IF NOT EXISTS idx_findings_run ON findings(run_id);
CREATE INDEX IF NOT EXISTS idx_findings_verdict ON findings(verdict);
CREATE INDEX IF NOT EXISTS idx_signals_category ON signals(category);
`

// NewSQLiteLearningStore opens or creates a SQLite-backed store.
func NewSQLiteLearningStore(dbPath string) (*SQLiteLearningStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite store: mkdir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: open: %w", err)
	}

	// Performance pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite store: pragma: %w", err)
		}
	}

	// Apply schema
	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite store: schema: %w", err)
	}

	return &SQLiteLearningStore{db: db}, nil
}

// Close closes the database connection.
func (s *SQLiteLearningStore) Close() error {
	return s.db.Close()
}

// RecordRun stores a run and all its findings.
func (s *SQLiteLearningStore) RecordRun(runID, gitSHA string, findings []Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("sqlite: begin: %w", err)
	}
	defer tx.Rollback()

	summary := summarizeRun(findings)
	summaryJSON, _ := json.Marshal(summary)

	_, err = tx.Exec(
		`INSERT OR IGNORE INTO audit_runs (run_id, git_sha, timestamp, summary)
		 VALUES (?, ?, ?, ?)`,
		runID, gitSHA, time.Now().UTC(), string(summaryJSON),
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert run: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO findings (run_id, file, category, severity, verdict, title, description, fingerprint)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("sqlite: prepare finding: %w", err)
	}
	defer stmt.Close()

	for _, f := range findings {
		fp := fingerprintFinding(f)
		_, err := stmt.Exec(runID, f.File, string(f.Category), f.Severity, f.Verdict, f.Title, f.Description, fp)
		if err != nil {
			slog.Warn("sqlite: insert finding", "error", err, "file", f.File)
		}
	}

	return tx.Commit()
}

// Trends returns the trend for a category over the last N runs.
func (s *SQLiteLearningStore) Trends(category string, lastN int) *Trend {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trend := &Trend{Category: category}

	rows, err := s.db.Query(`
		SELECT r.run_id, r.timestamp, COUNT(f.id) as fail_count
		FROM audit_runs r
		LEFT JOIN findings f ON f.run_id = r.run_id 
			AND f.category = ? AND f.verdict = 'FAIL'
		GROUP BY r.run_id
		ORDER BY r.timestamp DESC
		LIMIT ?
	`, category, lastN)
	if err != nil {
		slog.Warn("sqlite: trends query", "error", err)
		return trend
	}
	defer rows.Close()

	var points []TrendPoint
	for rows.Next() {
		var tp TrendPoint
		var ts time.Time
		if err := rows.Scan(&tp.RunID, &ts, &tp.Value); err != nil {
			continue
		}
		tp.Timestamp = ts
		points = append(points, tp)
	}

	// Reverse to chronological order
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	trend.DataPoints = points

	if len(points) >= 2 {
		first := points[0].Value
		last := points[len(points)-1].Value
		if last < first {
			trend.Direction = "improving"
		} else if last > first {
			trend.Direction = "degrading"
		} else {
			trend.Direction = "stable"
		}
		trend.Slope = float64(last-first) / float64(len(points))
	}

	return trend
}

// DetectRegressions finds findings that were fixed and came back.
func (s *SQLiteLearningStore) DetectRegressions(currentFindings []Finding) []Regression {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.RunCount_unlocked() < 2 {
		return nil
	}

	// Get all historical fingerprints with their run presence
	type fpHistory struct {
		firstRun string
		lastRun  string
		runs     []string
	}

	histories := make(map[string]*fpHistory)

	rows, err := s.db.Query(`
		SELECT f.fingerprint, f.run_id, r.timestamp
		FROM findings f
		JOIN audit_runs r ON r.run_id = f.run_id
		WHERE f.verdict = 'FAIL'
		ORDER BY r.timestamp ASC
	`)
	if err != nil {
		slog.Warn("sqlite: regressions query", "error", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var fp, runID string
		var ts time.Time
		if err := rows.Scan(&fp, &runID, &ts); err != nil {
			continue
		}
		if _, ok := histories[fp]; !ok {
			histories[fp] = &fpHistory{firstRun: runID}
		}
		histories[fp].lastRun = runID
		histories[fp].runs = append(histories[fp].runs, runID)
	}

	// Get all run IDs in order
	allRuns, _ := s.allRunIDs()

	var regressions []Regression
	for _, f := range currentFindings {
		if f.Verdict != "FAIL" {
			continue
		}
		fp := fingerprintFinding(f)
		hist, ok := histories[fp]
		if !ok || len(hist.runs) < 2 {
			continue
		}

		// Check if there's a gap (absent in some runs)
		runSet := make(map[string]bool)
		for _, r := range hist.runs {
			runSet[r] = true
		}

		var absentRun string
		for _, r := range allRuns {
			if !runSet[r] && r != hist.firstRun {
				absentRun = r
				break
			}
		}

		if absentRun != "" {
			regressions = append(regressions, Regression{
				Finding:         f,
				FirstSeenRun:    hist.firstRun,
				FixedInRun:      absentRun,
				RegressedRun:    hist.lastRun,
				OccurrenceCount: len(hist.runs),
			})
		}
	}

	return regressions
}

// FileRiskHistory returns FAIL count by file across all runs.
func (s *SQLiteLearningStore) FileRiskHistory() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)

	rows, err := s.db.Query(`
		SELECT file, COUNT(*) as cnt
		FROM findings
		WHERE verdict = 'FAIL' AND file != ''
		GROUP BY file
		ORDER BY cnt DESC
	`)
	if err != nil {
		return counts
	}
	defer rows.Close()

	for rows.Next() {
		var file string
		var cnt int
		if err := rows.Scan(&file, &cnt); err != nil {
			continue
		}
		counts[file] = cnt
	}

	return counts
}

// RunCount returns the number of recorded runs.
func (s *SQLiteLearningStore) RunCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RunCount_unlocked()
}

func (s *SQLiteLearningStore) RunCount_unlocked() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM audit_runs`).Scan(&count)
	return count
}

func (s *SQLiteLearningStore) allRunIDs() ([]string, error) {
	rows, err := s.db.Query(`SELECT run_id FROM audit_runs ORDER BY timestamp ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// AppendSignal stores a human outcome signal.
func (s *SQLiteLearningStore) AppendSignal(entry SignalEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	_, err := s.db.Exec(
		`INSERT INTO signals (timestamp, finding_id, file, category, outcome)
		 VALUES (?, ?, ?, ?, ?)`,
		ts, entry.FindingID, entry.File, entry.Category, string(entry.Outcome),
	)
	return err
}

// SignalCount returns the number of signals.
func (s *SQLiteLearningStore) SignalCount() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM signals`).Scan(&count)
	return count
}

// ReplaySignals feeds all signals into a policy engine.
func (s *SQLiteLearningStore) ReplaySignals(engine *PolicyEngine) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT file, category, outcome FROM signals ORDER BY timestamp ASC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var file, category, outcome string
		if err := rows.Scan(&file, &category, &outcome); err != nil {
			continue
		}
		finding := Finding{
			File:     file,
			Category: RemediationCategory(category),
		}
		engine.RecordOutcome(finding, HumanOutcome(outcome))
		count++
	}

	return count, nil
}

// MigrateFromJSON imports existing JSON run files into SQLite.
func (s *SQLiteLearningStore) MigrateFromJSON(jsonDir string) (int, error) {
	entries, err := os.ReadDir(jsonDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	imported := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(jsonDir, e.Name()))
		if err != nil {
			continue
		}
		var run AuditRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}

		// Skip if already imported
		var exists int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM audit_runs WHERE run_id = ?`, run.RunID).Scan(&exists)
		if exists > 0 {
			continue
		}

		if err := s.RecordRun(run.RunID, run.GitSHA, run.Findings); err != nil {
			slog.Warn("sqlite: migration skip", "run", run.RunID, "error", err)
			continue
		}
		imported++
	}

	if imported > 0 {
		slog.Info("sqlite: migrated JSON runs", "count", imported)
	}

	// Sort runs by timestamp for consistent ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return imported, nil
}
