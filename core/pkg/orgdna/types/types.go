package types

import "time"

// IntentArtifact represents the normalized intent from a natural language prompt.
// Schema: schemas/intent/intent_ticket.v1.schema.json
type IntentArtifact struct {
	ID          string            `json:"id"`
	Prompt      string            `json:"prompt"`
	GoalSpec    GoalSpec          `json:"goal_spec"`
	Constraints []string          `json:"constraints"`
	CreatedAt   time.Time         `json:"created_at"`
	Context     map[string]string `json:"context,omitempty"`
}

// GoalSpec defines the measurable exit criteria for an intent.
type GoalSpec struct {
	Outcome         string   `json:"outcome"`
	SuccessCriteria []string `json:"success_criteria"`
}

// ProcessGraph represents the stable logic of a business process, independent of tools.
// Schema: schemas/planning/plan_bundle.v1.schema.json
type ProcessGraph struct {
	Nodes []ProcessNode `json:"nodes"`
	Edges []ProcessEdge `json:"edges"`
}

// ProcessNode represents a single step in the process graph.
type ProcessNode struct {
	ID             string   `json:"id"`
	Capability     string   `json:"capability"`
	ObligationType string   `json:"obligation_type"`
	Constraints    []string `json:"constraints"`
}

// ProcessEdge represents a dependency between process nodes.
type ProcessEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // e.g., "DEPENDENCY"
}

// ToolBindingPlan maps the stable process graph to specific tools.
// Schema: schemas/planning/plan_bundle.v1.schema.json
type ToolBindingPlan struct {
	Bindings []ToolBinding `json:"bindings"`
	Missing  []MissingTool `json:"missing"`
}

// ToolBinding represents a successful binding of a node to a tool.
type ToolBinding struct {
	NodeID          string `json:"node_id"`
	ToolFingerprint string `json:"tool_fingerprint"`
	ConnectorID     string `json:"connector_id"`
	Status          string `json:"status"` // e.g., "BOUND"
}

// MissingTool represents a gap where no suitable tool was found.
type MissingTool struct {
	NodeID            string `json:"node_id"`
	Capability        string `json:"capability"`
	SuggestedTemplate string `json:"suggested_template"`
}

// MissingOrgansReport details the missing capabilities required to fulfill an intent.
// Schema: l4/MissingOrgansReport.md
type MissingOrgansReport struct {
	MissingCapabilities []string  `json:"missing_capabilities"`
	MissingEffectTypes  []string  `json:"missing_effect_types"`
	Rationale           string    `json:"rationale"`
	GeneratedAt         time.Time `json:"generated_at"`
	SchemaVersion       string    `json:"schema_version"`
}

// ContractViolationReport details a breach of the Phenotype Contract.
// Schema: protocols/json-schemas/orgdna/contract_violation.schema.json
type ContractViolationReport struct {
	ViolationID  string    `json:"violation_id"`
	GenomeID     string    `json:"genome_id"`
	ContractType string    `json:"contract_type"` // e.g. "MUST_PRODUCE"
	Details      string    `json:"details"`
	DetectedAt   time.Time `json:"detected_at"`
	Severity     string    `json:"severity"` // BLOCKING, WARNING
}

// ToolCatalog represents the available tools and their fingerprints.
// Schema: protocols/json-schemas/tooling/tool_catalog.schema.json
type ToolCatalog struct {
	Tools []ToolDefinition `json:"tools"`
}

type ToolDefinition struct {
	ID           string   `json:"id"`
	Fingerprint  string   `json:"fingerprint"`
	Capabilities []string `json:"capabilities"`
}

// --- OrgDNA Compiler Types ---

// OrgGenome represents the declarative organizational genome.
// Schema: protocols/json-schemas/orgdna/orggenome.schema.json
type OrgGenome struct {
	Meta              GenomeMeta            `json:"meta"`
	Morphogenesis     []MorphogenesisRule   `json:"morphogenesis"`
	Regulation        RegulationConfig      `json:"regulation"`
	PhenotypeContract PhenotypeContract     `json:"phenotype_contract"`
	Environment       *EnvironmentProfile   `json:"environment,omitempty"`
	Identity          *OrgIdentity          `json:"identity,omitempty"`
	Viability         *ViabilityModel       `json:"viability,omitempty"`
	Architecture      *OrgArchitecture      `json:"architecture,omitempty"`
	Primitives        *OrgPrimitives        `json:"primitives,omitempty"`
	Modules           []ModuleDeclaration   `json:"modules"`
	SpliceVariants    []SpliceVariantConfig `json:"splice_variants,omitempty"` // T1-2
	GeneRegulation    *GenomeRegulation     `json:"gene_regulation,omitempty"` // T2
	RNARegulation     *RNARegulation        `json:"rna_regulation,omitempty"`  // T3
	Organelles        *OrganelleConfig      `json:"organelles,omitempty"`      // T4
	Lifecycle         *LifecycleConfig      `json:"lifecycle,omitempty"`       // T5
	Enterprise        *EnterpriseConfig     `json:"enterprise,omitempty"`      // T6
	Provenance        map[string]string     `json:"provenance,omitempty"`
}

