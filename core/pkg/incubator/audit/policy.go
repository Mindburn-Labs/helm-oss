package audit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"
)

// ── Policy Gradient Engine ──────────────────────────────────────────────────
//
// Observes human responses to audit findings and adjusts thresholds
// automatically based on which findings get fixed vs. dismissed.
//
// The policy gradient works like this:
//   1. Audit produces findings with verdicts
//   2. Humans respond: fix, dismiss, override, or ignore
//   3. PolicyEngine accumulates "signal" for each category+threshold
//   4. `Evolve()` updates config thresholds toward observed human preferences
//
// Usage:
//
//	engine := audit.NewPolicyEngine(config)
//	engine.RecordOutcome(finding, "fixed")   // human fixed it
//	engine.RecordOutcome(finding, "dismissed") // human said false positive
//	evolved := engine.Evolve()  // returns updated config

// HumanOutcome records how a human responded to a finding.
type HumanOutcome string

const (
	OutcomeFixed      HumanOutcome = "fixed"      // Human fixed the issue
	OutcomeDismissed  HumanOutcome = "dismissed"  // False positive
	OutcomeOverridden HumanOutcome = "overridden" // Explicitly allowed
	OutcomeIgnored    HumanOutcome = "ignored"    // No action (timed out)
	OutcomeEscalated  HumanOutcome = "escalated"  // Needed more investigation
)

// PolicySignal records one data point for the gradient.
type PolicySignal struct {
	Category   string       `json:"category"`
	Severity   string       `json:"severity"`
	Outcome    HumanOutcome `json:"outcome"`
	Confidence float64      `json:"confidence"`
	Timestamp  time.Time    `json:"timestamp"`
}

// PolicyConfig represents the current (potentially evolving) audit policy.
type PolicyConfig struct {
	// Thresholds per category
	CoverageFloor    float64            `json:"coverage_floor"`
	MaxStaleCount    int                `json:"max_stale_count"`
	AISeverityFilter string             `json:"ai_severity_filter"` // critical, high, medium, low
	CategoryWeights  map[string]float64 `json:"category_weights"`

	// Evolution metadata
	Generation  int       `json:"generation"`
	LastEvolved time.Time `json:"last_evolved"`
	Frozen      bool      `json:"frozen"` // If true, don't evolve

	// Signal history
	Signals []PolicySignal `json:"signals"`
}

// DefaultPolicyConfig returns the baseline policy.
func DefaultPolicyConfig() *PolicyConfig {
	return &PolicyConfig{
		CoverageFloor:    30.0,
		MaxStaleCount:    50,
		AISeverityFilter: "medium",
		CategoryWeights: map[string]float64{
			"security":      1.0,
			"architecture":  0.8,
			"accessibility": 0.6,
			"brand_tone":    0.4,
			"motion":        0.3,
		},
		Generation: 0,
	}
}

// PolicyEngine manages policy evolution.
type PolicyEngine struct {
	config *PolicyConfig
}

// NewPolicyEngine creates an engine with the given config.
func NewPolicyEngine(config *PolicyConfig) *PolicyEngine {
	if config == nil {
		config = DefaultPolicyConfig()
	}
	return &PolicyEngine{config: config}
}

// RecordOutcome adds a signal from a human response.
func (e *PolicyEngine) RecordOutcome(finding Finding, outcome HumanOutcome) {
	signal := PolicySignal{
		Category:   string(finding.Category),
		Severity:   finding.Severity,
		Outcome:    outcome,
		Confidence: 0, // Will be set by the translator if available
		Timestamp:  time.Now().UTC(),
	}
	e.config.Signals = append(e.config.Signals, signal)
}

