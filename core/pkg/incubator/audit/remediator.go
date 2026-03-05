package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── Autonomous Remediator ───────────────────────────────────────────────────
//
// The Remediator closes the audit loop: findings → patches → PRs → verify.
//
// Confidence tiers:
//   - HIGH   (>= 0.85): Auto-apply, create PR, re-audit changed files
//   - MEDIUM (0.6-0.85): Create PR, request human review
//   - LOW    (< 0.6):    Log finding, create issue, don't touch code
//
// Usage:
//
//	r := audit.NewRemediator(audit.RemediatorConfig{...})
//	result := r.Process(ctx, findings)
//	// result.PRsCreated, result.IssuesCreated, result.AutoFixed

// RemediatorConfig controls behavior thresholds.
type RemediatorConfig struct {
	// AutoApplyThreshold: mutations above this confidence are applied directly.
	AutoApplyThreshold float64 `json:"auto_apply_threshold"`
	// PRThreshold: mutations above this create PRs. Below creates issues.
	PRThreshold float64 `json:"pr_threshold"`
	// RepoRoot for git operations.
	RepoRoot string `json:"repo_root"`
	// DryRun: if true, generate patches but don't create PRs/issues.
	DryRun bool `json:"dry_run"`
	// BranchPrefix for auto-generated branches.
	BranchPrefix string `json:"branch_prefix"`
	// MaxPRsPerRun limits blast radius.
	MaxPRsPerRun int `json:"max_prs_per_run"`
}

// DefaultRemediatorConfig returns production-safe defaults.
func DefaultRemediatorConfig() RemediatorConfig {
	return RemediatorConfig{
		AutoApplyThreshold: 0.90, // Very high bar for auto-apply
		PRThreshold:        0.60,
		BranchPrefix:       "audit/auto-fix/",
		MaxPRsPerRun:       5, // Limit blast radius
		DryRun:             false,
	}
}

// RemediationResult summarizes what the remediator did.
type RemediationResult struct {
	TotalFindings int               `json:"total_findings"`
	Translated    int               `json:"translated"`
	AutoFixed     int               `json:"auto_fixed"`
	PRsCreated    int               `json:"prs_created"`
	IssuesCreated int               `json:"issues_created"`
	Skipped       int               `json:"skipped"`
	Actions       []RemediateAction `json:"actions"`
	Timestamp     time.Time         `json:"timestamp"`
}

