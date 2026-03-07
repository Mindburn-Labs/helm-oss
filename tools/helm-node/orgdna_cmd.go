package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Mindburn-Labs/helm/core/pkg/canonicalize"
)

const orgDNASchemaPath = "schemas/orgdna.schema.json"

// runOrgDNA implements `helm orgdna` per §3.
func runOrgDNA(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		printOrgDNAUsage(stderr)
		return 2
	}

	switch args[0] {
	case "validate":
		return runOrgDNAValidate(args[1:], stdout, stderr)
	case "hash":
		return runOrgDNAHash(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown orgdna command: %s\n", args[0])
		printOrgDNAUsage(stderr)
		return 2
	}
}

func printOrgDNAUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: helm-node orgdna <command> [arguments]")
	fmt.Fprintln(w, "\nCommands:")
	fmt.Fprintf(w, "  validate   Validate an OrgDNA file against %s\n", orgDNASchemaPath)
	fmt.Fprintln(w, "  hash       Compute the canonical JCS hash of an OrgDNA file")
}

func runOrgDNAValidate(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("orgdna validate", flag.ContinueOnError)
	var path string
	cmd.StringVar(&path, "path", "", "Path to OrgDNA file (REQUIRED)")
	if err := cmd.Parse(args); err != nil {
		return 2
	}
	if path == "" {
		fmt.Fprintln(stderr, "Error: --path is required")
		return 2
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading file: %v\n", err)
		return 2
	}

	var dna map[string]any
	if err := json.Unmarshal(data, &dna); err != nil {
		fmt.Fprintf(stderr, "Error parsing JSON: %v\n", err)
		return 2
	}

	// Baseline validation (P3)
	required := []string{"version", "entity_id", "purpose"}
	for _, field := range required {
		if _, ok := dna[field]; !ok {
			fmt.Fprintf(stdout, "❌ Validation FAILED: missing required field %q\n", field)
			return 1
		}
	}

	fmt.Fprintln(stdout, "✅ OrgDNA validation PASSED (Seed Schema)")
	return 0
}

func runOrgDNAHash(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("orgdna hash", flag.ContinueOnError)
	var path string
	cmd.StringVar(&path, "path", "", "Path to OrgDNA file (REQUIRED)")
	if err := cmd.Parse(args); err != nil {
		return 2
	}
	if path == "" {
		fmt.Fprintln(stderr, "Error: --path is required")
		return 2
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading file: %v\n", err)
		return 2
	}

	var dna map[string]any
	if err := json.Unmarshal(data, &dna); err != nil {
		fmt.Fprintf(stderr, "Error parsing JSON: %v\n", err)
		return 2
	}

	hash, err := canonicalize.CanonicalHash(dna)
	if err != nil {
		fmt.Fprintf(stderr, "Error hashing: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "sha256:%s\n", hash)
	return 0
}
