package audit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

// ── AI Finding Parser ───────────────────────────────────────────────────────
//
// Parses the FINAL_AUDIT_REPORT.json (output of merge_audit_reports.sh)
// into structured []Finding for the pipeline.
//
// Two parsing layers:
//   1. Mechanical (L1) — from report.mechanical.details
//   2. AI Missions (L2) — from report.ai.mission_results
//
// Usage:
//
//   parser := audit.NewAIParser(missions)
//   findings, err := parser.ParseMergedReport("data/evidence/FINAL_AUDIT_REPORT.json")

// MergedReport represents the full FINAL_AUDIT_REPORT.json structure.
type MergedReport struct {
	Version    string          `json:"version"`
	Timestamp  string          `json:"timestamp"`
	GitSHA     string          `json:"git_sha"`
	Verdict    string          `json:"verdict"`
	Hash       string          `json:"report_hash"`
	Mechanical MechanicalLayer `json:"mechanical"`
	AI         AILayer         `json:"ai"`
}

// MechanicalLayer contains L1 mechanical audit results.
type MechanicalLayer struct {
	Sections int             `json:"sections"`
	Pass     int             `json:"pass"`
	Fail     int             `json:"fail"`
	Warn     int             `json:"warn"`
	Skip     int             `json:"skip"`
	Details  []SectionDetail `json:"details"`
}

// SectionDetail is one §section from L1.
type SectionDetail struct {
	Section string   `json:"section"`
	Name    string   `json:"name"`
	Verdict string   `json:"verdict"`
	Summary string   `json:"summary"`
	Count   int      `json:"count,omitempty"`
	Issues  []string `json:"issues,omitempty"`
}

// AILayer contains L2 AI mission results.
type AILayer struct {
	Model          string          `json:"model"`
	Missions       int             `json:"missions"`
	Completed      int             `json:"completed"`
	Findings       []Finding       `json:"findings"`
	CoverageScore  float64         `json:"coverage_score"`
	MissionResults []MissionResult `json:"mission_results"`
}

// MissionResult is one AI mission's output.
type MissionResult struct {
	MissionID    string `json:"mission_id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Status       string `json:"status"`
	Output       string `json:"output"`
	FindingCount int    `json:"finding_count"`
}

// AIParser converts AI mission output into structured findings.
type AIParser struct {
	missions *MissionRegistry
	// Regex patterns for extracting findings from free-form AI text
	filePattern     *regexp.Regexp
	severityPattern *regexp.Regexp
	findingPattern  *regexp.Regexp
}

// NewAIParser creates a parser with the mission registry for category lookup.
func NewAIParser(missions *MissionRegistry) *AIParser {
	return &AIParser{
		missions: missions,
		// Match patterns like "File: path/to/file.go" or "**path/to/file.go**"
		filePattern: regexp.MustCompile(`(?i)(?:file|path|in)\s*:\s*` + "`?" + `([^\s` + "`" + `]+\.(?:go|ts|js|py|rs|yaml|yml|json|sql|sh|dockerfile))` + "`?"),
		// Match severity indicators
		severityPattern: regexp.MustCompile(`(?i)\b(critical|high|medium|low|info)\b`),
		// Match numbered finding items
		findingPattern: regexp.MustCompile(`(?m)^\s*(?:\d+[\.\)]\s*|[-*]\s+)(.+)$`),
	}
}

// ParseMergedReport reads a FINAL_AUDIT_REPORT.json and extracts all findings.
func (p *AIParser) ParseMergedReport(reportPath string) ([]Finding, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("ai_parser: read report: %w", err)
	}

	var report MergedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("ai_parser: unmarshal: %w", err)
	}

	var findings []Finding

	// Layer 1: Mechanical findings from section details
	mechanicalFindings := p.parseMechanicalDetails(report.Mechanical)
	findings = append(findings, mechanicalFindings...)

	// Layer 2: AI mission findings
	aiFindings := p.parseAIMissions(report.AI)
	findings = append(findings, aiFindings...)

	// Also include any pre-structured AI findings
	findings = append(findings, report.AI.Findings...)

	slog.Info("ai_parser: parsed merged report",
		"mechanical", len(mechanicalFindings),
		"ai_parsed", len(aiFindings),
		"ai_structured", len(report.AI.Findings),
		"total", len(findings),
	)

	return findings, nil
}

