// Package contracts — Canonical Verdict and Reason Code Registry.
//
// Per HELM Standard v1.2 §1.2 — every governance decision MUST use these
// canonical verdict values. No other verdict strings are valid.
//
// This file is the SINGLE SOURCE OF TRUTH for verdict vocabulary.
// All kernel components (Guardian, Executor, PDP, SDK) import from here.
package contracts

// Verdict is the canonical verdict type for all HELM governance decisions.
// Wire format: JSON string field "verdict" in DecisionRecord and Receipt.
type Verdict string

const (
	// VerdictAllow indicates the effect is permitted. Decision is signed, Intent is issued.
	VerdictAllow Verdict = "ALLOW"

	// VerdictDeny indicates the effect is refused. A DenialReceipt is emitted.
	VerdictDeny Verdict = "DENY"

	// VerdictEscalate indicates the effect requires human/ceremony approval.
	VerdictEscalate Verdict = "ESCALATE"
)

// IsTerminal returns true if the verdict is a final state (not pending escalation).
func (v Verdict) IsTerminal() bool {
	return v == VerdictAllow || v == VerdictDeny
}

// ReasonCode is a typed, machine-readable reason for DENY/ESCALATE verdicts.
// These codes form a canonical registry analogous to HTTP status codes.
// Wire format: JSON string field "reason_code" in DecisionRecord.
type ReasonCode string

const (
	// ── Policy Reasons ─────────────────────────────────────
	ReasonPolicyViolation    ReasonCode = "POLICY_VIOLATION"
	ReasonNoPolicy           ReasonCode = "NO_POLICY_DEFINED"
	ReasonPRGEvalError       ReasonCode = "PRG_EVALUATION_ERROR"
	ReasonMissingRequirement ReasonCode = "MISSING_REQUIREMENT"

	// ── PDP Reasons ────────────────────────────────────────
	ReasonPDPDeny  ReasonCode = "PDP_DENY"
	ReasonPDPError ReasonCode = "PDP_ERROR"

	// ── Resource Reasons ───────────────────────────────────
	ReasonBudgetExceeded ReasonCode = "BUDGET_EXCEEDED"
	ReasonBudgetError    ReasonCode = "BUDGET_ERROR"

	// ── Envelope / Schema Reasons ──────────────────────────
	ReasonEnvelopeInvalid ReasonCode = "ENVELOPE_INVALID"
	ReasonSchemaViolation ReasonCode = "SCHEMA_VIOLATION"

	// ── Temporal Reasons ───────────────────────────────────
	ReasonTemporalIntervene ReasonCode = "TEMPORAL_INTERVENTION"
	ReasonTemporalThrottle  ReasonCode = "TEMPORAL_THROTTLE"

	// ── Security Reasons ───────────────────────────────────
	ReasonSandboxViolation ReasonCode = "SANDBOX_VIOLATION"
	ReasonProvenance       ReasonCode = "PROVENANCE_FAILURE"
	ReasonVerification     ReasonCode = "VERIFICATION_FAILURE"

	// ── Tenancy / Jurisdiction Reasons ─────────────────────
	ReasonTenantIsolation ReasonCode = "TENANT_ISOLATION"
	ReasonJurisdiction    ReasonCode = "JURISDICTION_VIOLATION"
)
