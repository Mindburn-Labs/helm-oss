package gates

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/conform"
)

// G10Formal validates formal verification export per §G10.
type G10Formal struct{}

func (g *G10Formal) ID() string   { return "G10" }
func (g *G10Formal) Name() string { return "Formal Verification Export" }

func (g *G10Formal) Run(ctx *conform.RunContext) *conform.GateResult {
	result := &conform.GateResult{
		GateID:        g.ID(),
		Pass:          true,
		Reasons:       []string{},
		EvidencePaths: []string{},
		Metrics:       conform.GateMetrics{Counts: make(map[string]int)},
	}

	formalDir := filepath.Join(ctx.EvidenceDir, "11_FORMAL")

	// 1. Check telemetry_events.jsonl
	telemetryPath := filepath.Join(formalDir, "telemetry_events.jsonl")
	if fileExists(telemetryPath) {
		result.EvidencePaths = append(result.EvidencePaths, "11_FORMAL/telemetry_events.jsonl")
		result.Metrics.Counts["telemetry_export"] = 1
	} else {
		result.Pass = false
		result.Reasons = append(result.Reasons, conform.ReasonFormalExportInvalid)
	}

	// 2. Check mdp_export.json
	mdpPath := filepath.Join(formalDir, "mdp_export.json")
	if fileExists(mdpPath) {
		result.EvidencePaths = append(result.EvidencePaths, "11_FORMAL/mdp_export.json")
		result.Metrics.Counts["mdp_export"] = 1
	} else {
		result.Pass = false
		result.Reasons = append(result.Reasons, conform.ReasonFormalExportInvalid)
	}

	// 3. Check properties
	propFiles, _ := filepath.Glob(filepath.Join(formalDir, "properties.*"))
	if len(propFiles) > 0 {
		result.Metrics.Counts["safety_properties"] = len(propFiles)
		result.EvidencePaths = append(result.EvidencePaths, "11_FORMAL/properties.*")
	} else {
		result.Pass = false
		result.Reasons = append(result.Reasons, conform.ReasonFormalExportInvalid)
	}

	// 4. Validate structure of MDP export
	if fileExists(mdpPath) {
		data, _ := os.ReadFile(mdpPath)
		var mdp map[string]any
		if json.Unmarshal(data, &mdp) == nil {
			if _, ok := mdp["states"]; !ok {
				result.Pass = false
				result.Reasons = append(result.Reasons, conform.ReasonFormalExportInvalid)
			}
		}
	}

	return result
}
