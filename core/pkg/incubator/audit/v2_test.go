package audit_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Mindburn-Labs/helm/core/pkg/incubator/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// Signal Log Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestSignalLog_AppendAndReplay(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "signals.jsonl")
	log := audit.NewSignalLog(logPath)

	// Append 3 signals
	require.NoError(t, log.Append(audit.SignalEntry{
		FindingID: "f-0", File: "go.mod", Category: "dependency",
		Outcome: audit.OutcomeFixed,
	}))
	require.NoError(t, log.Append(audit.SignalEntry{
		FindingID: "f-1", File: "main.go", Category: "security",
		Outcome: audit.OutcomeDismissed,
	}))
	require.NoError(t, log.Append(audit.SignalEntry{
		FindingID: "f-2", File: "bridge.go", Category: "architecture",
		Outcome: audit.OutcomeFixed,
	}))

	// Read back
	entries, err := log.ReadAll()
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "f-0", entries[0].FindingID)
	assert.Equal(t, audit.OutcomeFixed, entries[0].Outcome)
	assert.Equal(t, audit.OutcomeDismissed, entries[1].Outcome)
	assert.Equal(t, 3, log.Count())
}

func TestSignalLog_CrashSafe(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "signals.jsonl")

	// Write a valid entry + a corrupted partial line
	content := `{"finding_id":"f-0","outcome":"fixed"}
{"finding_id":"f-1","outcome":"dismi`
	require.NoError(t, os.WriteFile(logPath, []byte(content), 0o644))

	log := audit.NewSignalLog(logPath)
	entries, err := log.ReadAll()
	require.NoError(t, err)
	assert.Len(t, entries, 1) // Only the valid entry
	assert.Equal(t, "f-0", entries[0].FindingID)
}

func TestSignalLog_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "does_not_exist.jsonl")
	log := audit.NewSignalLog(logPath)

	entries, err := log.ReadAll()
	require.NoError(t, err)
	assert.Len(t, entries, 0)
	assert.Equal(t, 0, log.Count())
}

func TestSignalLog_Concurrent(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "signals.jsonl")
	log := audit.NewSignalLog(logPath)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = log.Append(audit.SignalEntry{
				FindingID: "f-concurrent",
				Outcome:   audit.OutcomeFixed,
			})
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 20, log.Count())
}

func TestSignalLog_ReplayInto(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "signals.jsonl")
	log := audit.NewSignalLog(logPath)

	// Write signals with categories
	for i := 0; i < 15; i++ {
		_ = log.Append(audit.SignalEntry{
			Category: "architecture",
			Outcome:  audit.OutcomeFixed,
		})
	}

	config, _ := audit.LoadPolicyConfig(filepath.Join(dir, "nope.json"))
	engine := audit.NewPolicyEngine(config)

	count, err := log.ReplayInto(engine)
	require.NoError(t, err)
	assert.Equal(t, 15, count)
}

// ═══════════════════════════════════════════════════════════════════════════
// AI Parser Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAIParser_ParseMissionOutput_JSON(t *testing.T) {
	missions := audit.NewMissionRegistry()
	parser := audit.NewAIParser(missions)

	// Strategy 1: JSON array
	rawJSON := `[
		{"file": "main.go", "severity": "high", "title": "SQL injection", "verdict": "FAIL"},
		{"file": "auth.go", "severity": "medium", "title": "Weak password hashing"}
	]`

	findings := parser.ParseMissionOutput("m-1", "security", rawJSON)
	assert.Len(t, findings, 2)
	assert.Equal(t, "main.go", findings[0].File)
	assert.Equal(t, audit.RemediationCategory("security"), findings[0].Category)
	assert.Equal(t, "FAIL", findings[0].Verdict)
}

func TestAIParser_ParseMissionOutput_FreeForm(t *testing.T) {
	missions := audit.NewMissionRegistry()
	parser := audit.NewAIParser(missions)

	rawText := `## Security Review

File: core/pkg/api/handler.go

1. SQL injection vulnerability in user input handling — High severity
2. Missing rate limiting on authentication endpoint — Medium severity

File: core/pkg/auth/token.go

3. Token validation does not check expiry — Critical issue
`
	findings := parser.ParseMissionOutput("m-2", "security", rawText)
	assert.Greater(t, len(findings), 0)
	// All should have security category
	for _, f := range findings {
		assert.Equal(t, audit.RemediationCategory("security"), f.Category)
	}
}