type GenomeMeta struct {
	GenomeID  string    `json:"genome_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	License   string    `json:"license,omitempty"`
}

type MorphogenesisRule struct {
	ID       string                 `json:"id"`
	When     string                 `json:"when"`     // CEL expression
	Generate MorphogenesisGenerator `json:"generate"` // Structure to generate
}

type MorphogenesisGenerator struct {
	Modules    []ModuleDeclaration `json:"modules,omitempty"`
	Regulation *RegulationConfig   `json:"regulation,omitempty"`
}

type RegulationConfig struct {
	EssentialVariables []EssentialVariable    `json:"essential_variables,omitempty"`
	ControlLoops       []ControlLoop          `json:"control_loops,omitempty"`
	RegulationGraph    *RegulationGraph       `json:"regulation_graph,omitempty"`
	PolicySet          map[string]interface{} `json:"policy_set,omitempty"`
}

type EssentialVariable struct {
	Name         string                  `json:"name"`
	VariableID   string                  `json:"variable_id"`
	Bounds       EssentialVariableBounds `json:"bounds"`
	CurrentValue float64                 `json:"current_value,omitempty"` // T0-6: Sensor-fed current value
}

type EssentialVariableBounds struct {
	Type string  `json:"type"` // e.g., "numeric_range", "boolean"
	Min  float64 `json:"min,omitempty"`
	Max  float64 `json:"max,omitempty"`
}

type ControlLoop struct {
	LoopID     string             `json:"loop_id"`
	Type       string             `json:"type,omitempty"`       // e.g., "pid", "hysteresis"
	Parameters map[string]float64 `json:"parameters,omitempty"` // e.g., Kp, Ki, Kd
	Input      string             `json:"input,omitempty"`      // Sensor variable ID
	Output     string             `json:"output,omitempty"`     // Actuator variable ID
	Expression string             `json:"expression,omitempty"` // CEL expression for custom logic
}

type RegulationGraph struct {
	InitialModeID string           `json:"initial_mode_id"`
	Modes         []RegulationMode `json:"modes"`
}

type RegulationMode struct {
	ModeID string `json:"mode_id"`
	// ...
}

type PhenotypeContract struct {
	MustProduce []string `json:"must_produce"`
	Determinism struct {
		RequiresRandomSeed bool `json:"requires_random_seed"`
	} `json:"determinism"`
}

type EnvironmentProfile struct {
	ProfileID string                 `json:"profile_id"`
	Bindings  map[string]interface{} `json:"bindings"`
}

type OrgIdentity struct {
	Name string `json:"name"`
}

type ViabilityModel struct {
	// ...
}

type OrgArchitecture struct {
	// ...
}

// OrgPrimitives defines the fundamental building blocks of the organization (L1).
// Per OrgDNA/1.0 Spec Section 4.
type OrgPrimitives struct {
	Units      []OrgUnit   `json:"units,omitempty"`
	Roles      []Role      `json:"roles,omitempty"`
	Principals []Principal `json:"principals,omitempty"`
}

// OrgUnit represents a recursive organizational unit (e.g., Department, Team, Pod).
// Per Section 4.2 - Units.
type OrgUnit struct {
	UnitID          string    `json:"unit_id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // e.g., "team", "division", "squad"
	ParentID        string    `json:"parent_id,omitempty"`
	Children        []OrgUnit `json:"children,omitempty"` // Recursive structure
	AssignedRoles   []string  `json:"assigned_roles,omitempty"`
	PolicyRefs      []string  `json:"policy_refs,omitempty"`
	BudgetRef       string    `json:"budget_ref,omitempty"`
	KnowledgeBaseID string    `json:"knowledge_base_id,omitempty"`
}

