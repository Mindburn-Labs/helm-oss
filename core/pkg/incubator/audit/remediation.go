package audit

import (
	"fmt"
	"strings"
)

// RemediationCategory maps audit finding categories to auto-fix strategies.
type RemediationCategory string

const (
	RemediationAccessibility RemediationCategory = "accessibility"
	RemediationArchitecture  RemediationCategory = "architecture"
	RemediationSecurity      RemediationCategory = "security"
	RemediationBrandTone     RemediationCategory = "brand_tone"
	RemediationMotion        RemediationCategory = "motion_semantics"
)

// Finding represents a single audit finding that may be auto-remediable.
type Finding struct {
	File           string              `json:"file"`
	Category       RemediationCategory `json:"category"`
	Severity       string              `json:"severity"`
	Verdict        string              `json:"verdict"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Recommendation string              `json:"recommendation"`
}

// Mutation represents a code change that remediates a finding.
type Mutation struct {
	File        string  `json:"file"`
	FindingID   string  `json:"finding_id"`
	Description string  `json:"description"`
	Patch       string  `json:"patch"`      // Unified diff format
	Confidence  float64 `json:"confidence"` // 0.0-1.0
	AutoApply   bool    `json:"auto_apply"` // Safe to apply without review?
}

// FindingTranslator maps audit findings to OrgDNA mutations.
//
// It supports trivially-fixable categories where the remediation is
// deterministic and low-risk. Complex findings are flagged for human review.
//
// Usage:
//
//	translator := audit.NewFindingTranslator()
//	mutations := translator.Translate(findings)
//	for _, m := range mutations {
//	    if m.AutoApply { applyPatch(m) }
//	}
type FindingTranslator struct {
	strategies map[RemediationCategory]RemediationStrategy
}

// RemediationStrategy generates mutations for a specific category.
type RemediationStrategy interface {
	CanRemediate(finding Finding) bool
	GenerateMutation(finding Finding) (*Mutation, error)
}

// NewFindingTranslator creates a translator with built-in strategies.
func NewFindingTranslator() *FindingTranslator {
	return &FindingTranslator{
		strategies: map[RemediationCategory]RemediationStrategy{
			RemediationArchitecture:  &architectureStrategy{},
			RemediationAccessibility: &accessibilityStrategy{},
			RemediationSecurity:      &securityStrategy{},
		},
	}
}

// RegisterStrategy adds a custom remediation strategy.
func (t *FindingTranslator) RegisterStrategy(category RemediationCategory, strategy RemediationStrategy) {
	t.strategies[category] = strategy
}

// Translate converts findings to mutations where possible.
// Returns only findings that have a viable auto-remediation.
func (t *FindingTranslator) Translate(findings []Finding) []Mutation {
	var mutations []Mutation
	for i, f := range findings {
		if f.Verdict != "FAIL" {
			continue
		}
		strategy, ok := t.strategies[f.Category]
		if !ok {
			continue
		}
		if !strategy.CanRemediate(f) {
			continue
		}
		m, err := strategy.GenerateMutation(f)
		if err != nil {
			continue
		}
		m.FindingID = fmt.Sprintf("finding-%d", i)
		mutations = append(mutations, *m)
	}
	return mutations
}

// --- Built-in Strategies ---

// architectureStrategy handles "unnecessary use client" findings.
type architectureStrategy struct{}

func (s *architectureStrategy) CanRemediate(f Finding) bool {
	return strings.Contains(strings.ToLower(f.Title), "use client") ||
		strings.Contains(strings.ToLower(f.Description), "'use client'")
}

func (s *architectureStrategy) GenerateMutation(f Finding) (*Mutation, error) {
	return &Mutation{
		File:        f.File,
		Description: fmt.Sprintf("Remove unnecessary 'use client' directive from %s", f.File),
		Patch:       fmt.Sprintf("--- a/%s\n+++ b/%s\n@@ -1 +0,0 @@\n-'use client'\n", f.File, f.File),
		Confidence:  0.85,
		AutoApply:   false, // Needs review — may break client components
	}, nil
}

// accessibilityStrategy handles missing ARIA/labels.
type accessibilityStrategy struct{}

func (s *accessibilityStrategy) CanRemediate(f Finding) bool {
	lower := strings.ToLower(f.Title + " " + f.Description)
	return strings.Contains(lower, "aria") ||
		strings.Contains(lower, "label") ||
		strings.Contains(lower, "alt text")
}

func (s *accessibilityStrategy) GenerateMutation(f Finding) (*Mutation, error) {
	return &Mutation{
		File:        f.File,
		Description: fmt.Sprintf("Add accessibility attributes to %s: %s", f.File, f.Title),
		Patch:       fmt.Sprintf("// TO"+"DO: Add %s to %s", f.Recommendation, f.File),
		Confidence:  0.60, // Lower — ARIA fixes often need context
		AutoApply:   false,
	}, nil
}

// securityStrategy handles input sanitization patterns.
type securityStrategy struct{}

func (s *securityStrategy) CanRemediate(f Finding) bool {
	lower := strings.ToLower(f.Title + " " + f.Description)
	return strings.Contains(lower, "dangerouslysetinnerhtml") ||
		strings.Contains(lower, "xss") ||
		strings.Contains(lower, "sanitiz")
}

func (s *securityStrategy) GenerateMutation(f Finding) (*Mutation, error) {
	return &Mutation{
		File:        f.File,
		Description: fmt.Sprintf("Security remediation for %s: %s", f.File, f.Title),
		Patch:       fmt.Sprintf("// SECURITY: %s — requires manual review\n// Recommendation: %s", f.Title, f.Recommendation),
		Confidence:  0.40, // Low — security changes always need review
		AutoApply:   false,
	}, nil
}
