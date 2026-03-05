package audit

// ── AuditStore Interface ────────────────────────────────────────────────────
//
// Abstracts the learning store backend so JSON and SQLite are interchangeable.

// AuditStore is the interface for audit history persistence.
type AuditStore interface {
	// RecordRun persists a completed audit run.
	RecordRun(runID, gitSHA string, findings []Finding) error

	// Trends returns the trend for a specific category over the last N runs.
	Trends(category string, lastN int) *Trend

	// DetectRegressions finds findings that were fixed and returned.
	DetectRegressions(currentFindings []Finding) []Regression

	// FileRiskHistory returns the FAIL count by file across all runs.
	FileRiskHistory() map[string]int

	// RunCount returns the number of recorded runs.
	RunCount() int
}

// Compile-time check: LearningStore satisfies AuditStore.
var _ AuditStore = (*LearningStore)(nil)
