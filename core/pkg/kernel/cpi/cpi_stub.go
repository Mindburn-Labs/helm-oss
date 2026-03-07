//go:build !cpi_native

// Package cpi provides the Canonical Policy Interface — a bridge to the
// HELM policy VM. The native (CGO/Rust) implementation is gated behind the
// `cpi_native` build tag. This stub provides a pure-Go fallback that returns
// typed errors, ensuring the package always compiles in the OSS repo.
package cpi

import "errors"

// ErrInvalidInput is returned when the input bytes are malformed.
var ErrInvalidInput = errors.New("cpi: invalid input")

// ErrBundleMismatch is returned when bundle hashes don't match.
var ErrBundleMismatch = errors.New("cpi: bundle mismatch")

// ErrInternal is returned for VM internal errors.
var ErrInternal = errors.New("cpi: internal error")

// ErrNotAvailable is returned when the native CPI bridge is not compiled in.
var ErrNotAvailable = errors.New("cpi: native policy VM not available (build with -tags cpi_native)")

// Validate evaluates a PlanIRDelta against a snapshot using compiled policy bytecode.
// This stub returns ErrNotAvailable — build with `cpi_native` for the real implementation.
func Validate(bytecode, snapshot, delta, facts []byte) ([]byte, error) {
	return nil, ErrNotAvailable
}

// Explain generates a TooltipModelV1 from a CpiVerdict.
// This stub returns ErrNotAvailable — build with `cpi_native` for the real implementation.
func Explain(verdict []byte) ([]byte, error) {
	return nil, ErrNotAvailable
}

// Compile converts DSL source text to a PolicyBundle.
// This stub returns ErrNotAvailable — build with `cpi_native` for the real implementation.
func Compile(source []byte) ([]byte, error) {
	return nil, ErrNotAvailable
}
