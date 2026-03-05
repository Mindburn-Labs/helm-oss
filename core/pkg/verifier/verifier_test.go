package verifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyBundle_Valid(t *testing.T) {
	// Create a minimal valid golden bundle
	dir := t.TempDir()

	// Create manifest
	manifest := map[string]any{
		"session_id":  "test-session-001",
		"version":     "1.0.0",
		"exported_at": "2026-01-01T00:00:00Z",
		"file_hashes": map[string]string{},
	}
	writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)

	// Create 00_INDEX.json
	index := map[string]any{
		"version": "1.0.0",
		"gates":   []string{"G0", "G1"},
	}
	writeJSON(t, filepath.Join(dir, "00_INDEX.json"), index)

	report, err := VerifyBundle(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Verified {
		t.Errorf("expected PASS, got FAIL: %s", report.Summary)
		for _, c := range report.Checks {
			if !c.Pass {
				t.Logf("  FAIL: %s — %s", c.Name, c.Reason)
			}
		}
	}
	if report.VerifierVer != VerifierVersion {
		t.Errorf("expected version %s, got %s", VerifierVersion, report.VerifierVer)
	}
}

func TestVerifyBundle_MissingManifest(t *testing.T) {
	dir := t.TempDir()

	report, err := VerifyBundle(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verified {
		t.Error("expected FAIL for missing manifest")
	}

	// Should fail on structure check
	found := false
	for _, c := range report.Checks {
		if c.Name == "structure" && !c.Pass {
			found = true
		}
	}
	if !found {
		t.Error("expected structure check to fail")
	}
}

func TestVerifyBundle_HashMismatch(t *testing.T) {
	dir := t.TempDir()

	// Write a file
	os.WriteFile(filepath.Join(dir, "receipt.json"), []byte(`{"id":"r1"}`),0o644)

	// Create manifest with wrong hash
	manifest := map[string]any{
		"session_id":  "test-session-002",
		"version":     "1.0.0",
		"exported_at": "2026-01-01T00:00:00Z",
		"file_hashes": map[string]string{
			"receipt.json": "0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
	writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)

	report, err := VerifyBundle(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verified {
		t.Error("expected FAIL for hash mismatch")
	}

	hashFailed := false
	for _, c := range report.Checks {
		if c.Name == "hash:receipt.json" && !c.Pass {
			hashFailed = true
		}
	}
	if !hashFailed {
		t.Error("expected hash check to fail for receipt.json")
	}
}

func TestVerifyBundle_ValidWithHashes(t *testing.T) {
	dir := t.TempDir()

	// Write a file and compute correct hash
	content := []byte(`{"id":"r1","type":"receipt"}`)
	os.WriteFile(filepath.Join(dir, "receipt.json"), content,0o644)
	hash := sha256Hex(content)

	manifest := map[string]any{
		"session_id":  "test-session-003",
		"version":     "1.0.0",
		"exported_at": "2026-01-01T00:00:00Z",
		"file_hashes": map[string]string{
			"receipt.json": hash,
		},
	}
	writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)

	report, err := VerifyBundle(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Verified {
		t.Errorf("expected PASS, got FAIL: %s", report.Summary)
	}
}

func TestVerifyBundle_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "manifest.json"), map[string]any{
		"session_id": "s1", "version": "1.0.0",
	})

	report, _ := VerifyBundle(dir)

	// Ensure the report serializes cleanly
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("cannot marshal report: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty JSON output")
	}

	// Roundtrip
	var rt VerifyReport
	if err := json.Unmarshal(data, &rt); err != nil {
		t.Fatalf("cannot unmarshal report: %v", err)
	}
	if rt.Bundle != dir {
		t.Errorf("bundle mismatch after roundtrip")
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data,0o644); err != nil {
		t.Fatal(err)
	}
}
