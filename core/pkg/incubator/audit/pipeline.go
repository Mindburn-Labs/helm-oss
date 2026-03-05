package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// ── Autonomous Pipeline ─────────────────────────────────────────────────────
//
// Wires together all autonomy modules into a single closed-loop pipeline:
//
//   Run → Translate → Heal → Record → Train → Evolve → Report
//
// Usage:
//
//   pipeline := audit.NewPipeline(config)
//   result := pipeline.Execute(ctx)
//   // result contains: findings, remediations, risk scores, policy changes

// PipelineConfig configures the autonomous pipeline.
type PipelineConfig struct {
	ProjectRoot     string `json:"project_root"`
	EvidenceDir     string `json:"evidence_dir"`
	LearningDir     string `json:"learning_dir"`
	DryRun          bool   `json:"dry_run"`
	AutoEvolve      bool   `json:"auto_evolve"`    // Auto-run policy evolution
	AutoRemediate   bool   `json:"auto_remediate"` // Auto-run heal
	SignAttestation bool   `json:"sign_attestation"`
	AIEnabled       bool   `json:"ai_enabled"` // Enable L2 AI missions
	GeminiAPIKey    string `json:"-"`          // API key (alt: OAuth auto-loaded)
}

// DefaultPipelineConfig returns production-safe defaults.
func DefaultPipelineConfig(projectRoot string) PipelineConfig {
	return PipelineConfig{
		ProjectRoot:     projectRoot,
		EvidenceDir:     filepath.Join(projectRoot, "data", "evidence"),
		LearningDir:     filepath.Join(projectRoot, "data", "evidence", "learning"),
		DryRun:          true,
		AutoEvolve:      true,
		AutoRemediate:   true,
		SignAttestation: false,
	}
}

// PipelineResult contains everything the pipeline produced.
type PipelineResult struct {
	// Audit results
	RunID        string    `json:"run_id"`
	GitSHA       string    `json:"git_sha"`
	Timestamp    time.Time `json:"timestamp"`
	FindingCount int       `json:"finding_count"`
	PassCount    int       `json:"pass_count"`
	FailCount    int       `json:"fail_count"`
	WarnCount    int       `json:"warn_count"`

	// Remediation
	Remediation *RemediationResult `json:"remediation,omitempty"`

	// Risk
	TopRisks []RiskScore `json:"top_risks,omitempty"`

	// Learning
	Regressions []Regression `json:"regressions,omitempty"`
	TotalRuns   int          `json:"total_runs"`

	// Policy
	PolicyGeneration int `json:"policy_generation"`

	// Meta
	ReportHash string `json:"report_hash"`
}

// Pipeline orchestrates the full autonomy loop.
type Pipeline struct {
	config       PipelineConfig
	translator   *FindingTranslator
	remediator   *Remediator
	scorer       *RiskScorer
	policy       *PolicyEngine
	store        AuditStore
	missions     *MissionRegistry
	signalLog    *SignalLog
	aiParser     *AIParser
	geminiRunner *GeminiRunner        // nil if AI disabled
	sqliteStore  *SQLiteLearningStore // nil if using JSON fallback
}

