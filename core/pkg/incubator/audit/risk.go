package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// ── Predictive Risk Scorer ──────────────────────────────────────────────────
//
// Scores files and packages by their historical audit FAIL rate, commit
// velocity, time-since-last-change, and dependency freshness.
//
// Used to:
//   - Prioritize which files to audit first (differential mode)
//   - Pre-emptively escalate audit level on risky PRs
//   - Feed the policy gradient engine with risk signals
//
// Usage:
//
//	scorer := audit.NewRiskScorer(store)
//	scores := scorer.ScoreFiles(files)
//	hotspots := scorer.Hotspots(10)

// RiskScore represents the computed risk for a single file.
type RiskScore struct {
	File            string   `json:"file"`
	Score           float64  `json:"score"` // 0.0 (safe) → 1.0 (critical)
	HistoricalFails int      `json:"historical_fails"`
	RecentCommits   int      `json:"recent_commits"` // in last 30 days
	DaysSinceChange int      `json:"days_since_change"`
	Complexity      int      `json:"complexity"` // lines of code approx
	RiskFactors     []string `json:"risk_factors"`
}

// RiskModel holds the trained weights for risk scoring.
type RiskModel struct {
	// Weights for each factor (they sum to 1.0)
	HistoryWeight    float64 `json:"history_weight"`
	VelocityWeight   float64 `json:"velocity_weight"`
	ComplexityWeight float64 `json:"complexity_weight"`
	RecencyWeight    float64 `json:"recency_weight"`

	// Historical data
	FileFailCounts map[string]int `json:"file_fail_counts"`
	FilePassCounts map[string]int `json:"file_pass_counts"`

	// Model metadata
	TrainedAt       time.Time `json:"trained_at"`
	TrainingSamples int       `json:"training_samples"`
	ModelHash       string    `json:"model_hash"`
}

// DefaultRiskModel returns initial weights before training.
func DefaultRiskModel() *RiskModel {
	return &RiskModel{
		HistoryWeight:    0.40,
		VelocityWeight:   0.25,
		ComplexityWeight: 0.20,
		RecencyWeight:    0.15,
		FileFailCounts:   make(map[string]int),
		FilePassCounts:   make(map[string]int),
		TrainedAt:        time.Now().UTC(),
	}
}

// RiskScorer computes risk scores for files based on multiple signals.
type RiskScorer struct {
	model    *RiskModel
	repoRoot string
}

// NewRiskScorer creates a scorer. If model is nil, uses defaults.
func NewRiskScorer(model *RiskModel, repoRoot string) *RiskScorer {
	if model == nil {
		model = DefaultRiskModel()
	}
	return &RiskScorer{model: model, repoRoot: repoRoot}
}

// Train updates the model with findings from a completed audit run.
func (s *RiskScorer) Train(findings []Finding) {
	for _, f := range findings {
		if f.File == "" {
			continue
		}
		if f.Verdict == "FAIL" {
			s.model.FileFailCounts[f.File]++
		} else if f.Verdict == "PASS" {
			s.model.FilePassCounts[f.File]++
		}
		s.model.TrainingSamples++
	}
	s.model.TrainedAt = time.Now().UTC()
	s.model.ModelHash = s.computeModelHash()
}

