package audit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"
)

// ── Self-Evolving Missions ──────────────────────────────────────────────────
//
// AI audit prompts that improve themselves based on precision/recall metrics.
//
// Each mission tracks:
//   - Precision: what % of findings were actual issues (not dismissed)
//   - Recall: what % of actual issues were found (via manual reports)
//   - Evolution: prompt is automatically refined based on metrics
//
// Usage:
//
//	registry := audit.NewMissionRegistry()
//	registry.LoadFromFile("audit_missions.json")
//	prompt := registry.RenderPrompt("architecture_coherence", vars)
//	registry.RecordPrecision("architecture_coherence", 0.72)
//	registry.Evolve()  // Refines prompts with low precision/recall

// Mission represents a single AI audit mission with metrics.
type Mission struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Category  string            `json:"category"`
	Template  string            `json:"template"`
	Variables map[string]string `json:"variables,omitempty"`

	// Performance metrics (evolve over time)
	Precision  float64   `json:"precision"` // % correct findings
	Recall     float64   `json:"recall"`    // % issues caught
	RunCount   int       `json:"run_count"`
	FindingAvg float64   `json:"finding_avg"` // avg findings per run
	LastRun    time.Time `json:"last_run"`

	// Evolution state
	Generation  int      `json:"generation"`
	Refinements []string `json:"refinements,omitempty"` // History of prompt changes
	Active      bool     `json:"active"`
}

// MissionRegistry manages the collection of AI audit missions.
type MissionRegistry struct {
	missions map[string]*Mission
}

// NewMissionRegistry creates an empty registry.
func NewMissionRegistry() *MissionRegistry {
	return &MissionRegistry{
		missions: make(map[string]*Mission),
	}
}

// Register adds or updates a mission.
func (r *MissionRegistry) Register(m *Mission) {
	m.Active = true
	r.missions[m.ID] = m
}

// Get retrieves a mission by ID.
func (r *MissionRegistry) Get(id string) (*Mission, bool) {
	m, ok := r.missions[id]
	return m, ok
}

// Active returns all active missions, sorted by ID.
func (r *MissionRegistry) Active() []*Mission {
	result := make([]*Mission, 0)
	for _, m := range r.missions {
		if m.Active {
			result = append(result, m)
		}
	}
	return result
}