// NewPipeline creates a fully-wired autonomous pipeline.
func NewPipeline(config PipelineConfig) *Pipeline {
	// Initialize translator with remediation strategies
	translator := NewFindingTranslator()
	RegisterRemediationStrategies(translator)

	// Load or create risk model
	modelPath := filepath.Join(config.EvidenceDir, "risk_model.json")
	model, _ := LoadRiskModel(modelPath)
	scorer := NewRiskScorer(model, config.ProjectRoot)

	// Load or create policy
	policyPath := filepath.Join(config.EvidenceDir, "policy.json")
	policyConfig, _ := LoadPolicyConfig(policyPath)
	policy := NewPolicyEngine(policyConfig)

	// Remediator — inject enhanced translator
	remCfg := DefaultRemediatorConfig()
	remCfg.DryRun = config.DryRun
	remCfg.RepoRoot = config.ProjectRoot
	remediator := NewRemediator(remCfg)
	remediator.WithTranslator(translator) // Wire remediation strategies

	// Learning store — auto-select SQLite when available
	var store AuditStore
	var sqliteStore *SQLiteLearningStore
	dbPath := filepath.Join(config.EvidenceDir, "learning.db")
	if sq, err := NewSQLiteLearningStore(dbPath); err == nil {
		store = sq
		sqliteStore = sq
		// Migrate existing JSON runs
		if n, _ := sq.MigrateFromJSON(config.LearningDir); n > 0 {
			slog.Info("pipeline: migrated JSON runs to SQLite", "count", n)
		}
		slog.Info("pipeline: using SQLite learning store", "path", dbPath)
	} else {
		store = NewLearningStore(config.LearningDir)
		slog.Info("pipeline: using JSON learning store", "path", config.LearningDir)
	}

	// Missions
	missions := NewMissionRegistry()
	missionsPath := filepath.Join(config.EvidenceDir, "missions.json")
	if err := missions.LoadFromFile(missionsPath); err != nil {
		for _, m := range DefaultMissions() {
			missions.Register(m)
		}
	}

	// Signal log (append-only JSONL)
	signalLog := NewSignalLog(filepath.Join(config.EvidenceDir, "signals.jsonl"))

	// AI parser
	aiParser := NewAIParser(missions)

	// Gemini runner (optional)
	var geminiRunner *GeminiRunner
	if config.AIEnabled {
		gcfg := DefaultGeminiConfig(config.GeminiAPIKey)
		// If no API key, check for gemini CLI (preferred — uses Google Ultra sub),
		// then fall back to gcloud ADC OAuth → Vertex AI.
		if gcfg.APIKey == "" {
			if HasGeminiCLI() {
				slog.Info("pipeline: using gemini CLI subprocess (Google Ultra subscription)")
			} else if token, err := LoadOAuthToken(); err == nil {
				gcfg.OAuthToken = token
				// Auto-detect GCP project for Vertex AI endpoint
				if project := detectGCPProject(); project != "" {
					gcfg.Project = project
					slog.Info("pipeline: using Vertex AI",
						"project", project,
						"region", gcfg.Region,
					)
				} else {
					slog.Warn("pipeline: OAuth token found but no GCP project — Vertex AI unavailable")
				}
			} else {
				slog.Warn("pipeline: no Gemini credentials found", "error", err)
			}
		}
		if gcfg.APIKey != "" || gcfg.OAuthToken != "" || HasGeminiCLI() {
			geminiRunner = NewGeminiRunner(gcfg, aiParser)
		}
	}

	return &Pipeline{
		config:       config,
		translator:   translator,
		remediator:   remediator,
		scorer:       scorer,
		policy:       policy,
		store:        store,
		missions:     missions,
		signalLog:    signalLog,
		aiParser:     aiParser,
		geminiRunner: geminiRunner,
		sqliteStore:  sqliteStore,
	}
}

// ProcessFindings takes structured findings and runs the full autonomy loop.
func (p *Pipeline) ProcessFindings(ctx context.Context, findings []Finding, gitSHA string) (*PipelineResult, error) {
	runID := fmt.Sprintf("run-%s-%d", gitSHA[:8], time.Now().Unix())
	result := &PipelineResult{
		RunID:     runID,
		GitSHA:    gitSHA,
		Timestamp: time.Now().UTC(),
	}

	// Count mechanical findings
	for _, f := range findings {
		switch f.Verdict {
		case "PASS":
			result.PassCount++
		case "FAIL":
			result.FailCount++
		case "WARN":
			result.WarnCount++
		}
	}

	// Step 0: Run L2 AI missions (if enabled)
	if p.geminiRunner != nil {
		vars := map[string]string{
			"RepoRoot": p.config.ProjectRoot,
			"GitSHA":   gitSHA,
		}
		slog.Info("pipeline: running L2 AI missions",
			"model", p.geminiRunner.config.Model,
			"missions", len(p.missions.Active()),
		)
		aiResults, err := p.geminiRunner.RunAllMissions(ctx, p.missions, vars)
		if err != nil {
			slog.Warn("pipeline: L2 missions failed", "error", err)
		}
		// Parse and merge AI findings
		for _, r := range aiResults {
			if r.Status == "completed" {
				aiFindings := p.aiParser.ParseMissionOutput(r.MissionID, r.Category, r.Output)
				findings = append(findings, aiFindings...)
				for _, f := range aiFindings {
					if f.Verdict == "FAIL" {
						result.FailCount++
					} else if f.Verdict == "WARN" {
						result.WarnCount++
					}
				}
				slog.Info("pipeline: L2 mission complete",
					"mission", r.MissionID,
					"findings", len(aiFindings),
				)
			}
		}
	}

	result.FindingCount = len(findings)

	slog.Info("pipeline: processing findings",
		"run_id", runID,
		"total", result.FindingCount,
		"fail", result.FailCount,
	)

	// Step 1: Record to learning store
	if err := p.store.RecordRun(runID, gitSHA, findings); err != nil {
		slog.Warn("pipeline: failed to record run", "error", err)
	}
	result.TotalRuns = p.store.RunCount()

	// Step 2: Detect regressions
	regressions := p.store.DetectRegressions(findings)
	result.Regressions = regressions
	if len(regressions) > 0 {
		slog.Warn("pipeline: regressions detected",
			"count", len(regressions),
		)
	}

	// Step 3: Train risk model
	p.scorer.Train(findings)
	modelPath := filepath.Join(p.config.EvidenceDir, "risk_model.json")
	if err := p.scorer.ExportModel(modelPath); err != nil {
		slog.Warn("pipeline: failed to export risk model", "error", err)
	}

	// Step 4: Get top risks
	result.TopRisks = p.scorer.Hotspots(10)

	// Step 5: Auto-remediate
	if p.config.AutoRemediate {
		remResult, err := p.remediator.Process(ctx, findings)
		if err != nil {
			slog.Warn("pipeline: remediation failed", "error", err)
		} else {
			result.Remediation = remResult
			slog.Info("pipeline: remediation complete",
				"translated", remResult.Translated,
				"auto_fixed", remResult.AutoFixed,
				"prs", remResult.PRsCreated,
			)
		}
	}

	// Step 6: Replay signal log → evolve policy
	if p.config.AutoEvolve {
		if n, err := p.signalLog.ReplayInto(p.policy); err == nil && n > 0 {
			slog.Info("pipeline: replayed signals into policy", "count", n)
		}
		evolved := p.policy.Evolve()
		result.PolicyGeneration = evolved.Generation
		policyPath := filepath.Join(p.config.EvidenceDir, "policy.json")
		_ = p.policy.ExportConfig(policyPath)
	}

	// Step 7: Evolve missions
	changes := p.missions.Evolve()
	if len(changes) > 0 {
		slog.Info("pipeline: missions evolved", "changes", len(changes))
		missionsPath := filepath.Join(p.config.EvidenceDir, "missions.json")
		_ = p.missions.SaveToFile(missionsPath)
	}

	// Step 8: Generate report hash
	reportData, _ := json.Marshal(result)
	hash := sha256.Sum256(reportData)
	result.ReportHash = "sha256:" + hex.EncodeToString(hash[:])

	// Step 9: Persist result
	resultPath := filepath.Join(p.config.EvidenceDir, "pipeline_result.json")
	if err := p.persistResult(result, resultPath); err != nil {
		slog.Warn("pipeline: failed to persist result", "error", err)
	}

	return result, nil
}