// Role defines a set of responsibilities and permissions.
// Per Section 4.3 - Principals and Roles.
type Role struct {
	RoleID             string   `json:"role_id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Responsibilities   []string `json:"responsibilities"`
	DelegationAllowed  bool     `json:"delegation_allowed"`
	SeparationOfDuties []string `json:"separation_of_duties,omitempty"` // Conflicts with these RoleIDs
	RequiredTraits     []string `json:"required_traits,omitempty"`
}

// Principal represents an actor that can assume a role.
// Per Section 4.3 - Principals and Roles.
type Principal struct {
	PrincipalID string            `json:"principal_id"`
	Type        string            `json:"type"` // "human", "agent", "system"
	Name        string            `json:"name"`
	Traits      []string          `json:"traits,omitempty"`
	RiskScore   float64           `json:"risk_score,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ModuleDeclaration struct {
	Type         string             `json:"type"`
	Jurisdiction string             `json:"jurisdiction"`
	Version      string             `json:"version"`
	Capabilities []string           `json:"capabilities,omitempty"` // Provided capabilities; falls back to "{type}_capability" if empty
	Attestation  *ModuleAttestation `json:"attestation,omitempty"`  // T0-3: Cryptographic attestation
}

// ModuleAttestation provides cryptographic proof of module provenance and integrity (T0-3).
// Spec: REGULATION_SPEC §4 — Module Attestation Validation.
type ModuleAttestation struct {
	SignerID  string `json:"signer_id"`            // Key ID of the attestation signer
	Signature string `json:"signature"`            // Ed25519 signature over canonical module content
	PublicKey string `json:"public_key,omitempty"` // Hex-encoded public key (for self-signed modules)
	Digest    string `json:"digest"`               // SHA-256 digest of (type + jurisdiction + version)
	IssuedAt  string `json:"issued_at,omitempty"`  // RFC3339 timestamp
}

// OrgPhenotype represents the compiled runtime artifact.
// Schema: protocols/json-schemas/orgdna/orgphenotype.schema.json
// OrgPhenotype represents the compiled runtime artifact.
// Schema: protocols/json-schemas/orgdna/orgphenotype.schema.json
type OrgPhenotype struct {
	Metadata            PhenotypeMetadata         `json:"metadata"`
	OrgGraph            *OrgArchitecture          `json:"org_graph,omitempty"`
	Primitives          *OrgPrimitives            `json:"primitives,omitempty"`
	ActiveModules       []ModuleDeclaration       `json:"active_modules,omitempty"`
	ActiveCapabilities  []string                  `json:"active_capabilities,omitempty"`
	RegulationConfig    RegulationConfig          `json:"regulation_config"`
	Environment         *EnvironmentProfile       `json:"environment,omitempty"`
	Factories           []FactorySpec             `json:"factories,omitempty"` // Autonomous Factories
	ActivePhenotypeHash string                    `json:"active_phenotype_hash"`
	Receipt             *CompilationReceipt       `json:"receipt,omitempty"`
	MorphTimeline       []MorphogenesisSnapshot   `json:"morph_timeline,omitempty"`
	ContractViolations  []ContractViolationReport `json:"contract_violations,omitempty"`
	OrganelleConfig     *OrganelleConfig          `json:"organelle_config,omitempty"`  // T4
	LifecycleConfig     *LifecycleConfig          `json:"lifecycle_config,omitempty"`  // T5
	EnterpriseConfig    *EnterpriseConfig         `json:"enterprise_config,omitempty"` // T6
}

type FactorySpec struct {
	ID                 string              `json:"id"`
	Goal               string              `json:"goal"`
	EssentialVariables []EssentialVariable `json:"essential_variables"`
	Actions            []FactoryActionDef  `json:"actions"`
	// PlanningLogic is a simplified rule set for now.
	// e.g. "tick % 10 == 0 -> emit(ActionScan)"
	PlanningLogic string `json:"planning_logic"`
}

type FactoryActionDef struct {
	Type       string            `json:"type"`
	Parameters map[string]string `json:"parameters"`
}

type PhenotypeMetadata struct {
	GenomeID      string    `json:"genome_id"`
	SourceHash    string    `json:"source_hash"`
	CompiledAt    time.Time `json:"compiled_at"`
	SchemaVersion string    `json:"schema_version"`
}

// CompilationReceipt represents a cryptographic attestation of the compilation process.
type CompilationReceipt struct {
	ReceiptID       string    `json:"receipt_id"`
	GenomeID        string    `json:"genome_id"`
	InputHash       string    `json:"input_hash"`
	OutputHash      string    `json:"output_hash"`
	CompiledAt      time.Time `json:"compiled_at"`
	Signature       string    `json:"signature,omitempty"`
	SignerID        string    `json:"signer_id,omitempty"`
	SignerPublicKey string    `json:"signer_public_key,omitempty"` // T0-7: For self-verification
}

// MorphogenesisSnapshot captures the state of the genome at a specific iteration of morphogenesis.
type MorphogenesisSnapshot struct {
	Iteration int       `json:"iteration"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}