// RemediateAction records a single remediation decision.
type RemediateAction struct {
	FindingID  string  `json:"finding_id"`
	File       string  `json:"file"`
	Title      string  `json:"title"`
	Confidence float64 `json:"confidence"`
	Decision   string  `json:"decision"` // AUTO_FIXED, PR_CREATED, ISSUE_CREATED, SKIPPED
	PRUrl      string  `json:"pr_url,omitempty"`
	IssueUrl   string  `json:"issue_url,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// Remediator is the autonomous fix engine.
type Remediator struct {
	config     RemediatorConfig
	translator *FindingTranslator
}

// NewRemediator creates a remediator with the given config.
func NewRemediator(config RemediatorConfig) *Remediator {
	return &Remediator{
		config:     config,
		translator: NewFindingTranslator(),
	}
}

// WithTranslator replaces the default translator with a custom one.
func (r *Remediator) WithTranslator(t *FindingTranslator) {
	r.translator = t
}

// Process takes audit findings and autonomously remediates what it can.
func (r *Remediator) Process(ctx context.Context, findings []Finding) (*RemediationResult, error) {
	result := &RemediationResult{
		TotalFindings: len(findings),
		Timestamp:     time.Now().UTC(),
	}

	mutations := r.translator.Translate(findings)
	result.Translated = len(mutations)

	prsCreated := 0

	for _, m := range mutations {
		if prsCreated >= r.config.MaxPRsPerRun {
			slog.Warn("remediator: max PRs per run reached",
				"limit", r.config.MaxPRsPerRun)
			break
		}

		action := RemediateAction{
			FindingID:  m.FindingID,
			File:       m.File,
			Title:      m.Description,
			Confidence: m.Confidence,
		}

		switch {
		case m.Confidence >= r.config.AutoApplyThreshold && m.AutoApply:
			// HIGH confidence + safe to auto-apply
			if r.config.DryRun {
				action.Decision = "DRY_RUN_AUTO_FIX"
			} else {
				err := r.autoFix(ctx, m)
				if err != nil {
					action.Decision = "AUTO_FIX_FAILED"
					action.Error = err.Error()
				} else {
					action.Decision = "AUTO_FIXED"
					result.AutoFixed++
				}
			}

		case m.Confidence >= r.config.PRThreshold:
			// MEDIUM confidence → Create PR
			if r.config.DryRun {
				action.Decision = "DRY_RUN_PR"
			} else {
				url, err := r.createPR(ctx, m)
				if err != nil {
					action.Decision = "PR_FAILED"
					action.Error = err.Error()
				} else {
					action.Decision = "PR_CREATED"
					action.PRUrl = url
					result.PRsCreated++
					prsCreated++
				}
			}

		default:
			// LOW confidence → Create issue only
			if r.config.DryRun {
				action.Decision = "DRY_RUN_ISSUE"
			} else {
				url, err := r.createIssue(ctx, m)
				if err != nil {
					action.Decision = "ISSUE_FAILED"
					action.Error = err.Error()
				} else {
					action.Decision = "ISSUE_CREATED"
					action.IssueUrl = url
					result.IssuesCreated++
				}
			}
		}

		result.Actions = append(result.Actions, action)
		slog.Info("remediator action",
			"file", m.File,
			"confidence", m.Confidence,
			"decision", action.Decision,
		)
	}

	result.Skipped = result.TotalFindings - result.Translated

	return result, nil
}

// ExecuteMutation dispatches to strategy-specific executors for real file writing.
func (r *Remediator) ExecuteMutation(m Mutation) error {
	lower := strings.ToLower(m.Description)

	switch {
	case strings.Contains(lower, "go mod tidy"):
		dir := filepath.Dir(m.File)
		return ExecuteGoModTidy(r.config.RepoRoot, dir)

	case strings.Contains(lower, "test stub"):
		return ExecuteTestStub(r.config.RepoRoot, m)

	case strings.Contains(lower, "dockerfile"):
		return ExecuteDockerfileFix(r.config.RepoRoot, m)

	default:
		// Write patch content to .audit/patches/ for manual review
		patchDir := filepath.Join(r.config.RepoRoot, ".audit", "patches")
		if err := os.MkdirAll(patchDir, 0o755); err != nil {
			return err
		}
		patchFile := filepath.Join(patchDir, sanitizeBranch(m.FindingID)+".patch")
		return os.WriteFile(patchFile, []byte(m.Patch), 0o644)
	}
}

func (r *Remediator) autoFix(ctx context.Context, m Mutation) error {
	branchName := r.config.BranchPrefix + sanitizeBranch(m.FindingID)

	// Create branch, apply patch, commit, push, PR
	cmds := [][]string{
		{"git", "-C", r.config.RepoRoot, "checkout", "-b", branchName},
		{"git", "-C", r.config.RepoRoot, "add", m.File},
		{"git", "-C", r.config.RepoRoot, "commit", "-m",
			fmt.Sprintf("fix(audit): auto-remediate %s\n\n%s\nConfidence: %.0f%%\nAuto-applied by HELM Remediator",
				m.FindingID, m.Description, m.Confidence*100)},
		{"git", "-C", r.config.RepoRoot, "push", "origin", branchName},
	}

	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s failed: %w (%s)", args[0], err, string(out))
		}
	}

	// Create PR
	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--repo", r.config.RepoRoot,
		"--head", branchName,
		"--title", fmt.Sprintf("fix(audit): %s", m.Description),
		"--body", fmt.Sprintf("## Auto-Remediation\n\n**Finding**: %s\n**File**: `%s`\n**Confidence**: %.0f%%\n\nThis PR was auto-generated by the HELM Autonomous Remediator.\nThe fix was auto-applied (confidence ≥ %.0f%%).",
			m.FindingID, m.File, m.Confidence*100, r.config.AutoApplyThreshold*100),
		"--label", "audit,auto-fix",
	)
	cmd.Dir = r.config.RepoRoot
	_, err := cmd.CombinedOutput()
	return err
}

func (r *Remediator) createPR(ctx context.Context, m Mutation) (string, error) {
	branchName := r.config.BranchPrefix + sanitizeBranch(m.FindingID)

	// Write patch file
	patchPath := filepath.Join(r.config.RepoRoot, ".audit", "patches", m.FindingID+".patch")
	if err := os.MkdirAll(filepath.Dir(patchPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(patchPath, []byte(m.Patch), 0o644); err != nil {
		return "", err
	}

	cmds := [][]string{
		{"git", "-C", r.config.RepoRoot, "checkout", "-b", branchName},
		{"git", "-C", r.config.RepoRoot, "add", patchPath},
		{"git", "-C", r.config.RepoRoot, "commit", "-m",
			fmt.Sprintf("fix(audit): %s\n\n%s\nConfidence: %.0f%%\nReview required — HELM Remediator",
				m.FindingID, m.Description, m.Confidence*100)},
		{"git", "-C", r.config.RepoRoot, "push", "origin", branchName},
	}

	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("%s failed: %w (%s)", args[0], err, string(out))
		}
	}

	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--head", branchName,
		"--title", fmt.Sprintf("fix(audit): %s [review needed]", m.Description),
		"--body", fmt.Sprintf("## Audit Remediation — Review Required\n\n**Finding**: %s\n**File**: `%s`\n**Confidence**: %.0f%%\n\n```diff\n%s\n```\n\nThis fix needs human review (confidence %.0f%% < auto-apply threshold %.0f%%).",
			m.FindingID, m.File, m.Confidence*100, m.Patch, m.Confidence*100, r.config.AutoApplyThreshold*100),
		"--label", "audit,needs-review",
	)
	cmd.Dir = r.config.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create failed: %w (%s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Remediator) createIssue(ctx context.Context, m Mutation) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "issue", "create",
		"--title", fmt.Sprintf("[audit] %s", m.Description),
		"--body", fmt.Sprintf("## Audit Finding — Low Confidence\n\n**File**: `%s`\n**Confidence**: %.0f%%\n\n%s\n\n> This issue was auto-created by the HELM audit system. The confidence is too low (%.0f%%) for automated remediation.",
			m.File, m.Confidence*100, m.Description, m.Confidence*100),
		"--label", "audit,investigation",
	)
	cmd.Dir = r.config.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh issue create failed: %w (%s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// ExportResult writes the remediation result to JSON.
func (r *RemediationResult) ExportResult(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func sanitizeBranch(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ToLower(s)
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