// RecordOutcome records a human's response to a finding for policy evolution.
// Appends to the durable signal log (JSONL) for crash-safe persistence.
func (p *Pipeline) RecordOutcome(findingIdx int, findings []Finding, outcome HumanOutcome) {
	if findingIdx < 0 || findingIdx >= len(findings) {
		return
	}
	f := findings[findingIdx]

	// Durable append to signal log
	entry := SignalEntry{
		FindingID: fmt.Sprintf("finding-%d", findingIdx),
		File:      f.File,
		Title:     f.Title,
		Category:  string(f.Category),
		Severity:  f.Severity,
		Outcome:   outcome,
	}
	if err := p.signalLog.Append(entry); err != nil {
		slog.Warn("pipeline: failed to append signal", "error", err)
	}

	// Also feed directly to in-memory policy for immediate use
	p.policy.RecordOutcome(f, outcome)
}

// LoadAndProcess reads a FINAL_AUDIT_REPORT.json, parses all findings
// (mechanical L1 + AI L2), and runs the full autonomy loop.
// Supports both legacy flat format and merged L1+L2 format.
func (p *Pipeline) LoadAndProcess(ctx context.Context, reportPath string) (*PipelineResult, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("pipeline: read report: %w", err)
	}

	// Auto-detect format by checking for top-level keys
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("pipeline: invalid JSON: %w", err)
	}

	var findings []Finding
	var gitSHA string

	// Extract git SHA (present in both formats)
	if raw, ok := probe["git_sha"]; ok {
		_ = json.Unmarshal(raw, &gitSHA)
	}
	if gitSHA == "" {
		gitSHA = "unknown-sha"
	}

	if _, hasMechanical := probe["mechanical"]; hasMechanical {
		// Merged format (from merge_audit_reports.sh)
		findings, err = p.aiParser.ParseMergedReport(reportPath)
		if err != nil {
			return nil, fmt.Errorf("pipeline: parse merged report: %w", err)
		}
	} else if _, hasFindings := probe["findings"]; hasFindings {
		// Legacy flat format (direct findings array)
		var legacy struct {
			Findings []Finding `json:"findings"`
		}
		if err := json.Unmarshal(data, &legacy); err != nil {
			return nil, fmt.Errorf("pipeline: parse legacy report: %w", err)
		}
		findings = legacy.Findings
		slog.Info("pipeline: loaded legacy flat report", "findings", len(findings))
	} else {
		return nil, fmt.Errorf("pipeline: unrecognized report format (no 'mechanical' or 'findings' key)")
	}

	return p.ProcessFindings(ctx, findings, gitSHA)
}

// SignalCount returns the number of outcome signals recorded.
func (p *Pipeline) SignalCount() int {
	return p.signalLog.Count()
}

// ShouldEscalate checks if the current changeset deserves a higher audit level.
func (p *Pipeline) ShouldEscalate(files []string) (bool, string) {
	return p.policy.ShouldEscalateAuditLevel(files, p.scorer)
}

func (p *Pipeline) persistResult(result *PipelineResult, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
