package gates

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/conform"
)

// G5AA2A validates inter-agent zero trust per §G5A.
type G5AA2A struct{}

func (g *G5AA2A) ID() string   { return "G5A" }
func (g *G5AA2A) Name() string { return "Inter-Agent Zero Trust (A2A)" }

func (g *G5AA2A) Run(ctx *conform.RunContext) *conform.GateResult {
	result := &conform.GateResult{
		GateID:        g.ID(),
		Pass:          true,
		Reasons:       []string{},
		EvidencePaths: []string{},
		Metrics:       conform.GateMetrics{Counts: make(map[string]int)},
	}

	a2aDir := filepath.Join(ctx.EvidenceDir, "10_A2A")

	// 1. Check sessions
	sessionsDir := filepath.Join(a2aDir, "sessions")
	if dirExists(sessionsDir) {
		files, _ := filepath.Glob(filepath.Join(sessionsDir, "*.json"))
		result.Metrics.Counts["sessions"] = len(files)
		result.EvidencePaths = append(result.EvidencePaths, "10_A2A/sessions/")
	}

	// 2. Check passports
	passportsDir := filepath.Join(a2aDir, "passports")
	if dirExists(passportsDir) {
		files, _ := filepath.Glob(filepath.Join(passportsDir, "*.json"))
		result.Metrics.Counts["passports"] = len(files)
		result.EvidencePaths = append(result.EvidencePaths, "10_A2A/passports/")
	}

	// 3. Check proof capsule verification report
	reportPath := filepath.Join(a2aDir, "proof_capsules", "verify_report.json")
	if fileExists(reportPath) {
		data, _ := os.ReadFile(reportPath)
		var report map[string]any
		if json.Unmarshal(data, &report) == nil {
			if valid, ok := report["all_valid"].(bool); ok && !valid {
				result.Pass = false
				result.Reasons = append(result.Reasons, conform.ReasonA2AProofMissing)
			}
		}
		result.EvidencePaths = append(result.EvidencePaths, "10_A2A/proof_capsules/verify_report.json")
	} else {
		result.Pass = false
		result.Reasons = append(result.Reasons, conform.ReasonA2AProofMissing)
	}

	return result
}
