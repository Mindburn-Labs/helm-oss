package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/Mindburn-Labs/helm/core/pkg/verifier"
)

// runVerifyCmd implements `helm verify` per §2.1.
//
// Validates a signed EvidencePack bundle: structure, hashes, chain integrity,
// and replay determinism.
//
// Exit codes:
//
//	0 = verification passed
//	1 = verification failed
//	2 = runtime error
func runVerifyCmd(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("verify", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var (
		bundle     string
		jsonOutput bool
	)

	cmd.StringVar(&bundle, "bundle", "", "Path to EvidencePack directory or bundle (REQUIRED)")
	cmd.BoolVar(&jsonOutput, "json", false, "Output detailed report as JSON (auditor mode)")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if bundle == "" {
		_, _ = fmt.Fprintln(stderr, "Error: --bundle is required")
		return 2
	}

	// Use the offline verifier library (P0.2)
	report, err := verifier.VerifyBundle(bundle)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: verification failed: %v\n", err)
		return 2
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(report, "", "  ")
		_, _ = fmt.Fprintln(stdout, string(data))
	} else {
		if report.Verified {
			_, _ = fmt.Fprintf(stdout, "✅ EvidencePack verification PASSED\n")
			_, _ = fmt.Fprintf(stdout, "Summary: %s\n", report.Summary)
		} else {
			_, _ = fmt.Fprintf(stdout, "❌ EvidencePack verification FAILED\n")
			_, _ = fmt.Fprintf(stdout, "Summary: %s\n", report.Summary)
			for _, check := range report.Checks {
				if !check.Pass {
					_, _ = fmt.Fprintf(stdout, "  - [%s] %s: %s\n", check.Name, check.Reason, check.Detail)
				}
			}
		}
	}

	if !report.Verified {
		return 1
	}
	return 0
}