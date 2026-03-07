// Package gates provides the gate implementations and a default registry.
package gates

import "github.com/Mindburn-Labs/helm-oss/core/pkg/conform"

// DefaultEngine returns a conformance engine pre-loaded with all gates
// in canonical registration order (G0 → G12, GX_TENANT, GX_ENVELOPE).
// This is the standard way to create an engine for CLI or CI usage.
func DefaultEngine() *conform.Engine {
	e := conform.NewEngine()

	// Core gates (§7)
	e.RegisterGate(&G0BuildIdentity{})
	e.RegisterGate(&G1ProofReceipts{})
	e.RegisterGate(&G2Replay{})
	e.RegisterGate(&G2ASchemaFirst{})
	e.RegisterGate(&G3Policy{})
	e.RegisterGate(&G3ABudget{})
	e.RegisterGate(&G4Secrets{})
	e.RegisterGate(&G5ToolTrust{})
	e.RegisterGate(&G5AA2A{})
	e.RegisterGate(&G6Taint{})
	e.RegisterGate(&G7Incident{})
	e.RegisterGate(&G8HITL{})
	e.RegisterGate(&G9Jurisdiction{})
	e.RegisterGate(&G10Formal{})
	e.RegisterGate(&G11Operability{})
	e.RegisterGate(&G12SupplyChain{})

	// Extension gates
	e.RegisterGate(&GXTenantIsolation{})
	e.RegisterGate(&GXEnvelopeBound{})
	e.RegisterGate(&GXSDKDrift{})

	return e
}
