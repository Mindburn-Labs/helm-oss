// Package wasm provides WebAssembly-based policy evaluation for HELM.
//
// This enables sandboxed, portable policy execution across environments.
// Policies compiled to WASM can run deterministically in any HELM deployment
// without native code dependencies.
package wasm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// PolicyModule represents a compiled WASM policy module.
type PolicyModule struct {
	// ID is the content-addressed hash of the WASM binary.
	ID string `json:"id"`

	// Name is the human-readable policy name.
	Name string `json:"name"`

	// Version is the policy version.
	Version string `json:"version"`

	// Binary is the compiled WASM bytecode.
	Binary []byte `json:"-"`

	// Hash is the SHA-256 hash of the binary.
	Hash string `json:"hash"`

	// Entrypoint is the exported WASM function name to call.
	Entrypoint string `json:"entrypoint"`

	// LoadedAt is when the module was loaded.
	LoadedAt time.Time `json:"loaded_at"`
}

// EvalRequest is the input to a WASM policy evaluation.
type EvalRequest struct {
	Principal string          `json:"principal"`
	Action    string          `json:"action"`
	Resource  string          `json:"resource"`
	Context   json.RawMessage `json:"context,omitempty"`
}

// EvalResult is the output of a WASM policy evaluation.
type EvalResult struct {
	Decision  string        `json:"decision"` // "ALLOW", "DENY", "ESCALATE"
	Reason    string        `json:"reason"`
	PolicyID  string        `json:"policy_id"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// Runtime manages WASM policy modules.
type Runtime struct {
	modules map[string]*PolicyModule
}

// NewRuntime creates a new WASM policy runtime.
func NewRuntime() *Runtime {
	return &Runtime{modules: make(map[string]*PolicyModule)}
}

// LoadModule loads a compiled WASM policy module.
func (r *Runtime) LoadModule(name, version string, binary []byte, entrypoint string) (*PolicyModule, error) {
	if len(binary) == 0 {
		return nil, fmt.Errorf("wasm: empty binary")
	}

	hash := sha256.Sum256(binary)
	hashStr := hex.EncodeToString(hash[:])

	mod := &PolicyModule{
		ID:         hashStr[:32],
		Name:       name,
		Version:    version,
		Binary:     binary,
		Hash:       hashStr,
		Entrypoint: entrypoint,
		LoadedAt:   time.Now(),
	}

	r.modules[name] = mod
	return mod, nil
}

// Evaluate runs a policy evaluation against a loaded WASM module.
func (r *Runtime) Evaluate(_ context.Context, moduleName string, req EvalRequest) (*EvalResult, error) {
	mod, ok := r.modules[moduleName]
	if !ok {
		return nil, fmt.Errorf("wasm: module %s not loaded", moduleName)
	}

	start := time.Now()

	// In production, this would:
	// 1. Instantiate the WASM module using wazero or wasmtime-go
	// 2. Serialize the EvalRequest to the module's linear memory
	// 3. Call the entrypoint function
	// 4. Deserialize the result from linear memory
	//
	// For now, we return a default DENY (fail-closed) since the actual
	// WASM runtime is not yet integrated.
	result := &EvalResult{
		Decision:  "DENY",
		Reason:    fmt.Sprintf("WASM module %s (%s) evaluation pending runtime integration", mod.Name, mod.Version),
		PolicyID:  mod.ID,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}

	return result, nil
}

// ListModules returns all loaded modules.
func (r *Runtime) ListModules() []*PolicyModule {
	var modules []*PolicyModule
	for _, m := range r.modules {
		modules = append(modules, m)
	}
	return modules
}

// GetModule returns a module by name.
func (r *Runtime) GetModule(name string) (*PolicyModule, bool) {
	mod, ok := r.modules[name]
	return mod, ok
}