func TestAIParser_ParseMergedReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "report.json")

	report := map[string]interface{}{
		"version": "2.0",
		"git_sha": "abc123",
		"verdict": "NOT_COMPLIANT",
		"mechanical": map[string]interface{}{
			"sections": 3,
			"pass":     1, "fail": 1, "warn": 1,
			"details": []map[string]interface{}{
				{
					"section": "§1", "name": "go_mod_tidy", "verdict": "FAIL",
					"summary": "go.mod is dirty", "issues": []string{"go.mod has uncommitted changes"},
				},
				{
					"section": "§2", "name": "go_vet", "verdict": "PASS",
					"summary": "All packages pass",
				},
			},
		},
		"ai": map[string]interface{}{
			"model": "gemini-2.5", "missions": 2, "completed": 1,
			"findings":       []interface{}{},
			"coverage_score": 0.5,
			"mission_results": []map[string]interface{}{
				{
					"mission_id": "arch_coherence", "name": "Architecture Coherence",
					"category": "architecture", "status": "completed",
					"output":        `[{"file":"router.go","title":"Circular dependency","severity":"high","verdict":"FAIL"}]`,
					"finding_count": 1,
				},
			},
		},
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	require.NoError(t, os.WriteFile(reportPath, data, 0o644))

	missions := audit.NewMissionRegistry()
	parser := audit.NewAIParser(missions)

	findings, err := parser.ParseMergedReport(reportPath)
	require.NoError(t, err)
	assert.Greater(t, len(findings), 0)

	// Should include both mechanical and AI findings
	hasGoMod := false
	hasCircular := false
	for _, f := range findings {
		if f.Verdict == "FAIL" && containsStr(f.Title, "go.mod") {
			hasGoMod = true
		}
		if containsStr(f.Title, "Circular dependency") {
			hasCircular = true
		}
	}
	assert.True(t, hasGoMod, "should include go.mod finding from mechanical layer")
	assert.True(t, hasCircular, "should include circular dep from AI layer")
}

