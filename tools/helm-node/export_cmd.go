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
// Supports deterministic .tar packaging per UCS Appendix A.3.
func runExportCmd(args []string, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("export", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var (
		evidenceDir string
		outPath     string
		audit       bool
		incident    string
		jsonOutput  bool
		tarball     bool
	)

	cmd.StringVar(&evidenceDir, "evidence", "", "Path to EvidencePack directory (REQUIRED)")
	cmd.StringVar(&outPath, "out", "", "Output directory or file path (REQUIRED)")
	cmd.BoolVar(&audit, "audit", false, "Export full audit bundle")
	cmd.StringVar(&incident, "incident", "", "Export incident-related evidence for given incident ID")
	cmd.BoolVar(&jsonOutput, "json", false, "Output manifest as JSON")
	cmd.BoolVar(&tarball, "tar", false, "Export as deterministic .tar")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if evidenceDir == "" || outPath == "" {
		_, _ = fmt.Fprintln(stderr, "Error: --evidence and --out are required")
		return 2
	}

	if !audit && incident == "" {
		_, _ = fmt.Fprintln(stderr, "Error: specify --audit or --incident <id>")
		return 2
	}

	// Determine which items to export
	exportItems := []string{}
	if audit {
		exportItems = []string{
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
		exportItems = []string{
			"00_INDEX.json",
			"01_SCORE.json",
			"02_PROOFGRAPH",
			"06_LOGS",
			"07_ATTESTATIONS",
		}
	}

	if tarball {
		if err := exportTarball(evidenceDir, outPath, exportItems); err != nil {
			_, _ = fmt.Fprintf(stderr, "Error: failed to create tarball: %v\n", err)
			return 2
		}
		_, _ = fmt.Fprintf(stdout, "Exported deterministic tarball to %s\n", outPath)
		return 0
	}

	// Create output directory
	if err := os.MkdirAll(outPath, 0750); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: cannot create output directory: %v\n", err)
		return 2
	}

	// Copy selected items
	var exported []string
	for _, item := range exportItems {
		src := filepath.Join(evidenceDir, item)
		dst := filepath.Join(outPath, item)

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
		"output_dir":   outPath,
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
		_, _ = fmt.Fprintf(stdout, "Exported %d items to %s\n", len(exported), outPath)
		for _, item := range exported {
			_, _ = fmt.Fprintf(stdout, "  ✅ %s\n", item)
		}
	}

	return 0
}

// exportTarball creates a deterministic .tar archive of the selected items.
// Per UCS Appendix A.3.
func exportTarball(srcDir, dstFile string, items []string) error {
	f, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Gzip with no timestamp for determinism
	gw, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	gw.Name = ""
	gw.ModTime = time.Time{}
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Sort items for determinism
	sort.Strings(items)

	for _, item := range items {
		if err := addFileToTar(tw, srcDir, item); err != nil {
			return err
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, baseDir, relPath string) error {
	src := filepath.Join(baseDir, relPath)
	info, err := os.Stat(src)
	if err != nil {
		return nil // skip missing
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	// Normalize header for determinism (UCS Appendix A.3)
	header.Name = filepath.ToSlash(relPath)
	header.Uid = 0
	header.Gid = 0
	header.Uname = ""
	header.Gname = ""
	header.ModTime = time.Time{} // No timestamp
	header.AccessTime = time.Time{}
	header.ChangeTime = time.Time{}

	if info.IsDir() {
		header.Name += "/"
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// Recurse and sort children
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)

		for _, name := range names {
			if err := addFileToTar(tw, baseDir, filepath.Join(relPath, name)); err != nil {
				return err
			}
		}
		return nil
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
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
