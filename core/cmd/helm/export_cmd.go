package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// runExportCmd implements `helm export` per §2.1.
//
// Exports selected EvidencePack sections for audit or incident response.
//
// Exit codes:
//
//	0 = export completed
//	2 = runtime error
func runExportCmd(args []string, stdout, stderr io.Writer) int {
	// Support `helm export pack` subcommand (Gap #21 DoD)
	if len(args) > 0 && args[0] == "pack" {
		return handlePackCreate(args[1:])
	}

	cmd := flag.NewFlagSet("export", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var (
		evidenceDir string
		outDir      string
		audit       bool
		incident    string
		jsonOutput  bool
		tarOutput   bool
	)

	cmd.StringVar(&evidenceDir, "evidence", "", "Path to EvidencePack directory (REQUIRED)")
	cmd.StringVar(&outDir, "out", "", "Output directory (REQUIRED)")
	cmd.BoolVar(&audit, "audit", false, "Export full audit bundle")
	cmd.StringVar(&incident, "incident", "", "Export incident-related evidence for given incident ID")
	cmd.BoolVar(&jsonOutput, "json", false, "Output manifest as JSON")
	cmd.BoolVar(&tarOutput, "tar", false, "Export as deterministic .tar (sorted, epoch mtime, root uid)")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if evidenceDir == "" || outDir == "" {
		_, _ = fmt.Fprintln(stderr, "Error: --evidence and --out are required")
		return 2
	}

	if !audit && incident == "" {
		_, _ = fmt.Fprintln(stderr, "Error: specify --audit or --incident <id>")
		return 2
	}

	// Determine which subdirs to export
	var exportDirs []string
	if audit {
		// Full audit export
		exportDirs = []string{
			"00_INDEX.json",
			"01_SCORE.json",
			"02_PROOFGRAPH",
			"03_TELEMETRY",
			"05_DIFFS",
			"06_LOGS",
			"07_ATTESTATIONS",
			"08_TAPES",
			"09_SCHEMAS",
			"12_REPORTS",
		}
	} else if incident != "" {
		// Incident-focused export
		exportDirs = []string{
			"00_INDEX.json",
			"01_SCORE.json",
			"02_PROOFGRAPH",
			"06_LOGS",
			"07_ATTESTATIONS",
		}
	}

	// Create output directory
	if err := os.MkdirAll(outDir, 0750); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: cannot create output directory: %v\n", err)
		return 2
	}

	// Copy selected items
	var exported []string
	for _, item := range exportDirs {
		src := filepath.Join(evidenceDir, item)
		dst := filepath.Join(outDir, item)

		info, err := os.Stat(src)
		if err != nil {
			continue // skip non-existent items
		}

		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				_, _ = fmt.Fprintf(stderr, "Warning: failed to copy %s: %v\n", item, err)
				continue
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				_, _ = fmt.Fprintf(stderr, "Warning: failed to copy %s: %v\n", item, err)
				continue
			}
		}
		exported = append(exported, item)
	}

	result := map[string]any{
		"evidence_dir": evidenceDir,
		"output_dir":   outDir,
		"exported":     exported,
		"count":        len(exported),
	}
	if incident != "" {
		result["incident_id"] = incident
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(result, "", "  ")
		_, _ = fmt.Fprintln(stdout, string(data))
	} else {
		_, _ = fmt.Fprintf(stdout, "Exported %d items to %s\n", len(exported), outDir)
		for _, item := range exported {
			_, _ = fmt.Fprintf(stdout, "  ✅ %s\n", item)
		}
	}

	// Deterministic .tar export (Phase 9: identical hashes for identical content)
	if tarOutput {
		tarPath := outDir + ".tar"
		if err := deterministicTarGz(outDir, tarPath); err != nil {
			_, _ = fmt.Fprintf(stderr, "Error creating .tar: %v\n", err)
			return 2
		}
		if jsonOutput {
			result["tar_path"] = tarPath
		} else {
			_, _ = fmt.Fprintf(stdout, "  📦 Deterministic .tar: %s\n", tarPath)
		}
	}

	return 0
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0750); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0750)
		}
		return copyFile(path, dstPath)
	})
}

// deterministicTarGz creates a byte-identical .tar archive from a directory.
// Invariants:
//   - Paths sorted lexicographically
//   - Mtime fixed to Unix epoch (1970-01-01T00:00:00Z)
//   - Uid/Gid = 0/0, Uname/Gname = "root"/"root"
//   - Files: 0644, Dirs: 0755
//
// Two exports of identical content produce identical SHA-256.
func deterministicTarGz(srcDir, dstPath string) error {
	// 1. Collect all paths, sorted
	var paths []string
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(srcDir, path)
		if relPath == "." {
			return nil
		}
		paths = append(paths, relPath)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk failed: %w", err)
	}
	sort.Strings(paths)

	// 2. Create .tar archive
	f, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create .tar failed: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	gw.ModTime = time.Unix(0, 0) // Deterministic gzip header

	tw := tar.NewWriter(gw)
	defer tw.Close()

	epoch := time.Unix(0, 0)

	for _, relPath := range paths {
		fullPath := filepath.Join(srcDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name:    relPath,
			ModTime: epoch,
			Uname:   "root",
			Gname:   "root",
			Uid:     0,
			Gid:     0,
		}

		if info.IsDir() {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0755
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			hdr.Typeflag = tar.TypeReg
			hdr.Mode = 0644
			hdr.Size = info.Size()
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
	}

	return nil
}
