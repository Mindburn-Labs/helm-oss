package cpi

import (
	"testing"
)

// These tests require the helm-policy-vm cdylib to be built.
// Run: cargo build --release -p helm-policy-vm
// Then: go test ./kernel/cpi/...

func TestValidateEmpty(t *testing.T) {
	// Empty inputs should return OK with empty verdict
	verdict, err := Validate(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	// Empty verdict is valid (stub implementation returns empty bytes)
	t.Logf("verdict bytes: %d", len(verdict))
}

func TestCompileInvalidSource(t *testing.T) {
	// Invalid source should return ErrInvalidInput
	_, err := Compile([]byte("this is not valid policy DSL"))
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestExplainEmpty(t *testing.T) {
	// Empty verdict should return OK with empty tooltip
	tooltip, err := Explain(nil)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	t.Logf("tooltip bytes: %d", len(tooltip))
}
