package contracts

// Canonical threat-surface effect type IDs.
// These are the stable identifiers for effects that require specific
// enforcement behavior, risk classification, and approval semantics.
//
// Per the HELM Canonical Implementation Plan: every high-risk effect
// MUST be named, classified, and registered in DefaultEffectCatalog().
const (
	// Infrastructure effects
	EffectTypeInfraDestroy        = "INFRA_DESTROY"             // Destroy infrastructure (e.g., terraform destroy)
	EffectTypeEnvRecreate         = "ENV_RECREATE"              // Recreate/replace an execution environment
	EffectTypeProtectedInfraWrite = "PROTECTED_INFRA_STRUCTURE" // Mutate protected infrastructure (e.g., production DB schema)

	// CI/CD and supply-chain effects
	EffectTypeCICredentialAccess = "CI_CREDENTIAL_ACCESS" // Access CI/CD credentials or secrets
	EffectTypeSoftwarePublish    = "SOFTWARE_PUBLISH"     // Publish software artifact (npm, Docker, etc.)

	// Agent and identity effects
	EffectTypeAgentInvokePrivileged  = "AGENT_INVOKE_PRIVILEGED"  // Agent invoking privileged operation
	EffectTypeAgentIdentityIsolation = "AGENT_IDENTITY_ISOLATION" // Agent credential/identity boundary check

	// Network and data effects
	EffectTypeDataEgress  = "DATA_EGRESS"  // Transmit data to external endpoint
	EffectTypeTunnelStart = "TUNNEL_START" // Establish network tunnel (SSH, VPN, etc.)

	// Resource effects
	EffectTypeCloudComputeBudget = "CLOUD_COMPUTE_BUDGET" // Consume cloud compute resources against budget
)

