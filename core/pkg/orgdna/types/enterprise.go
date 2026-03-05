package types

// T6 — Advanced Architecture & Enterprise Readiness
// Cross-cutting improvements for enterprise-grade operation and strategic advantage.

// EnterpriseConfig is the unified container for all enterprise architecture subsystems.
type EnterpriseConfig struct {
	MPC            *MPCConfig            `json:"mpc,omitempty"`             // T6-1
	VSM            *VSMConfig            `json:"vsm,omitempty"`             // T6-2
	VarietyMetrics *VarietyMetricsConfig `json:"variety_metrics,omitempty"` // T6-3
	Simulation     *SimulationConfig     `json:"simulation,omitempty"`      // T6-4
	KnowledgeGraph *KnowledgeGraphConfig `json:"knowledge_graph,omitempty"` // T6-5
	EAAF           *EAAFConfig           `json:"eaaf,omitempty"`            // T6-6
	DigitalTwin    *DigitalTwinConfig    `json:"digital_twin,omitempty"`    // T6-7
	ImmuneSystem   *ImmuneSystemConfig   `json:"immune_system,omitempty"`   // T6-8
}

// MPCConfig — Model Predictive Control (T6-1).
// Replace/augment PID with horizon-planning controllers.
type MPCConfig struct {
	PredictionHorizon int             `json:"prediction_horizon"` // Steps to look ahead
	ControlHorizon    int             `json:"control_horizon"`    // Steps to actuate
	SystemModel       string          `json:"system_model"`       // Model type: "linear", "nonlinear"
	Constraints       []MPCConstraint `json:"constraints,omitempty"`
	CostFunction      string          `json:"cost_function"` // CEL expression for optimization
}

// MPCConstraint defines a constraint for the MPC controller.
type MPCConstraint struct {
	Variable string  `json:"variable"`
	MinValue float64 `json:"min_value"`
	MaxValue float64 `json:"max_value"`
}

// VSMConfig — Viable System Model (T6-2).
// Beer's VSM made explicit and recursive in the genome.
type VSMConfig struct {
	System1     []VSMUnit       `json:"system_1"`                // Operational units
	System2     VSMSystem2      `json:"system_2"`                // Anti-oscillation / coordination
	System3     VSMSystem3      `json:"system_3"`                // Control and optimization
	System3Star *VSMSystem3Star `json:"system_3_star,omitempty"` // Sporadic audit
	System4     VSMSystem4      `json:"system_4"`                // Intelligence / environment scanning
	System5     VSMSystem5      `json:"system_5"`                // Identity / purpose
}

// VSMUnit is a System 1 operational unit (contains its own recursive VSM).
type VSMUnit struct {
	UnitID    string     `json:"unit_id"`
	Purpose   string     `json:"purpose"`
	NestedVSM *VSMConfig `json:"nested_vsm,omitempty"` // Recursive!
}

// VSMSystem2 — Coordination and anti-oscillation.
type VSMSystem2 struct {
	DampeningFactor float64  `json:"dampening_factor"`
	Channels        []string `json:"channels"` // Communication channels between S1 units
}

// VSMSystem3 — Control, optimization, resource bargaining.
type VSMSystem3 struct {
	OptimizationGoal string `json:"optimization_goal"` // CEL expression
	ResourcePolicy   string `json:"resource_policy"`   // "equal", "performance_based", "need_based"
}

// VSMSystem3Star — Sporadic auditing.
type VSMSystem3Star struct {
	AuditFrequency string   `json:"audit_frequency"` // "daily", "weekly", "random"
	AuditTargets   []string `json:"audit_targets"`   // What to audit
}

// VSMSystem4 — Environment scanning and intelligence.
type VSMSystem4 struct {
	ScanSources  []string `json:"scan_sources"`  // External data sources
	UpdatePolicy string   `json:"update_policy"` // "continuous", "periodic", "triggered"
}

// VSMSystem5 — Identity, purpose, and values.
type VSMSystem5 struct {
	Mission  string   `json:"mission"`
	Values   []string `json:"values"`
	Identity string   `json:"identity"` // Organizational identity definition
}

// VarietyMetricsConfig — Requisite Variety Measurement (T6-3).
type VarietyMetricsConfig struct {
	EnvironmentalVariety float64  `json:"environmental_variety"` // Shannon entropy of external demands
	ResponseVariety      float64  `json:"response_variety"`      // Shannon entropy of org capabilities
	VarietyRatio         float64  `json:"variety_ratio"`         // Response / Environmental (must be >= 1.0)
	Amplifiers           []string `json:"amplifiers,omitempty"`  // Mechanisms that increase response variety
	Attenuators          []string `json:"attenuators,omitempty"` // Mechanisms that reduce environmental variety
	AutoAdapt            bool     `json:"auto_adapt"`            // Auto-adapt when ratio < 1.0
}

