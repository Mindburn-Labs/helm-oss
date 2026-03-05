package audit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mindburn-Labs/helm/core/pkg/incubator/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// Remediation Strategy Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestRemediation_GoModTidy(t *testing.T) {
	translator := audit.NewFindingTranslator()
	audit.RegisterRemediationStrategies(translator)

	findings := []audit.Finding{
		{
			File:     "infra/bridge/go.mod",
			Category: audit.RemediationDependency,
			Severity: "high",
			Verdict:  "FAIL",
			Title:    "go.mod dirty — uncommitted dependency drift",
		},
	}

	mutations := translator.Translate(findings)
	require.Len(t, mutations, 1)
	assert.Contains(t, mutations[0].Description, "go mod tidy")
	assert.Equal(t, 0.95, mutations[0].Confidence)
	assert.True(t, mutations[0].AutoApply) // Safe to auto-apply
}

func TestRemediation_TestStub(t *testing.T) {
	translator := audit.NewFindingTranslator()
	audit.RegisterRemediationStrategies(translator)

	findings := []audit.Finding{
		{
			File:     "commercial/teams/merkle_bridge.go",
			Category: audit.RemediationArchitecture,
			Severity: "medium",
			Verdict:  "FAIL",
			Title:    "Bridge file has no corresponding test",
		},
	}

	mutations := translator.Translate(findings)
	require.Len(t, mutations, 1)
	assert.Contains(t, mutations[0].File, "_test.go")
	assert.Equal(t, 0.85, mutations[0].Confidence)
	assert.Contains(t, mutations[0].Patch, "func TestBridge_Placeholder")
}

func TestRemediation_UndefinedSymbol(t *testing.T) {
	translator := audit.NewFindingTranslator()
	audit.RegisterRemediationStrategies(translator)

	findings := []audit.Finding{
		{
			File:     "commercial/teams/approval_test.go",
			Category: audit.RemediationArchitecture,
			Severity: "medium",
			Verdict:  "FAIL",
			Title:    "undefined: InMemoryApprovalService",
		},
	}

	mutations := translator.Translate(findings)
	require.Len(t, mutations, 1)
	assert.Equal(t, 0.65, mutations[0].Confidence)
	assert.Contains(t, mutations[0].Description, "undefined symbol")
}

func TestRemediation_Dockerfile(t *testing.T) {
	translator := audit.NewFindingTranslator()
	audit.RegisterRemediationStrategies(translator)

	findings := []audit.Finding{
		{
			File:     "Dockerfile.commercial",
			Category: audit.RemediationSecurity,
			Severity: "high",
			Verdict:  "FAIL",
			Title:    "No USER directive — container runs as root",
		},
	}

	mutations := translator.Translate(findings)
	require.Len(t, mutations, 1)
	assert.Contains(t, mutations[0].Patch, "USER helm")
	assert.Equal(t, 0.80, mutations[0].Confidence)
}

func TestRemediation_SlogMigration(t *testing.T) {
	translator := audit.NewFindingTranslator()
	audit.RegisterRemediationStrategies(translator)

	findings := []audit.Finding{
		{
			File:     "core/cmd/bootstrap/main.go",
			Category: audit.RemediationReliability,
			Severity: "low",
			Verdict:  "FAIL",
			Title:    "Raw fmt.Print/log.Print calls — migrate to slog",
		},
	}

	mutations := translator.Translate(findings)
	require.Len(t, mutations, 1)
	assert.Contains(t, mutations[0].Patch, "slog.Info")
	assert.Equal(t, 0.50, mutations[0].Confidence)
	assert.False(t, mutations[0].AutoApply)
}

func TestRemediation_PassFindingsSkipped(t *testing.T) {
	translator := audit.NewFindingTranslator()
	audit.RegisterRemediationStrategies(translator)

	findings := []audit.Finding{
		{
			File:     "protocols/json-schemas/",
			Category: audit.RemediationArchitecture,
			Severity: "low",
			Verdict:  "PASS",
			Title:    "All 144 schemas valid JSON",
		},
	}

	mutations := translator.Translate(findings)
	assert.Len(t, mutations, 0) // PASS findings don't get remediated
}