// DefaultEffectCatalog returns the canonical EffectTypeCatalog pre-populated
// with all threat-surface effect types and their classifications.
//
// Each entry specifies:
//   - Risk classification (reversibility, blast radius, urgency)
//   - Default approval level (none, single_human, dual_control, quorum)
//   - Whether evidence is required
//   - Whether compensation/rollback is required
func DefaultEffectCatalog() *EffectTypeCatalog {
	return &EffectTypeCatalog{
		CatalogVersion: "1.0.0",
		EffectTypes: []EffectType{
			{
				TypeID:      EffectTypeInfraDestroy,
				Name:        "Infrastructure Destroy",
				Description: "Destroys infrastructure resources (e.g., terraform destroy, droplet deletion). Irreversible without backup.",
				Idempotency: IdempotencyRef{Strategy: "none"},
				Classification: Classification{
					Reversibility: "irreversible",
					BlastRadius:   "system_wide",
					Urgency:       "immediate",
				},
				DefaultApprovalLevel: "dual_control",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/infra_destroy.json",
			},
			{
				TypeID:      EffectTypeEnvRecreate,
				Name:        "Environment Recreate",
				Description: "Recreates or replaces an execution environment. May invalidate context fingerprints and running state.",
				Idempotency: IdempotencyRef{Strategy: "content_hash"},
				Classification: Classification{
					Reversibility: "compensatable",
					BlastRadius:   "system_wide",
					Urgency:       "time_sensitive",
				},
				DefaultApprovalLevel: "single_human",
				RequiresEvidence:     true,
				CompensationRequired: true,
				ReceiptSchema:        "effects/threat/env_recreate.json",
			},
			{
				TypeID:      EffectTypeProtectedInfraWrite,
				Name:        "Protected Infrastructure Mutation",
				Description: "Mutates protected infrastructure such as production database schemas, load balancer configs, or DNS records.",
				Idempotency: IdempotencyRef{Strategy: "content_hash"},
				Classification: Classification{
					Reversibility: "compensatable",
					BlastRadius:   "system_wide",
					Urgency:       "time_sensitive",
				},
				DefaultApprovalLevel: "dual_control",
				RequiresEvidence:     true,
				CompensationRequired: true,
				ReceiptSchema:        "effects/threat/protected_infra_structure.json",
			},
			{
				TypeID:      EffectTypeCICredentialAccess,
				Name:        "CI Credential Access",
				Description: "Accesses CI/CD credentials, signing keys, or deployment secrets. Supply-chain attack vector.",
				Idempotency: IdempotencyRef{Strategy: "none"},
				Classification: Classification{
					Reversibility: "irreversible",
					BlastRadius:   "system_wide",
					Urgency:       "immediate",
				},
				DefaultApprovalLevel: "dual_control",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/ci_credential_access.json",
			},
			{
				TypeID:      EffectTypeSoftwarePublish,
				Name:        "Software Publish",
				Description: "Publishes a software artifact to a registry (npm, Docker Hub, PyPI, etc.). Irreversible in public registries.",
				Idempotency: IdempotencyRef{Strategy: "content_hash", KeyComposition: []string{"registry", "package", "version"}},
				Classification: Classification{
					Reversibility: "irreversible",
					BlastRadius:   "system_wide",
					Urgency:       "time_sensitive",
				},
				DefaultApprovalLevel: "dual_control",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/software_publish.json",
			},
			{
				TypeID:      EffectTypeAgentInvokePrivileged,
				Name:        "Agent Invoke Privileged",
				Description: "Agent invokes a privileged operation requiring elevated principal strength or explicit delegation.",
				Idempotency: IdempotencyRef{Strategy: "effect_id"},
				Classification: Classification{
					Reversibility: "compensatable",
					BlastRadius:   "dataset",
					Urgency:       "immediate",
				},
				DefaultApprovalLevel: "single_human",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/agent_invoke_privileged.json",
			},
			{
				TypeID:      EffectTypeAgentIdentityIsolation,
				Name:        "Agent Identity Isolation Check",
				Description: "Validates that agent instances maintain credential isolation. Detects shared-secret or impersonation attempts.",
				Idempotency: IdempotencyRef{Strategy: "none"},
				Classification: Classification{
					Reversibility: "reversible",
					BlastRadius:   "single_record",
					Urgency:       "immediate",
				},
				DefaultApprovalLevel: "none",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/agent_identity_isolation.json",
			},
			{
				TypeID:      EffectTypeDataEgress,
				Name:        "Data Egress",
				Description: "Transmits data to an external endpoint. Primary exfiltration vector.",
				Idempotency: IdempotencyRef{Strategy: "none"},
				Classification: Classification{
					Reversibility: "irreversible",
					BlastRadius:   "system_wide",
					Urgency:       "immediate",
				},
				DefaultApprovalLevel: "dual_control",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/data_egress.json",
			},
			{
				TypeID:      EffectTypeTunnelStart,
				Name:        "Tunnel Start",
				Description: "Establishes a network tunnel (SSH, VPN, reverse proxy). Enables persistent covert channels.",
				Idempotency: IdempotencyRef{Strategy: "content_hash", KeyComposition: []string{"destination", "protocol"}},
				Classification: Classification{
					Reversibility: "reversible",
					BlastRadius:   "system_wide",
					Urgency:       "immediate",
				},
				DefaultApprovalLevel: "single_human",
				RequiresEvidence:     true,
				CompensationRequired: false,
				ReceiptSchema:        "effects/threat/tunnel_start.json",
			},
			{
				TypeID:      EffectTypeCloudComputeBudget,
				Name:        "Cloud Compute Budget",
				Description: "Consumes cloud compute resources against a tenant budget. Crypto-mining and resource-hijack vector.",
				Idempotency: IdempotencyRef{Strategy: "effect_id"},
				Classification: Classification{
					Reversibility: "compensatable",
					BlastRadius:   "dataset",
					Urgency:       "time_sensitive",
				},
				DefaultApprovalLevel: "none",
				RequiresEvidence:     true,
				CompensationRequired: true,
				ReceiptSchema:        "effects/threat/cloud_compute_budget.json",
			},
		},
	}
}

// EffectRiskClass maps a canonical effect type ID to its E-class risk level.
// This bridges the named effect taxonomy to the governance engine's E0-E4 system.
//
//   - E0: Informational (read-only, no side effects)
//   - E1: Low Risk / Reversible
//   - E2: Medium Risk / State Mutation
//   - E3: High Risk / Sensitive Data
//   - E4: Critical / Irreversible
func EffectRiskClass(effectTypeID string) string {
	switch effectTypeID {
	case EffectTypeInfraDestroy, EffectTypeCICredentialAccess,
		EffectTypeSoftwarePublish, EffectTypeDataEgress:
		return "E4" // Critical / Irreversible
	case EffectTypeProtectedInfraWrite, EffectTypeEnvRecreate,
		EffectTypeAgentInvokePrivileged, EffectTypeTunnelStart:
		return "E3" // High Risk
	case EffectTypeCloudComputeBudget:
		return "E2" // Medium Risk (budget-gated)
	case EffectTypeAgentIdentityIsolation:
		return "E1" // Low Risk (validation check)
	default:
		return "E3" // Fail-closed: unknown effect types default to high risk
	}
}

// LookupEffectType returns the EffectType definition for a given type ID
// from the default catalog, or nil if not found.
func LookupEffectType(typeID string) *EffectType {
	catalog := DefaultEffectCatalog()
	for i := range catalog.EffectTypes {
		if catalog.EffectTypes[i].TypeID == typeID {
			return &catalog.EffectTypes[i]
		}
	}
	return nil
}