// Evolve runs the policy gradient and returns updated config.
//
// The gradient logic:
//   - High fix rate → raise category weight (these findings matter)
//   - High dismiss rate → lower category weight (noise)
//   - Pattern: if coverage FAILs are always fixed → raise coverage floor
//   - Pattern: if stale TODO count always exceeds threshold → raise threshold
func (e *PolicyEngine) Evolve() *PolicyConfig {
	if e.config.Frozen {
		slog.Info("policy engine: config frozen, skipping evolution")
		return e.config
	}

	if len(e.config.Signals) < 10 {
		slog.Info("policy engine: insufficient signals for evolution",
			"signals", len(e.config.Signals),
			"minimum", 10)
		return e.config
	}

	// Calculate fix rate and dismiss rate per category
	categoryStats := make(map[string]*struct{ fixes, dismissals, total int })

	for _, sig := range e.config.Signals {
		cat := sig.Category
		if _, ok := categoryStats[cat]; !ok {
			categoryStats[cat] = &struct{ fixes, dismissals, total int }{}
		}
		categoryStats[cat].total++
		switch sig.Outcome {
		case OutcomeFixed, OutcomeEscalated:
			categoryStats[cat].fixes++
		case OutcomeDismissed, OutcomeOverridden:
			categoryStats[cat].dismissals++
		}
	}

	// Adjust category weights
	for cat, stats := range categoryStats {
		if stats.total < 3 {
			continue // Not enough data
		}

		fixRate := float64(stats.fixes) / float64(stats.total)
		dismissRate := float64(stats.dismissals) / float64(stats.total)

		currentWeight := e.config.CategoryWeights[cat]
		if currentWeight == 0 {
			currentWeight = 0.5
		}

		// Gradient step
		var delta float64
		if fixRate > 0.7 {
			// Humans consistently fix these → increase weight
			delta = 0.05
		} else if dismissRate > 0.7 {
			// Humans consistently dismiss → decrease weight (noise)
			delta = -0.1
		} else if fixRate > dismissRate {
			delta = 0.02
		} else {
			delta = -0.02
		}

		newWeight := math.Max(0.1, math.Min(1.0, currentWeight+delta))
		e.config.CategoryWeights[cat] = newWeight

		slog.Info("policy evolved",
			"category", cat,
			"fix_rate", fmt.Sprintf("%.0f%%", fixRate*100),
			"dismiss_rate", fmt.Sprintf("%.0f%%", dismissRate*100),
			"weight", fmt.Sprintf("%.2f → %.2f", currentWeight, newWeight),
		)
	}

	// Evolve coverage floor
	coverageSignals := 0
	coverageFixes := 0
	for _, sig := range e.config.Signals {
		if sig.Category == "coverage" {
			coverageSignals++
			if sig.Outcome == OutcomeFixed {
				coverageFixes++
			}
		}
	}
	if coverageSignals >= 5 && float64(coverageFixes)/float64(coverageSignals) > 0.8 {
		// People always fix coverage failures → raise the floor
		newFloor := math.Min(80.0, e.config.CoverageFloor+5.0)
		slog.Info("policy: raising coverage floor",
			"from", e.config.CoverageFloor,
			"to", newFloor,
		)
		e.config.CoverageFloor = newFloor
	}

	e.config.Generation++
	e.config.LastEvolved = time.Now().UTC()

	return e.config
}

// ShouldEscalateAuditLevel decides if a PR deserves L3 instead of L1.
func (e *PolicyEngine) ShouldEscalateAuditLevel(files []string, scorer *RiskScorer) (bool, string) {
	if scorer == nil {
		return false, ""
	}

	scores := scorer.ScoreFiles(files)

	var maxScore float64
	var maxFile string
	highRiskCount := 0

	for _, s := range scores {
		if s.Score > maxScore {
			maxScore = s.Score
			maxFile = s.File
		}
		if s.Score > 0.7 {
			highRiskCount++
		}
	}

	// Escalate if any file is very high risk
	if maxScore > 0.8 {
		return true, fmt.Sprintf("file %s has risk score %.2f", maxFile, maxScore)
	}

	// Escalate if many files are moderately risky
	if highRiskCount > 3 {
		return true, fmt.Sprintf("%d files above 0.7 risk", highRiskCount)
	}

	return false, ""
}

// ExportConfig persists the evolved policy.
func (e *PolicyEngine) ExportConfig(path string) error {
	data, err := json.MarshalIndent(e.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadPolicyConfig loads a previously saved policy.
func LoadPolicyConfig(path string) (*PolicyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config PolicyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
