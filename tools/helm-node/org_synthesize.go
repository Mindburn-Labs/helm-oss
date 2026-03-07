package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/canonicalize"
	"github.com/Mindburn-Labs/helm/core/pkg/contracts"
)

// runOrgSynthesize implements a baseline Verified Genesis Loop (VGL) per §4.
// It compiles a minimal OrgGenome and renders a Deterministic Semantic Mirror.
func runOrgSynthesize(args []string, stdout io.Writer) int {
	cmd := flag.NewFlagSet("synthesize", flag.ContinueOnError)
	var (
		orgName string
		outPath string
	)
	cmd.StringVar(&orgName, "name", "Default Org", "Organization name")
	cmd.StringVar(&outPath, "out", "org_genome.json", "Output path for compiled OrgGenome")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	// 1. Compile OrgGenome (Baseline)
	genome := &contracts.OrgGenome{
		Meta: contracts.GenomeMeta{
			GenomeID:  "gen-001",
			Name:      orgName,
			CreatedAt: time.Now().UTC(),
		},
		Phenotype: contracts.PhenotypeMeta{
			PhenotypeID:     "phen-001",
			SpecVersion:     "1.2",
			CompilerVersion: "helm-oss-0.1.0",
			CompiledAt:      time.Now().UTC(),
		},
		Regulation: map[string]any{
			"max_daily_spend_usd": 100,
			"risk_class_max":      "T2",
		},
	}

	// 2. Compute Canonical Hash
	genomeHash, err := canonicalize.CanonicalHash(genome)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "Error computing genome hash: %v\n", err)
		return 2
	}
	genome.Phenotype.CanonicalHash = "sha256:" + genomeHash

	// 3. Render Deterministic Semantic Mirror (UCS §4.2)
	_, _ = fmt.Fprintln(stdout, "--- DETERMINISTIC SEMANTIC MIRROR ---")
	_, _ = fmt.Fprintf(stdout, "Organization: %s\n", genome.Meta.Name)
	_, _ = fmt.Fprintf(stdout, "Genome ID:    %s\n", genome.Meta.GenomeID)
	_, _ = fmt.Fprintf(stdout, "Hash:         %s\n", genome.Phenotype.CanonicalHash)
	_, _ = fmt.Fprintln(stdout, "\nPOLICY SUMMARY:")
	_, _ = fmt.Fprintln(stdout, "  - Max Daily Spend: $100.00")
	_, _ = fmt.Fprintln(stdout, "  - Max Risk Class:  T2 (Medium)")
	_, _ = fmt.Fprintln(stdout, "\nBY SIGNING THIS, YOU ACTIVATE DETERMINISTIC LAW FOR THIS ORG.")
	_, _ = fmt.Fprintln(stdout, "--------------------------------------")

	// 4. Write to file
	data, _ := json.MarshalIndent(genome, "", "  ")
	if err := os.WriteFile(outPath, data, 0600); err != nil {
		_, _ = fmt.Fprintf(stdout, "Error writing genome: %v\n", err)
		return 2
	}

	_, _ = fmt.Fprintf(stdout, "\n✅ Compiled OrgGenome written to %s\n", outPath)
	return 0
}