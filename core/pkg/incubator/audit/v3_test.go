package audit_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/incubator/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// SQLite Learning Store Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestSQLiteStore_RecordAndQuery(t *testing.T) {
	dir := t.TempDir()
	store, err := audit.NewSQLiteLearningStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	findings := []audit.Finding{
		{File: "main.go", Category: "security", Severity: "high", Verdict: "FAIL", Title: "XSS vuln"},
		{File: "auth.go", Category: "security", Severity: "medium", Verdict: "FAIL", Title: "Weak hash"},
		{File: "ok.go", Category: "architecture", Severity: "low", Verdict: "PASS", Title: "Clean"},
	}

	err = store.RecordRun("run-001", "abc123", findings)
	require.NoError(t, err)

	assert.Equal(t, 1, store.RunCount())

	// File risk history
	history := store.FileRiskHistory()
	assert.Equal(t, 1, history["main.go"])
	assert.Equal(t, 1, history["auth.go"])
	assert.Equal(t, 0, history["ok.go"]) // PASS doesn't count
}

func TestSQLiteStore_Trends(t *testing.T) {
	dir := t.TempDir()
	store, err := audit.NewSQLiteLearningStore(filepath.Join(dir, "trend.db"))
	require.NoError(t, err)
	defer store.Close()

	// Run 1: 3 security failures
	run1 := []audit.Finding{
		{File: "a.go", Category: "security", Verdict: "FAIL", Title: "Issue 1"},
		{File: "b.go", Category: "security", Verdict: "FAIL", Title: "Issue 2"},
		{File: "c.go", Category: "security", Verdict: "FAIL", Title: "Issue 3"},
	}
	require.NoError(t, store.RecordRun("run-001", "sha1", run1))

	// Run 2: 1 security failure (improving)
	run2 := []audit.Finding{
		{File: "a.go", Category: "security", Verdict: "FAIL", Title: "Issue 1"},
		{File: "d.go", Category: "architecture", Verdict: "FAIL", Title: "Arch issue"},
	}
	require.NoError(t, store.RecordRun("run-002", "sha2", run2))

	trend := store.Trends("security", 10)
	assert.Equal(t, "security", trend.Category)
	assert.Len(t, trend.DataPoints, 2)
	assert.Equal(t, "improving", trend.Direction)
}

func TestSQLiteStore_Regressions(t *testing.T) {
	dir := t.TempDir()
	store, err := audit.NewSQLiteLearningStore(filepath.Join(dir, "reg.db"))
	require.NoError(t, err)
	defer store.Close()

	finding := audit.Finding{File: "a.go", Category: "security", Verdict: "FAIL", Title: "XSS"}

	// Run 1: present
	require.NoError(t, store.RecordRun("run-001", "sha1", []audit.Finding{finding}))
	// Run 2: absent (fixed)
	require.NoError(t, store.RecordRun("run-002", "sha2", []audit.Finding{}))
	// Run 3: present again (regression!)
	require.NoError(t, store.RecordRun("run-003", "sha3", []audit.Finding{finding}))

	regressions := store.DetectRegressions([]audit.Finding{finding})
	assert.Len(t, regressions, 1)
	assert.Equal(t, "run-001", regressions[0].FirstSeenRun)
	assert.Equal(t, "run-002", regressions[0].FixedInRun)
}

func TestSQLiteStore_Signals(t *testing.T) {
	dir := t.TempDir()
	store, err := audit.NewSQLiteLearningStore(filepath.Join(dir, "sig.db"))
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.AppendSignal(audit.SignalEntry{
		FindingID: "f-0", Category: "security", Outcome: audit.OutcomeFixed,
	}))
	require.NoError(t, store.AppendSignal(audit.SignalEntry{
		FindingID: "f-1", Category: "architecture", Outcome: audit.OutcomeDismissed,
	}))

	assert.Equal(t, 2, store.SignalCount())
}

func TestSQLiteStore_MigrateFromJSON(t *testing.T) {
	dir := t.TempDir()
	jsonDir := filepath.Join(dir, "json_runs")

	// Create a JSON learning store with some runs
	jsonStore := audit.NewLearningStore(jsonDir)
	require.NoError(t, jsonStore.RecordRun("legacy-001", "sha1", []audit.Finding{
		{File: "x.go", Category: "security", Verdict: "FAIL", Title: "Legacy finding"},
	}))
	// Small delay to avoid timestamp collision in JSON filenames (second resolution)
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, jsonStore.RecordRun("legacy-002", "sha2", []audit.Finding{
		{File: "y.go", Category: "architecture", Verdict: "PASS", Title: "OK"},
	}))

	// Migrate to SQLite
	sqlStore, err := audit.NewSQLiteLearningStore(filepath.Join(dir, "migrated.db"))
	require.NoError(t, err)
	defer sqlStore.Close()

	imported, err := sqlStore.MigrateFromJSON(jsonDir)
	require.NoError(t, err)
	assert.Equal(t, 2, imported)
	assert.Equal(t, 2, sqlStore.RunCount())
}

func TestSQLiteStore_Interface(t *testing.T) {
	dir := t.TempDir()
	store, err := audit.NewSQLiteLearningStore(filepath.Join(dir, "iface.db"))
	require.NoError(t, err)
	defer store.Close()

	// Both satisfy AuditStore
	var _ audit.AuditStore = store
	var _ audit.AuditStore = audit.NewLearningStore(dir)
}

// ═══════════════════════════════════════════════════════════════════════════
// Gemini Runner Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestGeminiRunner_NoCredentials(t *testing.T) {
	parser := audit.NewAIParser(audit.NewMissionRegistry())
	runner := audit.NewGeminiRunner(audit.GeminiConfig{}, parser)

	mission := &audit.Mission{
		ID:       "test",
		Name:     "Test",
		Category: "security",
		Template: "test prompt",
		Active:   true,
	}

	t.Setenv("HELM_DISABLE_GEMINI_CLI", "1")
	_, err := runner.RunMission(context.Background(), mission, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no API key")
}
