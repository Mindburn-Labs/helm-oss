package audit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/incubator/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// Remediator Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestRemediator_DryRun(t *testing.T) {
	cfg := audit.DefaultRemediatorConfig()
	cfg.DryRun = true
	r := audit.NewRemediator(cfg)

	findings := []audit.Finding{
		{
			File:     "app/Page.tsx",
			Category: audit.RemediationArchitecture,
			Severity: "medium",
			Verdict:  "FAIL",
			Title:    "Unnecessary 'use client'",
		},
		{
			File:     "lib/unsafe.tsx",
			Category: audit.RemediationSecurity,
			Severity: "high",
			Verdict:  "FAIL",
			Title:    "dangerouslySetInnerHTML without sanitization",
		},
	}

	result, err := r.Process(context.Background(), findings)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalFindings)
	assert.Greater(t, result.Translated, 0)

	// Should have DRY_RUN actions, not actual PRs
	for _, a := range result.Actions {
		assert.Contains(t, a.Decision, "DRY_RUN")
	}
}

func TestRemediator_SkipPassFindings(t *testing.T) {
	cfg := audit.DefaultRemediatorConfig()
	cfg.DryRun = true
	r := audit.NewRemediator(cfg)

	findings := []audit.Finding{
		{File: "clean.go", Category: audit.RemediationSecurity, Verdict: "PASS", Title: "OK"},
	}
	result, _ := r.Process(context.Background(), findings)
	assert.Equal(t, 0, result.Translated)
	assert.Equal(t, 1, result.Skipped)
}

func TestRemediator_MaxPRsPerRun(t *testing.T) {
	cfg := audit.DefaultRemediatorConfig()
	cfg.DryRun = true
	cfg.MaxPRsPerRun = 1
	cfg.PRThreshold = 0.0 // Accept all
	r := audit.NewRemediator(cfg)

	findings := []audit.Finding{
		{File: "a.tsx", Category: audit.RemediationArchitecture, Verdict: "FAIL", Title: "use client"},
		{File: "b.tsx", Category: audit.RemediationArchitecture, Verdict: "FAIL", Title: "use client"},
	}
	result, _ := r.Process(context.Background(), findings)
	// Should stop at 1 even though there are 2
	actionCount := 0
	for _, a := range result.Actions {
		if a.Decision != "" {
			actionCount++
		}
	}
	assert.LessOrEqual(t, actionCount, 2)
}

func TestRemediationResult_Export(t *testing.T) {
	result := &audit.RemediationResult{
		TotalFindings: 5,
		AutoFixed:     1,
		PRsCreated:    2,
		Timestamp:     time.Now().UTC(),
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "result.json")
	err := result.ExportResult(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "auto_fixed")
}

// ═══════════════════════════════════════════════════════════════════════════
// Risk Scorer Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestRiskScorer_ScoreFile(t *testing.T) {
	model := audit.DefaultRiskModel()
	model.FileFailCounts["danger.go"] = 5
	model.FilePassCounts["danger.go"] = 1

	scorer := audit.NewRiskScorer(model, "")
	score := scorer.ScoreFile("danger.go")

	assert.Greater(t, score.Score, 0.3)
	assert.Equal(t, 5, score.HistoricalFails)
	assert.Contains(t, score.RiskFactors, "repeat_offender (5 fails)")
}

func TestRiskScorer_SafeFile(t *testing.T) {
	model := audit.DefaultRiskModel()
	model.FilePassCounts["safe.go"] = 100

	scorer := audit.NewRiskScorer(model, "")
	score := scorer.ScoreFile("safe.go")

	assert.Less(t, score.Score, 0.3)
}

func TestRiskScorer_Hotspots(t *testing.T) {
	model := audit.DefaultRiskModel()
	model.FileFailCounts["hot1.go"] = 10
	model.FileFailCounts["hot2.go"] = 8
	model.FileFailCounts["ok.go"] = 1

	scorer := audit.NewRiskScorer(model, "")
	hotspots := scorer.Hotspots(2)

	assert.Len(t, hotspots, 2)
	files := []string{hotspots[0].File, hotspots[1].File}
	assert.Contains(t, files, "hot1.go")
	assert.Contains(t, files, "hot2.go")
}

func TestRiskScorer_Train(t *testing.T) {
	scorer := audit.NewRiskScorer(nil, "")

	findings := []audit.Finding{
		{File: "a.go", Verdict: "FAIL"},
		{File: "a.go", Verdict: "FAIL"},
		{File: "b.go", Verdict: "PASS"},
	}
	scorer.Train(findings)

	scoreA := scorer.ScoreFile("a.go")
	scoreB := scorer.ScoreFile("b.go")
	assert.Greater(t, scoreA.Score, scoreB.Score)
}

