package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Mindburn-Labs/helm/core/pkg/conform"
	"github.com/Mindburn-Labs/helm/core/pkg/conform/gates"
	"github.com/Mindburn-Labs/helm/core/pkg/crypto"
)

// runConform implements `helm conform` per §2.1.
//
// Exit codes:
//
//	0 = all gates pass
//	1 = any gate failed
//	2 = runtime error
func runConform(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("conform", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var (
		profile      string
		jurisdiction string
		outputDir    string
		jsonOutput   bool
		signed       bool
		gateFilter   multiFlag
	)

	cmd.StringVar(&profile, "profile", "", "Conformance profile (REQUIRED): SMB, CORE, ENTERPRISE, REGULATED_FINANCE, REGULATED_HEALTH, AGENTIC_WEB_ROUTER")
	cmd.StringVar(&jurisdiction, "jurisdiction", "", "Jurisdiction code (e.g. US, EU, APAC)")
	cmd.StringVar(&outputDir, "output", "", "Output directory for EvidencePack (default: artifacts/conformance)")
	cmd.BoolVar(&jsonOutput, "json", false, "Output report as JSON to stdout")
	cmd.BoolVar(&signed, "signed", false, "Cryptographically sign the conformance report")
	cmd.Var(&gateFilter, "gate", "Run only specific gate(s) (repeatable)")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if profile == "" {
		_, _ = fmt.Fprintln(stderr, "Error: --profile is required")
		_, _ = fmt.Fprintln(stderr, "Valid profiles: SMB, CORE, ENTERPRISE, REGULATED_FINANCE, REGULATED_HEALTH, AGENTIC_WEB_ROUTER")
		return 2
	}

	// Validate profile
	profileID := conform.ProfileID(profile)
	if conform.GatesForProfile(profileID) == nil && len(gateFilter) == 0 {
		_, _ = fmt.Fprintf(stderr, "Error: unknown profile %q\n", profile)
		return 2
	}

	// Resolve project root
	projectRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: cannot determine working directory: %v\n", err)
		return 2
	}

	// Build engine with all gates
	engine := gates.DefaultEngine()

	// Run conformance
	opts := &conform.RunOptions{
		Profile:      profileID,
		Jurisdiction: jurisdiction,
		GateFilter:   []string(gateFilter),
		ProjectRoot:  projectRoot,
		OutputDir:    outputDir,
	}

	report, err := engine.Run(opts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: conformance run failed: %v\n", err)
		return 2
	}

	// Sign report if requested (P0.3)
	if signed {
		// Use an ephemeral signer for OSS release
		signer, _ := crypto.NewEd25519Signer("ephemeral-conform-key")
		dateStr := report.Timestamp.Format("2006-01-02")
		evidenceDir := filepath.Join(projectRoot, "artifacts", "conformance", dateStr, report.RunID)
		if outputDir != "" {
			evidenceDir = filepath.Join(outputDir, dateStr, report.RunID)
		}

		err := engine.SignReport(evidenceDir, func(data []byte) (string, error) {
			return signer.Sign(data)
		})
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Error: failed to sign report: %v\n", err)
			return 2
		}
		if !jsonOutput {
			_, _ = fmt.Fprintf(stdout, "✅ Signed conformance report in %s\n\n", evidenceDir)
		}
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(report, "", "  ")
		_, _ = fmt.Fprintln(stdout, string(data))
	} else {
		printConformanceReport(stdout, report)
	}

	if !report.Pass {
		return 1
	}
	return 0
}

func printConformanceReport(w io.Writer, report *conform.ConformanceReport) {
	_, _ = fmt.Fprintf(w, "HELM Conformance Report\n")
	_, _ = fmt.Fprintf(w, "───────────────────────\n")
	_, _ = fmt.Fprintf(w, "Run ID:    %s\n", report.RunID)
	_, _ = fmt.Fprintf(w, "Profile:   %s\n", report.Profile)
	_, _ = fmt.Fprintf(w, "Timestamp: %s\n", report.Timestamp.Format("2006-01-02T15:04:05Z"))
	_, _ = fmt.Fprintf(w, "Duration:  %s\n\n", report.Duration)

	for _, gr := range report.GateResults {
		status := "✅ PASS"
		if !gr.Pass {
			status = "❌ FAIL"
		}
		_, _ = fmt.Fprintf(w, "  %s  %s", status, gr.GateID)
		if len(gr.Reasons) > 0 {
			_, _ = fmt.Fprintf(w, "  [%s]", gr.Reasons[0])
			if len(gr.Reasons) > 1 {
				_, _ = fmt.Fprintf(w, " (+%d more)", len(gr.Reasons)-1)
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w)
	if report.Pass {
		_, _ = fmt.Fprintf(w, "Result: ✅ PASS (%d gates)\n", len(report.GateResults))
	} else {
		failed := 0
		for _, gr := range report.GateResults {
			if !gr.Pass {
				failed++
			}
		}
		_, _ = fmt.Fprintf(w, "Result: ❌ FAIL (%d/%d gates failed)\n", failed, len(report.GateResults))
	}
}

// multiFlag allows repeatable flag values (e.g. --gate G0 --gate G1).
type multiFlag []string

func (f *multiFlag) String() string { return fmt.Sprintf("%v", *f) }
func (f *multiFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
