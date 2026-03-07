package cpi

import (
	"errors"
	"testing"
)

func TestValidateStubReturnsNotAvailable(t *testing.T) {
	_, err := Validate(nil, nil, nil, nil)
	if err == nil {
		// If CPI native is compiled in we accept success too
		return
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("expected ErrNotAvailable, got: %v", err)
	}
}

func TestCompileStubReturnsNotAvailable(t *testing.T) {
	_, err := Compile([]byte("source"))
	if err == nil {
		return
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("expected ErrNotAvailable, got: %v", err)
	}
}

func TestExplainStubReturnsNotAvailable(t *testing.T) {
	_, err := Explain(nil)
	if err == nil {
		return
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("expected ErrNotAvailable, got: %v", err)
	}
}
