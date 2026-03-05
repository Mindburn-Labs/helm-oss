package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ── Append-Only Signal Log ──────────────────────────────────────────────────
//
// Durable JSONL log for human outcome signals. Each resolve appends one line.
// On evolve, all signals are replayed into the policy engine.
//
// Format: one JSON object per line (JSONL)
//   {"ts":"2026-02-20T18:00:00Z","finding_id":"f-3","file":"main.go","category":"security","outcome":"fixed"}
//
// Guarantees:
//   - Atomic append (O_APPEND + fsync)
//   - Thread-safe
//   - Crash-safe (partial last line is skipped on replay)

// SignalEntry represents a single human outcome signal.
type SignalEntry struct {
	Timestamp time.Time    `json:"ts"`
	FindingID string       `json:"finding_id"`
	File      string       `json:"file"`
	Title     string       `json:"title"`
	Category  string       `json:"category"`
	Severity  string       `json:"severity"`
	Outcome   HumanOutcome `json:"outcome"`
}

// SignalLog is an append-only JSONL file of outcome signals.
type SignalLog struct {
	path string
	mu   sync.Mutex
}

// NewSignalLog creates or opens a signal log at the given path.
func NewSignalLog(path string) *SignalLog {
	return &SignalLog{path: path}
}

// Append atomically appends a signal entry to the log.
func (l *SignalLog) Append(entry SignalEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("signal_log: marshal: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("signal_log: open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("signal_log: write: %w", err)
	}
	return f.Sync()
}

// ReadAll replays all signals from the log.
// Malformed lines (e.g. from crash) are silently skipped.
func (l *SignalLog) ReadAll() ([]SignalEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("signal_log: read: %w", err)
	}

	var entries []SignalEntry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var entry SignalEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Count returns the number of signals in the log.
func (l *SignalLog) Count() int {
	entries, _ := l.ReadAll()
	return len(entries)
}

// Path returns the log file path.
func (l *SignalLog) Path() string {
	return l.path
}

// ReplayInto feeds all logged signals into a policy engine.
func (l *SignalLog) ReplayInto(engine *PolicyEngine) (int, error) {
	entries, err := l.ReadAll()
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		finding := Finding{
			File:     e.File,
			Title:    e.Title,
			Category: RemediationCategory(e.Category),
			Severity: e.Severity,
		}
		engine.RecordOutcome(finding, e.Outcome)
	}
	return len(entries), nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