func TestPipeline_LoadAndProcess_LegacyFormat(t *testing.T) {
	dir := t.TempDir()
	config := audit.PipelineConfig{
		ProjectRoot:   dir,
		EvidenceDir:   filepath.Join(dir, "evidence"),
		LearningDir:   filepath.Join(dir, "learning"),
		DryRun:        true,
		AutoRemediate: true,
		AutoEvolve:    false,
	}
	os.MkdirAll(config.EvidenceDir, 0o755)
	os.MkdirAll(config.LearningDir, 0o755)

	// Write legacy format report
	report := map[string]interface{}{
		"git_sha": "deadbeef12345678",
		"verdict": "NOT_COMPLIANT",
		"findings": []map[string]interface{}{
			{"file": "go.mod", "category": "dependency", "verdict": "FAIL", "title": "go.mod dirty"},
			{"file": "clean.go", "category": "security", "verdict": "PASS", "title": "OK"},
		},
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	reportPath := filepath.Join(config.EvidenceDir, "test_report.json")
	require.NoError(t, os.WriteFile(reportPath, data, 0o644))

	pipeline := audit.NewPipeline(config)
	result, err := pipeline.LoadAndProcess(context.Background(), reportPath)
	require.NoError(t, err)
	assert.Equal(t, 2, result.FindingCount)
	assert.Equal(t, 1, result.FailCount)
	assert.Equal(t, 1, result.PassCount)
	assert.Contains(t, result.RunID, "deadbeef")
}

func TestPipeline_LoadAndProcess_MergedFormat(t *testing.T) {
	dir := t.TempDir()
	config := audit.PipelineConfig{
		ProjectRoot:   dir,
		EvidenceDir:   filepath.Join(dir, "evidence"),
		LearningDir:   filepath.Join(dir, "learning"),
		DryRun:        true,
		AutoRemediate: false,
		AutoEvolve:    false,
	}
	os.MkdirAll(config.EvidenceDir, 0o755)
	os.MkdirAll(config.LearningDir, 0o755)

	report := map[string]interface{}{
		"git_sha": "fedcba9876543210",
		"mechanical": map[string]interface{}{
			"sections": 1, "pass": 0, "fail": 1, "warn": 0,
			"details": []map[string]interface{}{
				{"section": "§1", "name": "go_mod_tidy", "verdict": "FAIL",
					"summary": "dirty", "issues": []string{"go.mod not clean"}},
			},
		},
		"ai": map[string]interface{}{
			"missions": 0, "completed": 0,
			"findings": []interface{}{}, "mission_results": []interface{}{},
		},
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	reportPath := filepath.Join(config.EvidenceDir, "merged.json")
	require.NoError(t, os.WriteFile(reportPath, data, 0o644))

	pipeline := audit.NewPipeline(config)
	result, err := pipeline.LoadAndProcess(context.Background(), reportPath)
	require.NoError(t, err)
	assert.Greater(t, result.FindingCount, 0)
	assert.Contains(t, result.RunID, "fedcba98")
}

// ═══════════════════════════════════════════════════════════════════════════
// Real Executor Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestExecuteTestStub_WritesFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pkg", "bridges"), 0o755)

	mutation := audit.Mutation{
		File:  "pkg/bridges/auth_bridge_test.go",
		Patch: "package bridges\n\nimport \"testing\"\n\nfunc TestBridge(t *testing.T) {}\n",
	}

	err := audit.ExecuteTestStub(dir, mutation)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "pkg", "bridges", "auth_bridge_test.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "TestBridge")
}

func TestExecuteTestStub_WontOverwrite(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "existing_test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("existing"), 0o644))

	mutation := audit.Mutation{
		File:  "existing_test.go",
		Patch: "new content",
	}

	err := audit.ExecuteTestStub(dir, mutation)
	assert.Error(t, err) // Should refuse to overwrite
	assert.Contains(t, err.Error(), "already exists")
}

func TestExecuteDockerfileFix_UserDirective(t *testing.T) {
	dir := t.TempDir()
	dockerfile := filepath.Join(dir, "Dockerfile")
	content := `FROM golang:1.24 AS builder
RUN go build -o app .

FROM alpine:3.19
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
`
	require.NoError(t, os.WriteFile(dockerfile, []byte(content), 0o644))

	mutation := audit.Mutation{
		File:        "Dockerfile",
		Description: "Dockerfile hardening: No USER directive",
	}

	err := audit.ExecuteDockerfileFix(dir, mutation)
	require.NoError(t, err)

	result, err := os.ReadFile(dockerfile)
	require.NoError(t, err)
	assert.Contains(t, string(result), "USER helm")
	assert.Contains(t, string(result), "adduser -S helm")
}

func TestPipeline_SignalLog_Integration(t *testing.T) {
	dir := t.TempDir()
	config := audit.PipelineConfig{
		ProjectRoot: dir,
		EvidenceDir: filepath.Join(dir, "evidence"),
		LearningDir: filepath.Join(dir, "learning"),
		DryRun:      true,
		AutoEvolve:  false,
	}
	os.MkdirAll(config.EvidenceDir, 0o755)
	os.MkdirAll(config.LearningDir, 0o755)

	pipeline := audit.NewPipeline(config)

	findings := []audit.Finding{
		{File: "a.go", Category: "security", Verdict: "FAIL", Title: "XSS"},
		{File: "b.go", Category: "architecture", Verdict: "FAIL", Title: "Bad design"},
	}

	// Record outcomes
	pipeline.RecordOutcome(0, findings, audit.OutcomeFixed)
	pipeline.RecordOutcome(1, findings, audit.OutcomeDismissed)

	// Verify signal log was written
	signalPath := filepath.Join(config.EvidenceDir, "signals.jsonl")
	assert.FileExists(t, signalPath)

	// Verify count
	assert.Equal(t, 2, pipeline.SignalCount())
}

// helper
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findStr(s, substr))
}

func findStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