// ═══════════════════════════════════════════════════════════════════════════
// Pipeline Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPipeline_ProcessFindings(t *testing.T) {
	dir := t.TempDir()
	config := audit.PipelineConfig{
		ProjectRoot:   dir,
		EvidenceDir:   filepath.Join(dir, "evidence"),
		LearningDir:   filepath.Join(dir, "learning"),
		DryRun:        true,
		AutoRemediate: true,
		AutoEvolve:    true,
	}
	os.MkdirAll(config.EvidenceDir, 0o755)
	os.MkdirAll(config.LearningDir, 0o755)

	pipeline := audit.NewPipeline(config)

	findings := []audit.Finding{
		{File: "go.mod", Category: audit.RemediationDependency, Severity: "high", Verdict: "FAIL", Title: "go.mod dirty"},
		{File: "bridge.go", Category: audit.RemediationArchitecture, Severity: "medium", Verdict: "FAIL", Title: "Bridge file has no corresponding test"},
		{File: "clean.go", Category: audit.RemediationSecurity, Severity: "low", Verdict: "PASS", Title: "No issues"},
	}

	result, err := pipeline.ProcessFindings(context.Background(), findings, "abc12345deadbeef")
	require.NoError(t, err)

	assert.Equal(t, 3, result.FindingCount)
	assert.Equal(t, 1, result.PassCount)
	assert.Equal(t, 2, result.FailCount)
	assert.Equal(t, 1, result.TotalRuns)
	assert.NotEmpty(t, result.ReportHash)
	assert.Contains(t, result.RunID, "abc12345")
}

func TestPipeline_RecordOutcome(t *testing.T) {
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
		{File: "a.go", Category: audit.RemediationSecurity, Verdict: "FAIL", Title: "XSS"},
	}

	// Should not panic
	pipeline.RecordOutcome(0, findings, audit.OutcomeFixed)
	pipeline.RecordOutcome(-1, findings, audit.OutcomeFixed) // out of range — no-op
	pipeline.RecordOutcome(99, findings, audit.OutcomeFixed) // out of range — no-op
}

func TestPipeline_MultiRun_RegressionDetection(t *testing.T) {
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

	pipeline := audit.NewPipeline(config)

	// Run 1: Bug exists
	bug := audit.Finding{File: "vuln.go", Category: "security", Title: "SQL injection", Verdict: "FAIL"}
	r1, _ := pipeline.ProcessFindings(context.Background(), []audit.Finding{bug}, "sha1sha1sha1sha1")
	assert.Equal(t, 1, r1.TotalRuns)

	// Run 2: Bug fixed
	r2, _ := pipeline.ProcessFindings(context.Background(), []audit.Finding{}, "sha2sha2sha2sha2")
	assert.Equal(t, 2, r2.TotalRuns)
	assert.Len(t, r2.Regressions, 0) // No regressions — it's fixed

	// Run 3: Bug returns!
	r3, _ := pipeline.ProcessFindings(context.Background(), []audit.Finding{bug}, "sha3sha3sha3sha3")
	assert.Equal(t, 3, r3.TotalRuns)
	assert.Len(t, r3.Regressions, 1) // REGRESSION!
	assert.Equal(t, "vuln.go", r3.Regressions[0].Finding.File)
}

func TestPipeline_RiskTraining(t *testing.T) {
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

	pipeline := audit.NewPipeline(config)

	// Run with many failures for one file
	findings := []audit.Finding{
		{File: "hot.go", Verdict: "FAIL", Title: "Problem 1"},
		{File: "hot.go", Verdict: "FAIL", Title: "Problem 2"},
		{File: "hot.go", Verdict: "FAIL", Title: "Problem 3"},
		{File: "safe.go", Verdict: "PASS", Title: "OK"},
	}

	result, _ := pipeline.ProcessFindings(context.Background(), findings, "sha4sha4sha4sha4")

	// Risk model should have trained — hot.go should appear in top risks
	assert.Greater(t, len(result.TopRisks), 0)

	// Verify the model was persisted
	modelPath := filepath.Join(config.EvidenceDir, "risk_model.json")
	assert.FileExists(t, modelPath)
}

func TestPipeline_PersistsResult(t *testing.T) {
	dir := t.TempDir()
	config := audit.PipelineConfig{
		ProjectRoot: dir,
		EvidenceDir: filepath.Join(dir, "evidence"),
		LearningDir: filepath.Join(dir, "learning"),
		DryRun:      true,
	}
	os.MkdirAll(config.EvidenceDir, 0o755)
	os.MkdirAll(config.LearningDir, 0o755)

	pipeline := audit.NewPipeline(config)
	_, err := pipeline.ProcessFindings(context.Background(), nil, "sha5sha5sha5sha5")
	require.NoError(t, err)

	resultPath := filepath.Join(config.EvidenceDir, "pipeline_result.json")
	assert.FileExists(t, resultPath)
}