// SimulationConfig — Deterministic Simulation Testing (T6-4).
// FoundationDB-style simulation harness with fault injection.
type SimulationConfig struct {
	DeterministicClock bool             `json:"deterministic_clock"` // Always true for sim
	FaultInjection     []FaultInjection `json:"fault_injection,omitempty"`
	Invariants         []InvariantCheck `json:"invariants,omitempty"`
	MaxTicks           int              `json:"max_ticks"` // Maximum simulation ticks
	Seed               string           `json:"seed"`      // Deterministic PRNG seed
}

// FaultInjection defines a simulated failure.
type FaultInjection struct {
	FaultID  string  `json:"fault_id"`
	Type     string  `json:"type"`     // "crash", "slow", "corrupt", "partition"
	Target   string  `json:"target"`   // Module or subsystem to affect
	AtTick   int     `json:"at_tick"`  // When to inject
	Duration int     `json:"duration"` // How long the fault lasts
	Severity float64 `json:"severity"` // 0.0-1.0
}

// InvariantCheck defines a condition that must hold at all times during simulation.
type InvariantCheck struct {
	InvariantID string `json:"invariant_id"`
	Condition   string `json:"condition"` // CEL expression that must be true
	Description string `json:"description"`
}

// KnowledgeGraphConfig — Ontology-Based Genome (T6-5).
type KnowledgeGraphConfig struct {
	OntologyRef     string   `json:"ontology_ref"`              // URI to OWL ontology
	InferenceRules  []string `json:"inference_rules,omitempty"` // Reasoning rules
	ProvenanceModel string   `json:"provenance_model"`          // "W3C_PROV_O", "custom"
}

// EAAFConfig — Enterprise Agentic Architecture Framework (T6-6).
// 8-category guardrail taxonomy for agentic AI governance.
type EAAFConfig struct {
	PolicyEngine     string          `json:"policy_engine"`     // Policy engine backend
	IdentityProvider string          `json:"identity_provider"` // IdP for agent identity
	CoordinationBus  string          `json:"coordination_bus"`  // NATS subject prefix
	Guardrails       GuardrailConfig `json:"guardrails"`
	SandboxManager   string          `json:"sandbox_manager"` // Sandbox backend
}

// GuardrailConfig defines the 8-category guardrail taxonomy.
type GuardrailConfig struct {
	Identity    GuardrailPolicy `json:"identity"`    // Who is the agent?
	Data        GuardrailPolicy `json:"data"`        // What data can it access?
	Action      GuardrailPolicy `json:"action"`      // What actions can it take?
	Tool        GuardrailPolicy `json:"tool"`        // What tools can it use?
	Autonomy    GuardrailPolicy `json:"autonomy"`    // How independent is it?
	Behavioral  GuardrailPolicy `json:"behavioral"`  // What behaviors are allowed?
	Audit       GuardrailPolicy `json:"audit"`       // What must be logged?
	Containment GuardrailPolicy `json:"containment"` // What are the blast radius limits?
}

// GuardrailPolicy defines a single guardrail category's policy.
type GuardrailPolicy struct {
	Level     string   `json:"level"` // "strict", "moderate", "permissive"
	AllowList []string `json:"allow_list,omitempty"`
	DenyList  []string `json:"deny_list,omitempty"`
}

// DigitalTwinConfig — Digital Twin of Organization (T6-7).
type DigitalTwinConfig struct {
	TwinID         string            `json:"twin_id"`
	SimProfile     *SimulationConfig `json:"sim_profile,omitempty"`
	ScenarioEngine []Scenario        `json:"scenario_engine,omitempty"`
	TimeWarpFactor float64           `json:"time_warp_factor"` // Speed multiplier
}

// Scenario defines a "what-if" planning scenario.
type Scenario struct {
	ScenarioID  string                 `json:"scenario_id"`
	Description string                 `json:"description"`
	Mutations   map[string]interface{} `json:"mutations"` // Genome mutations to apply
}

// ImmuneSystemConfig — Adaptive Threat Response (T6-8).
// Danger theory, memory cells, clonal selection.
type ImmuneSystemConfig struct {
	DangerDetectors []DangerDetector `json:"danger_detectors,omitempty"`
	MemoryCells     []MemoryCell     `json:"memory_cells,omitempty"`
	ToleranceSet    []string         `json:"tolerance_set,omitempty"` // Known "self" patterns
	ResponsePolicy  string           `json:"response_policy"`         // "investigate", "quarantine", "eliminate"
}

// DangerDetector watches for abnormal patterns.
type DangerDetector struct {
	DetectorID string  `json:"detector_id"`
	Pattern    string  `json:"pattern"`   // Regex or CEL for what constitutes danger
	Threshold  float64 `json:"threshold"` // Sensitivity (0.0-1.0)
}

// MemoryCell records previously encountered threats for faster future response.
type MemoryCell struct {
	ThreatSignature string `json:"threat_signature"`
	ResponseAction  string `json:"response_action"` // What action to take on re-encounter
	LearnedFrom     string `json:"learned_from"`    // Reference to original incident
}
