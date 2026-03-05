package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ── Audit Learning Store ────────────────────────────────────────────────────
//
// Persists findings across audit runs to build institutional knowledge.
// Each run is a "generation" in the store. The store supports:
//   - Trend queries: "how has FAIL count changed over last 10 runs?"
//   - Pattern matching: "which files fail most often?"
//   - Regression detection: "did this finding exist before and get re-introduced?"
//
// Usage:
//
//	store := audit.NewLearningStore("/path/to/audit/history")
//	store.RecordRun(runID, findings)
//	trends := store.Trends("security", 10)
//	regressions := store.DetectRegressions(currentFindings)

// AuditRun represents a single completed audit run.
type AuditRun struct {
	RunID     string     `json:"run_id"`
	GitSHA    string     `json:"git_sha"`
	Timestamp time.Time  `json:"timestamp"`
	Findings  []Finding  `json:"findings"`
	Summary   RunSummary `json:"summary"`
}

// RunSummary provides aggregate stats for a run.
type RunSummary struct {
	TotalFindings int            `json:"total_findings"`
	ByVerdict     map[string]int `json:"by_verdict"`
	ByCategory    map[string]int `json:"by_category"`
	BySeverity    map[string]int `json:"by_severity"`
	CoverageScore float64        `json:"coverage_score"`
}

// Trend represents a metric over time.
type Trend struct {
	Category   string       `json:"category"`
	DataPoints []TrendPoint `json:"data_points"`
	Direction  string       `json:"direction"` // improving, degrading, stable
	Slope      float64      `json:"slope"`
}

// TrendPoint is a single data point in a trend.
type TrendPoint struct {
	RunID     string    `json:"run_id"`
	Timestamp time.Time `json:"timestamp"`
	Value     int       `json:"value"`
}

// Regression is a finding that was previously fixed but has returned.
type Regression struct {
	Finding         Finding `json:"finding"`
	FirstSeenRun    string  `json:"first_seen_run"`
	FixedInRun      string  `json:"fixed_in_run"`
	RegressedRun    string  `json:"regressed_run"`
	OccurrenceCount int     `json:"occurrence_count"`
}

// LearningStore persists audit history for cross-run learning.
type LearningStore struct {
	mu       sync.RWMutex
	basePath string
	runs     []*AuditRun
	loaded   bool
}

// NewLearningStore creates a store rooted at the given path.
func NewLearningStore(basePath string) *LearningStore {
	return &LearningStore{
		basePath: basePath,
	}
}

// RecordRun persists a completed audit run.
func (s *LearningStore) RecordRun(runID, gitSHA string, findings []Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoaded(); err != nil {
		return err
	}

	run := &AuditRun{
		RunID:     runID,
		GitSHA:    gitSHA,
		Timestamp: time.Now().UTC(),
		Findings:  findings,
		Summary:   summarizeRun(findings),
	}

	s.runs = append(s.runs, run)

	// Persist to disk
	return s.persistRun(run)
}

// Trends returns the trend for a specific category over the last N runs.
func (s *LearningStore) Trends(category string, lastN int) *Trend {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_ = s.ensureLoaded()

	trend := &Trend{Category: category}

	start := 0
	if len(s.runs) > lastN {
		start = len(s.runs) - lastN
	}

	for _, run := range s.runs[start:] {
		count := 0
		for _, f := range run.Findings {
			if string(f.Category) == category && f.Verdict == "FAIL" {
				count++
			}
		}
		trend.DataPoints = append(trend.DataPoints, TrendPoint{
			RunID:     run.RunID,
			Timestamp: run.Timestamp,
			Value:     count,
		})
	}

	// Calculate direction
	if len(trend.DataPoints) >= 2 {
		first := trend.DataPoints[0].Value
		last := trend.DataPoints[len(trend.DataPoints)-1].Value
		if last < first {
			trend.Direction = "improving"
			trend.Slope = float64(last-first) / float64(len(trend.DataPoints))
		} else if last > first {
			trend.Direction = "degrading"
			trend.Slope = float64(last-first) / float64(len(trend.DataPoints))
		} else {
			trend.Direction = "stable"
			trend.Slope = 0
		}
	}

	return trend
}