// parseMechanicalDetails converts §section results into Finding structs.
func (p *AIParser) parseMechanicalDetails(mech MechanicalLayer) []Finding {
	var findings []Finding

	sectionCategories := map[string]RemediationCategory{
		"go_mod_tidy":            RemediationDependency,
		"go_vet":                 RemediationArchitecture,
		"govulncheck":            RemediationSecurity,
		"golangci_lint":          RemediationArchitecture,
		"coverage_gate":          RemediationReliability,
		"structured_logging":     RemediationReliability,
		"secret_scan":            RemediationSecurity,
		"schema_validation":      RemediationArchitecture,
		"dockerfile_audit":       RemediationSecurity,
		"forbidden_patterns":     RemediationSecurity,
		"codeowners_coverage":    RemediationReliability,
		"doc_link_integrity":     RemediationDocumentation,
		"migration_integrity":    RemediationReliability,
		"binary_size_budget":     RemediationReliability,
		"orphan_packages":        RemediationArchitecture,
		"interface_impl_drift":   RemediationArchitecture,
		"unused_exports":         RemediationArchitecture,
		"test_orphans":           RemediationArchitecture,
		"incomplete_integration": RemediationArchitecture,
		"api_route_coverage":     RemediationSecurity,
		"stale_todos":            RemediationReliability,
		"dep_freshness":          RemediationDependency,
		"import_boundaries":      RemediationArchitecture,
	}

	for _, detail := range mech.Details {
		cat, ok := sectionCategories[detail.Name]
		if !ok {
			cat = RemediationArchitecture
		}

		severity := mapVerdictToSeverity(detail.Verdict)

		// If issues are listed, create one finding per issue
		if len(detail.Issues) > 0 {
			for _, issue := range detail.Issues {
				file := p.extractFileFromText(issue)
				findings = append(findings, Finding{
					File:     file,
					Category: cat,
					Severity: severity,
					Verdict:  detail.Verdict,
					Title:    fmt.Sprintf("[%s] %s", detail.Section, truncateStr(issue, 120)),
				})
			}
		} else {
			// One finding for the whole section
			findings = append(findings, Finding{
				File:     "",
				Category: cat,
				Severity: severity,
				Verdict:  detail.Verdict,
				Title:    fmt.Sprintf("[%s] %s: %s", detail.Section, detail.Name, detail.Summary),
			})
		}
	}

	return findings
}

// parseAIMissions extracts findings from free-form AI mission output.
func (p *AIParser) parseAIMissions(ai AILayer) []Finding {
	var findings []Finding

	for _, result := range ai.MissionResults {
		if result.Status != "completed" && result.Status != "done" {
			continue
		}

		missionFindings := p.ParseMissionOutput(result.MissionID, result.Category, result.Output)
		findings = append(findings, missionFindings...)

		// Record metrics back to mission registry
		if p.missions != nil {
			p.missions.RecordFindingCount(result.MissionID, len(missionFindings))
		}
	}

	return findings
}

// ParseMissionOutput extracts structured findings from a single mission's free-form text.
func (p *AIParser) ParseMissionOutput(missionID, category, rawOutput string) []Finding {
	if rawOutput == "" {
		return nil
	}

	var findings []Finding

	// Strategy 1: Try to parse as JSON array of findings
	var jsonFindings []Finding
	if err := json.Unmarshal([]byte(rawOutput), &jsonFindings); err == nil {
		for i := range jsonFindings {
			if jsonFindings[i].Category == "" {
				jsonFindings[i].Category = RemediationCategory(category)
			}
			if jsonFindings[i].Verdict == "" {
				jsonFindings[i].Verdict = "FAIL"
			}
		}
		return jsonFindings
	}

	// Strategy 2: Try to parse as JSON object with "findings" key
	var wrapper struct {
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(rawOutput), &wrapper); err == nil && len(wrapper.Findings) > 0 {
		for i := range wrapper.Findings {
			if wrapper.Findings[i].Category == "" {
				wrapper.Findings[i].Category = RemediationCategory(category)
			}
			if wrapper.Findings[i].Verdict == "" {
				wrapper.Findings[i].Verdict = "FAIL"
			}
		}
		return wrapper.Findings
	}

	// Strategy 3: Parse free-form text line by line
	lines := strings.Split(rawOutput, "\n")
	currentFile := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for file references
		if matches := p.filePattern.FindStringSubmatch(line); len(matches) > 1 {
			currentFile = matches[1]
		}

		// Check for finding patterns (numbered lists, bullet points)
		if matches := p.findingPattern.FindStringSubmatch(line); len(matches) > 1 {
			title := strings.TrimSpace(matches[1])
			if len(title) < 10 {
				continue // too short to be a real finding
			}

			file := currentFile
			if file == "" {
				file = p.extractFileFromText(title)
			}

			severity := "medium"
			if sevMatches := p.severityPattern.FindStringSubmatch(line); len(sevMatches) > 1 {
				severity = strings.ToLower(sevMatches[1])
			}

			findings = append(findings, Finding{
				File:     file,
				Category: RemediationCategory(category),
				Severity: severity,
				Verdict:  "FAIL",
				Title:    truncateStr(title, 200),
			})
		}
	}

	return findings
}

func (p *AIParser) extractFileFromText(text string) string {
	if matches := p.filePattern.FindStringSubmatch(text); len(matches) > 1 {
		return matches[1]
	}
	// Fallback: look for Go-style paths
	goPath := regexp.MustCompile(`\b([a-zA-Z0-9_/]+\.go)\b`)
	if matches := goPath.FindStringSubmatch(text); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func mapVerdictToSeverity(verdict string) string {
	switch verdict {
	case "FAIL":
		return "high"
	case "WARN":
		return "medium"
	case "PASS":
		return "low"
	default:
		return "medium"
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