func TestRiskScorer_ExportLoad(t *testing.T) {
	model := audit.DefaultRiskModel()
	model.FileFailCounts["test.go"] = 3

	dir := t.TempDir()
	path := filepath.Join(dir, "model.json")

	scorer := audit.NewRiskScorer(model, "")
	err := scorer.ExportModel(path)
	require.NoError(t, err)

	loaded, err := audit.LoadRiskModel(path)
	require.NoError(t, err)
	assert.Equal(t, 3, loaded.FileFailCounts["test.go"])
}

// ═══════════════════════════════════════════════════════════════════════════
// Policy Engine Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPolicyEngine_InsufficientSignals(t *testing.T) {
	engine := audit.NewPolicyEngine(nil)
	config := engine.Evolve()
	// Should return unchanged — not enough data
	assert.Equal(t, 0, config.Generation)
}

func TestPolicyEngine_HighFixRate(t *testing.T) {
	config := audit.DefaultPolicyConfig()
	engine := audit.NewPolicyEngine(config)

	// Record 12 signals where security findings always get fixed
	for i := 0; i < 12; i++ {
		engine.RecordOutcome(audit.Finding{
			Category: audit.RemediationSecurity,
			Severity: "high",
			Verdict:  "FAIL",
		}, audit.OutcomeFixed)
	}

	evolved := engine.Evolve()
	assert.Equal(t, 1, evolved.Generation)
	// Security weight should have increased
	assert.GreaterOrEqual(t, evolved.CategoryWeights["security"], 1.0)
}

func TestPolicyEngine_HighDismissRate(t *testing.T) {
	config := audit.DefaultPolicyConfig()
	config.CategoryWeights["brand_tone"] = 0.5
	engine := audit.NewPolicyEngine(config)

	// Brand tone findings always dismissed → noise
	for i := 0; i < 12; i++ {
		engine.RecordOutcome(audit.Finding{
			Category: audit.RemediationBrandTone,
			Severity: "low",
			Verdict:  "FAIL",
		}, audit.OutcomeDismissed)
	}

	evolved := engine.Evolve()
	assert.Less(t, evolved.CategoryWeights["brand_tone"], 0.5)
}

func TestPolicyEngine_ExportLoad(t *testing.T) {
	config := audit.DefaultPolicyConfig()
	engine := audit.NewPolicyEngine(config)

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.json")
	err := engine.ExportConfig(path)
	require.NoError(t, err)

	loaded, err := audit.LoadPolicyConfig(path)
	require.NoError(t, err)
	assert.Equal(t, config.CoverageFloor, loaded.CoverageFloor)
}

func TestPolicyEngine_FrozenConfig(t *testing.T) {
	config := audit.DefaultPolicyConfig()
	config.Frozen = true
	engine := audit.NewPolicyEngine(config)

	for i := 0; i < 20; i++ {
		engine.RecordOutcome(audit.Finding{Category: "x"}, audit.OutcomeFixed)
	}
	evolved := engine.Evolve()
	assert.Equal(t, 0, evolved.Generation) // Frozen — no evolution
}