// DetectRegressions finds findings that existed before, were fixed, and returned.
func (s *LearningStore) DetectRegressions(currentFindings []Finding) []Regression {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_ = s.ensureLoaded()

	if len(s.runs) < 2 {
		return nil
	}

	// Build fingerprint → history map
	type findingHistory struct {
		firstSeen string
		lastSeen  string
		absent    []string // runs where this finding was absent
		present   []string // runs where this finding was present
	}

	fingerprints := make(map[string]*findingHistory)

	for _, run := range s.runs {
		presentInRun := make(map[string]bool)
		for _, f := range run.Findings {
			if f.Verdict != "FAIL" {
				continue
			}
			fp := fingerprintFinding(f)
			presentInRun[fp] = true

			if _, ok := fingerprints[fp]; !ok {
				fingerprints[fp] = &findingHistory{firstSeen: run.RunID}
			}
			fingerprints[fp].lastSeen = run.RunID
			fingerprints[fp].present = append(fingerprints[fp].present, run.RunID)
		}

		// Record absences
		for fp, hist := range fingerprints {
			if !presentInRun[fp] && hist.firstSeen != run.RunID {
				hist.absent = append(hist.absent, run.RunID)
			}
		}
	}

	// Check current findings for regressions
	var regressions []Regression
	for _, f := range currentFindings {
		if f.Verdict != "FAIL" {
			continue
		}
		fp := fingerprintFinding(f)
		hist, ok := fingerprints[fp]
		if !ok {
			continue // Brand new finding
		}

		// Regression: was present, then absent, now back
		if len(hist.absent) > 0 && len(hist.present) > 1 {
			regressions = append(regressions, Regression{
				Finding:         f,
				FirstSeenRun:    hist.firstSeen,
				FixedInRun:      hist.absent[0],
				RegressedRun:    hist.present[len(hist.present)-1],
				OccurrenceCount: len(hist.present),
			})
		}
	}

	return regressions
}

// FileRiskHistory returns the FAIL count by file across all runs.
func (s *LearningStore) FileRiskHistory() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_ = s.ensureLoaded()

	counts := make(map[string]int)
	for _, run := range s.runs {
		for _, f := range run.Findings {
			if f.Verdict == "FAIL" {
				counts[f.File]++
			}
		}
	}
	return counts
}

// RunCount returns the number of recorded runs.
func (s *LearningStore) RunCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = s.ensureLoaded()
	return len(s.runs)
}

func (s *LearningStore) ensureLoaded() error {
	if s.loaded {
		return nil
	}
	s.loaded = true

	if err := os.MkdirAll(s.basePath, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.basePath, e.Name()))
		if err != nil {
			continue
		}
		var run AuditRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		s.runs = append(s.runs, &run)
	}

	// Sort by timestamp
	sort.Slice(s.runs, func(i, j int) bool {
		return s.runs[i].Timestamp.Before(s.runs[j].Timestamp)
	})

	return nil
}

func (s *LearningStore) persistRun(run *AuditRun) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	idSuffix := run.RunID
	if len(idSuffix) > 8 {
		idSuffix = idSuffix[:8]
	}
	filename := fmt.Sprintf("run_%s_%s.json",
		run.Timestamp.Format("20060102_150405"),
		idSuffix)
	return os.WriteFile(filepath.Join(s.basePath, filename), data, 0o644)
}

func summarizeRun(findings []Finding) RunSummary {
	s := RunSummary{
		TotalFindings: len(findings),
		ByVerdict:     make(map[string]int),
		ByCategory:    make(map[string]int),
		BySeverity:    make(map[string]int),
	}
	for _, f := range findings {
		s.ByVerdict[f.Verdict]++
		s.ByCategory[string(f.Category)]++
		s.BySeverity[f.Severity]++
	}
	return s
}

func fingerprintFinding(f Finding) string {
	return fmt.Sprintf("%s::%s::%s", f.File, f.Category, f.Title)
}
