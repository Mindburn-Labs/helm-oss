package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

const orgDNASchemaPath = "schemas/orgdna.schema.json"

// runOrgDNACmd implements `helm orgdna <validate|hash>`.
func runOrgDNACmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Usage: helm orgdna <validate|hash>")
		return 2
	}

	switch args[0] {
	case "validate":
		return runOrgDNAValidate(args[1:], stdout, stderr)
	case "hash":
		return runOrgDNAHash(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "Unknown orgdna command: %s\n", args[0])
		return 2
	}
}

// runOrgDNAValidate validates an OrgDNA pack against the schema.
func runOrgDNAValidate(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("orgdna validate", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var packPath string
	var jsonOutput bool

	cmd.StringVar(&packPath, "pack", "", "Path to OrgDNA pack JSON (REQUIRED)")
	cmd.BoolVar(&jsonOutput, "json", false, "Output as JSON")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if packPath == "" {
		_, _ = fmt.Fprintln(stderr, "Error: --pack is required")
		return 2
	}

	data, err := os.ReadFile(packPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: cannot read pack: %v\n", err)
		return 2
	}

	var pack orgDNAPack
	if err := json.Unmarshal(data, &pack); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: invalid JSON: %v\n", err)
		return 1
	}

	// Validate required fields
	issues := validateOrgDNA(&pack)

	result := map[string]any{
		"valid":    len(issues) == 0,
		"org_id":   pack.OrgID,
		"pack_id":  pack.PackID,
		"version":  pack.SchemaVersion,
		"policies": len(pack.Policies),
		"issues":   issues,
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(result, "", "  ")
		_, _ = fmt.Fprintln(stdout, string(out))
	} else {
		if len(issues) == 0 {
			_, _ = fmt.Fprintf(stdout, "✅ Valid OrgDNA pack: %s/%s\n", pack.OrgID, pack.PackID)
			_, _ = fmt.Fprintf(stdout, "   Schema file: %s\n", orgDNASchemaPath)
			_, _ = fmt.Fprintf(stdout, "   Schema version: %s\n", pack.SchemaVersion)
			_, _ = fmt.Fprintf(stdout, "   Policies: %d\n", len(pack.Policies))
			if pack.RiskThresholds != nil {
				_, _ = fmt.Fprintf(stdout, "   Risk thresholds: configured\n")
			}
			_, _ = fmt.Fprintf(stdout, "   Approval chains: %d\n", len(pack.ApprovalChains))
		} else {
			_, _ = fmt.Fprintf(stdout, "❌ Invalid OrgDNA pack: %s\n", packPath)
			for _, issue := range issues {
				_, _ = fmt.Fprintf(stdout, "   - %s\n", issue)
			}
		}
	}

	if len(issues) > 0 {
		return 1
	}
	return 0
}

// runOrgDNAHash computes the content-addressed hash of an OrgDNA pack.
func runOrgDNAHash(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("orgdna hash", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var packPath string
	cmd.StringVar(&packPath, "pack", "", "Path to OrgDNA pack JSON (REQUIRED)")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if packPath == "" {
		_, _ = fmt.Fprintln(stderr, "Error: --pack is required")
		return 2
	}

	data, err := os.ReadFile(packPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: cannot read pack: %v\n", err)
		return 2
	}

	// Normalize JSON for deterministic hashing
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: invalid JSON: %v\n", err)
		return 1
	}

	canonical, _ := json.Marshal(raw)
	hash := sha256.Sum256(canonical)
	hashHex := hex.EncodeToString(hash[:])

	_, _ = fmt.Fprintf(stdout, "sha256:%s\n", hashHex)
	return 0
}

// --- Structs ---

type orgDNAPack struct {
	SchemaVersion  string           `json:"schema_version"`
	OrgID          string           `json:"org_id"`
	PackID         string           `json:"pack_id"`
	Description    string           `json:"description"`
	Policies       []orgDNAPolicy   `json:"policies"`
	RiskThresholds *json.RawMessage `json:"risk_thresholds,omitempty"`
	ApprovalChains []orgDNAApproval `json:"approval_chains"`
}

type orgDNAPolicy struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Action string `json:"action"`
}

type orgDNAApproval struct {
	ActionPattern string   `json:"action_pattern"`
	Chain         []string `json:"chain"`
}

// validateOrgDNA checks required fields.
func validateOrgDNA(pack *orgDNAPack) []string {
	var issues []string

	if pack.SchemaVersion == "" {
		issues = append(issues, "missing schema_version")
	}
	if pack.OrgID == "" {
		issues = append(issues, "missing org_id")
	}
	if pack.PackID == "" {
		issues = append(issues, "missing pack_id")
	}
	if len(pack.Policies) == 0 {
		issues = append(issues, "no policies defined")
	}

	for i, p := range pack.Policies {
		if p.ID == "" {
			issues = append(issues, fmt.Sprintf("policy[%d]: missing id", i))
		}
		if p.Name == "" {
			issues = append(issues, fmt.Sprintf("policy[%d]: missing name", i))
		}
		if p.Action == "" {
			issues = append(issues, fmt.Sprintf("policy[%d]: missing action", i))
		}
	}

	return issues
}