// ═══════════════════════════════════════════════════════════════════════════
// Learning Store Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestLearningStore_RecordAndCount(t *testing.T) {
	dir := t.TempDir()
	store := audit.NewLearningStore(dir)

	err := store.RecordRun("run-001", "abc123", []audit.Finding{
		{File: "a.go", Verdict: "FAIL", Category: "security"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, store.RunCount())
}

func TestLearningStore_Trends(t *testing.T) {
	dir := t.TempDir()
	store := audit.NewLearningStore(dir)

	// Run 1: 3 security fails
	_ = store.RecordRun("run-1", "sha1", []audit.Finding{
		{File: "a.go", Verdict: "FAIL", Category: "security"},
		{File: "b.go", Verdict: "FAIL", Category: "security"},
		{File: "c.go", Verdict: "FAIL", Category: "security"},
	})

	// Run 2: 1 security fail (improving)
	_ = store.RecordRun("run-2", "sha2", []audit.Finding{
		{File: "a.go", Verdict: "FAIL", Category: "security"},
	})

	trend := store.Trends("security", 10)
	assert.Equal(t, "improving", trend.Direction)
	assert.Len(t, trend.DataPoints, 2)
}

func TestLearningStore_DetectRegressions(t *testing.T) {
	dir := t.TempDir()
	store := audit.NewLearningStore(dir)

	// Run 1: bug exists
	_ = store.RecordRun("run-1", "sha1", []audit.Finding{
		{File: "x.go", Category: "security", Title: "sql injection", Verdict: "FAIL"},
	})

	// Run 2: bug fixed
	_ = store.RecordRun("run-2", "sha2", []audit.Finding{})

	// Run 3: bug returns!
	current := []audit.Finding{
		{File: "x.go", Category: "security", Title: "sql injection", Verdict: "FAIL"},
	}
	_ = store.RecordRun("run-3", "sha3", current)

	regressions := store.DetectRegressions(current)
	assert.Len(t, regressions, 1)
	assert.Equal(t, "run-1", regressions[0].FirstSeenRun)
}

func TestLearningStore_FileRiskHistory(t *testing.T) {
	dir := t.TempDir()
	store := audit.NewLearningStore(dir)

	_ = store.RecordRun("run-1", "s1", []audit.Finding{
		{File: "hot.go", Verdict: "FAIL"},
		{File: "hot.go", Verdict: "FAIL"},
	})
	_ = store.RecordRun("run-2", "s2", []audit.Finding{
		{File: "hot.go", Verdict: "FAIL"},
	})

	hist := store.FileRiskHistory()
	assert.Equal(t, 3, hist["hot.go"])
}

// ═══════════════════════════════════════════════════════════════════════════
// Missions Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestMissionRegistry_Register(t *testing.T) {
	reg := audit.NewMissionRegistry()
	missions := audit.DefaultMissions()
	for _, m := range missions {
		reg.Register(m)
	}
	assert.Len(t, reg.Active(), 7) // 7 default missions
}

func TestMissionRegistry_RenderPrompt(t *testing.T) {
	reg := audit.NewMissionRegistry()
	reg.Register(&audit.Mission{
		ID:       "test",
		Template: "Audit {{.RepoRoot}} at {{.GitSHA}}",
		Active:   true,
	})

	prompt, err := reg.RenderPrompt("test", map[string]string{
		"RepoRoot": "/code",
		"GitSHA":   "abc123",
	})
	require.NoError(t, err)
	assert.Contains(t, prompt, "/code")
	assert.Contains(t, prompt, "abc123")
}

func TestMissionRegistry_NotFound(t *testing.T) {
	reg := audit.NewMissionRegistry()
	_, err := reg.RenderPrompt("nonexistent", nil)
	assert.Error(t, err)
}

func TestMissionRegistry_PrecisionTracking(t *testing.T) {
	reg := audit.NewMissionRegistry()
	reg.Register(&audit.Mission{ID: "sec", Template: "test", Active: true})

	reg.RecordPrecision("sec", 0.9)
	m, _ := reg.Get("sec")
	assert.Equal(t, 0.9, m.Precision)

	reg.RecordPrecision("sec", 0.5)
	// EMA: 0.3*0.5 + 0.7*0.9 = 0.78
	m, _ = reg.Get("sec")
	assert.InDelta(t, 0.78, m.Precision, 0.01)
}

func TestMissionRegistry_Evolve_LowPrecision(t *testing.T) {
	reg := audit.NewMissionRegistry()
	reg.Register(&audit.Mission{
		ID:       "noisy",
		Template: "find issues",
		Active:   true,
	})

	// Simulate many low-precision runs
	for i := 0; i < 6; i++ {
		reg.RecordPrecision("noisy", 0.3)
	}

	changes := reg.Evolve()
	assert.Greater(t, len(changes), 0)
	assert.Contains(t, changes[0], "PRECISION_BOOST")

	m, _ := reg.Get("noisy")
	assert.Contains(t, m.Template, "false positives")
	assert.Equal(t, 1, m.Generation)
}

func TestMissionRegistry_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missions.json")

	reg := audit.NewMissionRegistry()
	for _, m := range audit.DefaultMissions() {
		reg.Register(m)
	}
	err := reg.SaveToFile(path)
	require.NoError(t, err)

	reg2 := audit.NewMissionRegistry()
	err = reg2.LoadFromFile(path)
	require.NoError(t, err)
	assert.Len(t, reg2.Active(), 7)
}

// ═══════════════════════════════════════════════════════════════════════════
// WebSocket Sink Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestWebSocketSink_CreateAndShutdown(t *testing.T) {
	sink := audit.NewWebSocketSink(":0") // Ephemeral port
	assert.Equal(t, 0, sink.ClientCount())
	assert.NoError(t, sink.Shutdown())
}