// ScoreFile computes risk for a single file.
func (s *RiskScorer) ScoreFile(file string) RiskScore {
	rs := RiskScore{File: file}

	// Factor 1: Historical fail rate + absolute count
	fails := s.model.FileFailCounts[file]
	passes := s.model.FilePassCounts[file]
	rs.HistoricalFails = fails

	var historyScore float64
	total := fails + passes
	if total > 0 {
		failRate := float64(fails) / float64(total)
		// Blend rate (70%) with absolute count magnitude (30%)
		// to differentiate files with same rate but different counts
		countMagnitude := math.Min(float64(fails)/10.0, 1.0)
		historyScore = 0.7*failRate + 0.3*countMagnitude
	}
	if fails >= 3 {
		rs.RiskFactors = append(rs.RiskFactors, fmt.Sprintf("repeat_offender (%d fails)", fails))
	}

	// Factor 2: Commit velocity (recent changes = higher risk)
	recentCommits := s.getRecentCommits(file, 30)
	rs.RecentCommits = recentCommits
	velocityScore := math.Min(float64(recentCommits)/10.0, 1.0) // 10+ commits = max
	if recentCommits >= 5 {
		rs.RiskFactors = append(rs.RiskFactors, fmt.Sprintf("high_velocity (%d commits/30d)", recentCommits))
	}

	// Factor 3: Complexity (lines of code)
	complexity := s.getComplexity(file)
	rs.Complexity = complexity
	complexityScore := math.Min(float64(complexity)/500.0, 1.0) // 500+ lines = max
	if complexity > 300 {
		rs.RiskFactors = append(rs.RiskFactors, fmt.Sprintf("complex (%d lines)", complexity))
	}

	// Factor 4: Recency (recently changed files = more likely to have issues)
	daysSince := s.getDaysSinceChange(file)
	rs.DaysSinceChange = daysSince
	var recencyScore float64
	if daysSince <= 7 {
		recencyScore = 0.8
		rs.RiskFactors = append(rs.RiskFactors, "recently_changed")
	} else if daysSince <= 30 {
		recencyScore = 0.4
	} else {
		recencyScore = 0.1
	}

	// Weighted combination
	rs.Score = (historyScore * s.model.HistoryWeight) +
		(velocityScore * s.model.VelocityWeight) +
		(complexityScore * s.model.ComplexityWeight) +
		(recencyScore * s.model.RecencyWeight)

	// Clamp to [0, 1]
	rs.Score = math.Max(0, math.Min(1, rs.Score))

	return rs
}

// ScoreFiles scores multiple files and sorts by risk (highest first).
func (s *RiskScorer) ScoreFiles(files []string) []RiskScore {
	scores := make([]RiskScore, 0, len(files))
	for _, f := range files {
		scores = append(scores, s.ScoreFile(f))
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})
	return scores
}

// Hotspots returns the N riskiest files from the entire model history.
func (s *RiskScorer) Hotspots(n int) []RiskScore {
	allFiles := make(map[string]bool)
	for f := range s.model.FileFailCounts {
		allFiles[f] = true
	}
	for f := range s.model.FilePassCounts {
		allFiles[f] = true
	}

	files := make([]string, 0, len(allFiles))
	for f := range allFiles {
		files = append(files, f)
	}

	scores := s.ScoreFiles(files)
	if n > len(scores) {
		n = len(scores)
	}
	return scores[:n]
}

// ExportModel persists the trained model to disk.
func (s *RiskScorer) ExportModel(path string) error {
	data, err := json.MarshalIndent(s.model, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadModel loads a previously trained model.
func LoadRiskModel(path string) (*RiskModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var model RiskModel
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *RiskScorer) getRecentCommits(file string, days int) int {
	if s.repoRoot == "" {
		return 0
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	cmd := exec.Command("git", "-C", s.repoRoot, "log",
		"--oneline", "--since", since, "--", file)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

func (s *RiskScorer) getComplexity(file string) int {
	fullPath := file
	if s.repoRoot != "" && !strings.HasPrefix(file, "/") {
		fullPath = s.repoRoot + "/" + file
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return 0
	}
	return strings.Count(string(data), "\n")
}

func (s *RiskScorer) getDaysSinceChange(file string) int {
	if s.repoRoot == "" {
		return 999
	}
	cmd := exec.Command("git", "-C", s.repoRoot, "log", "-1",
		"--format=%ci", "--", file)
	out, err := cmd.Output()
	if err != nil {
		return 999
	}
	dateStr := strings.TrimSpace(string(out))
	if dateStr == "" {
		return 999
	}
	t, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
	if err != nil {
		return 999
	}
	return int(time.Since(t).Hours() / 24)
}

func (s *RiskScorer) computeModelHash() string {
	data, _ := json.Marshal(s.model.FileFailCounts)
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:8])
}