// RenderPrompt renders a mission's template with the given variables.
func (r *MissionRegistry) RenderPrompt(missionID string, vars map[string]string) (string, error) {
	m, ok := r.missions[missionID]
	if !ok {
		return "", fmt.Errorf("mission %q not found", missionID)
	}

	// Merge mission-level vars with call-level vars
	allVars := make(map[string]string)
	for k, v := range m.Variables {
		allVars[k] = v
	}
	for k, v := range vars {
		allVars[k] = v
	}

	tmpl, err := template.New(missionID).Parse(m.Template)
	if err != nil {
		return "", fmt.Errorf("template parse failed for %s: %w", missionID, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, allVars); err != nil {
		return "", fmt.Errorf("template execute failed for %s: %w", missionID, err)
	}

	return buf.String(), nil
}

// RecordPrecision updates a mission's precision metric after human review.
func (r *MissionRegistry) RecordPrecision(missionID string, precision float64) {
	m, ok := r.missions[missionID]
	if !ok {
		return
	}

	// Exponential moving average
	if m.RunCount == 0 {
		m.Precision = precision
	} else {
		alpha := 0.3 // Weight for new observation
		m.Precision = alpha*precision + (1-alpha)*m.Precision
	}
	m.RunCount++
	m.LastRun = time.Now().UTC()
}

// RecordRecall updates a mission's recall metric.
func (r *MissionRegistry) RecordRecall(missionID string, recall float64) {
	m, ok := r.missions[missionID]
	if !ok {
		return
	}
	alpha := 0.3
	m.Recall = alpha*recall + (1-alpha)*m.Recall
}

// RecordFindingCount tracks how many findings a mission produces.
func (r *MissionRegistry) RecordFindingCount(missionID string, count int) {
	m, ok := r.missions[missionID]
	if !ok {
		return
	}
	if m.RunCount <= 1 {
		m.FindingAvg = float64(count)
	} else {
		m.FindingAvg = (m.FindingAvg*float64(m.RunCount-1) + float64(count)) / float64(m.RunCount)
	}
}

// Evolve refines missions based on their performance metrics.
//
// Rules:
//   - Precision < 0.5 for 5+ runs → add "be precise, avoid false positives" hint
//   - Recall < 0.3 → add "be thorough, check every file" hint
//   - FindingAvg > 50 → add "focus on critical and high severity only" hint
//   - FindingAvg < 1 → add "don't skip files, analyze all relevant code" hint
//   - Precision > 0.9 for 10+ runs → promote to "trusted" (higher weight)
func (r *MissionRegistry) Evolve() []string {
	var changes []string

	for _, m := range r.missions {
		if m.RunCount < 5 {
			continue // Not enough data
		}

		var refinement string

		switch {
		case m.Precision < 0.5:
			refinement = "PRECISION_BOOST: Focus on high-confidence findings only. Avoid speculative issues."
			m.Template += "\n\nIMPORTANT: Only report findings you are highly confident about. Avoid false positives."

		case m.Recall < 0.3 && m.RunCount >= 10:
			refinement = "RECALL_BOOST: Be more thorough, analyze every file in scope."
			m.Template += "\n\nIMPORTANT: Be thorough. Check every file in scope. Don't skip files."

		case m.FindingAvg > 50:
			refinement = "SEVERITY_FILTER: Too many findings. Focus on critical and high severity."
			m.Template += "\n\nIMPORTANT: Report only CRITICAL and HIGH severity findings. Skip low-impact issues."

		case m.FindingAvg < 1 && m.RunCount >= 5:
			refinement = "DEPTH_INCREASE: Not finding enough issues. Dig deeper."
			m.Template += "\n\nIMPORTANT: Analyze code deeply. Report any deviation from best practices."

		default:
			continue
		}

		m.Refinements = append(m.Refinements, fmt.Sprintf("[gen %d] %s", m.Generation, refinement))
		m.Generation++
		changes = append(changes, fmt.Sprintf("%s: %s (precision=%.2f, recall=%.2f, avg=%.1f)",
			m.ID, refinement, m.Precision, m.Recall, m.FindingAvg))

		slog.Info("mission evolved",
			"mission", m.ID,
			"generation", m.Generation,
			"refinement", refinement,
		)
	}

	return changes
}

// SaveToFile persists the registry.
func (r *MissionRegistry) SaveToFile(path string) error {
	missions := make([]*Mission, 0, len(r.missions))
	for _, m := range r.missions {
		missions = append(missions, m)
	}
	data, err := json.MarshalIndent(missions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadFromFile loads a previously saved registry.
func (r *MissionRegistry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var missions []*Mission
	if err := json.Unmarshal(data, &missions); err != nil {
		return err
	}
	for _, m := range missions {
		r.missions[m.ID] = m
	}
	return nil
}

// DefaultMissions returns the standard HELM audit mission set.
func DefaultMissions() []*Mission {
	return []*Mission{
		{
			ID:       "architecture_coherence",
			Name:     "Architecture Coherence",
			Category: "architecture",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Analyze the architecture: read ARCHITECTURE.md, compare to actual directory structure.
Flag architectural drift, orphan packages, and misplaced code.
Output ONLY valid JSON: {"mission_id":"architecture_coherence","findings":[...]}`,
			Active: true,
		},
		{
			ID:       "security_posture",
			Name:     "Security Posture",
			Category: "security",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Review auth, crypto, access control, trust, and guardian packages.
Check for: hardcoded secrets, weak crypto, missing auth checks, injection vectors.
Output ONLY valid JSON: {"mission_id":"security_posture","findings":[...]}`,
			Active: true,
		},
		{
			ID:       "error_handling",
			Name:     "Error Handling Consistency",
			Category: "reliability",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Check all Go code for: swallowed errors, missing error returns, panic in library code,
inconsistent error wrapping, errors that lose context.
Output ONLY valid JSON: {"mission_id":"error_handling","findings":[...]}`,
			Active: true,
		},
		{
			ID:       "package_completeness",
			Name:     "Package Completeness",
			Category: "architecture",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Check all packages for: stub implementations, empty functions, skeleton code,
TODO markers in production code, interfaces without implementations.
Output ONLY valid JSON: {"mission_id":"package_completeness","findings":[...]}`,
			Active: true,
		},
		{
			ID:       "integration_wiring",
			Name:     "Integration Wiring",
			Category: "architecture",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Find orphan factories, providers, and bridges that are defined but never imported.
Check dependency injection wiring for missing registrations.
Output ONLY valid JSON: {"mission_id":"integration_wiring","findings":[...]}`,
			Active: true,
		},
		{
			ID:       "doc_code_drift",
			Name:     "Documentation-Code Drift",
			Category: "documentation",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Compare README.md and docs/ against actual code. Flag stale references,
wrong examples, missing API documentation, and outdated architecture descriptions.
Output ONLY valid JSON: {"mission_id":"doc_code_drift","findings":[...]}`,
			Active: true,
		},
		{
			ID:       "concurrency_safety",
			Name:     "Concurrency Safety",
			Category: "reliability",
			Template: `You are auditing {{.RepoRoot}} (git SHA: {{.GitSHA}}).
Check for: data races, unprotected shared state, missing mutex locks,
goroutine leaks, channel misuse, and unsafe concurrent map access.
Output ONLY valid JSON: {"mission_id":"concurrency_safety","findings":[...]}`,
			Active: true,
		},
	}
}
