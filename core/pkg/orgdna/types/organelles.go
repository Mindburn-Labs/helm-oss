package types

// T4 — Runtime Organelle Subsystems
// Maps every cellular organelle to a runtime subsystem configuration.

// OrganelleConfig is the unified container for all runtime organelle subsystems.
type OrganelleConfig struct {
	Golgi              *GolgiConfig        `json:"golgi,omitempty"`               // T4-1: Dispatch & Routing
	Lysosome           *LysosomeConfig     `json:"lysosome,omitempty"`            // T4-2: Recycling
	RoughER            *RoughERConfig      `json:"rough_er,omitempty"`            // T4-3: Outbound Service Layer
	SmoothER           *SmoothERConfig     `json:"smooth_er,omitempty"`           // T4-4: Resource Metabolism
	Peroxisome         *PeroxisomeConfig   `json:"peroxisome,omitempty"`          // T4-5: Compliance Filter
	Centrosome         *CentrosomeConfig   `json:"centrosome,omitempty"`          // T4-6: Division Orchestrator
	Energy             *EnergyConfig       `json:"energy,omitempty"`              // T4-7: Mitochondria / ATP
	Cytoskeleton       *CytoskeletonConfig `json:"cytoskeleton,omitempty"`        // T4-8
	Membrane           *MembraneConfig     `json:"membrane,omitempty"`            // T4-9
	VesicularTransport *VesicularConfig    `json:"vesicular_transport,omitempty"` // T4-10
}

// GolgiConfig — Dispatch & Routing (T4-1).
// Sort, modify, tag, and route factory outputs to correct recipients.
type GolgiConfig struct {
	// ReceivingCisterna defines the intake interface for factory outputs.
	ReceivingCisterna ReceivingCisterna `json:"receiving_cisterna"`

	// ModificationStack is an ordered list of post-processors applied to outputs.
	ModificationStack []OutputModifier `json:"modification_stack,omitempty"`

	// SortingSignals define routing rules for processed outputs.
	SortingSignals []SortingSignal `json:"sorting_signals,omitempty"`
}

// ReceivingCisterna is the Golgi intake interface.
type ReceivingCisterna struct {
	AcceptTypes []string `json:"accept_types"` // Output types to accept
	BufferSize  int      `json:"buffer_size"`  // Max queued items
}

// OutputModifier is a post-processing step in the Golgi stack.
type OutputModifier struct {
	ModifierID string                 `json:"modifier_id"`
	Type       string                 `json:"type"` // "format", "validate", "encrypt", "tag"
	Config     map[string]interface{} `json:"config,omitempty"`
}

// SortingSignal defines routing rules for output dispatch.
type SortingSignal struct {
	Signal      string `json:"signal"`      // Routing tag
	Destination string `json:"destination"` // Target: "external", "internal:{module}", "archive"
	Priority    int    `json:"priority"`
}

// LysosomeConfig — Organizational Recycling (T4-2).
// Decommission deprecated modules, recycle resources, clean audit logs.
type LysosomeConfig struct {
	DegradationRules []DegradationRule `json:"degradation_rules,omitempty"`
	AutophagyQueue   []string          `json:"autophagy_queue,omitempty"` // Module types queued for recycling
	PHLevel          float64           `json:"ph_level"`                  // Aggressiveness (0.0 = gentle, 1.0 = aggressive)
}

// DegradationRule defines when and how a module should be decomposed.
type DegradationRule struct {
	RuleID        string `json:"rule_id"`
	TargetModule  string `json:"target_module"`  // Module type to degrade
	Condition     string `json:"condition"`      // CEL expression
	RecycleOutput bool   `json:"recycle_output"` // Reclaim resources?
}

// RoughERConfig — Outbound Service Layer (T4-3).
// Prepare and validate factory outputs for external delivery.
type RoughERConfig struct {
	ValidationRules []ValidationRule `json:"validation_rules,omitempty"`
	QualityGate     QualityGate      `json:"quality_gate"`
}

// ValidationRule checks factory outputs before they leave the org boundary.
type ValidationRule struct {
	RuleID    string                 `json:"rule_id"`
	CheckType string                 `json:"check_type"` // "schema", "policy", "format", "size"
	Config    map[string]interface{} `json:"config,omitempty"`
}

// QualityGate is the final check before output exits.
type QualityGate struct {
	MinConfidence float64 `json:"min_confidence"` // Minimum quality threshold
	RejectAction  string  `json:"reject_action"`  // "drop", "quarantine", "retry"
}

// SmoothERConfig — Resource Metabolism (T4-4).
// Currency conversion, compliance filtering, resource buffering.
type SmoothERConfig struct {
	ConversionRules []ResourceConversion `json:"conversion_rules,omitempty"`
	BufferConfig    BufferConfig         `json:"buffer_config"`
}

// ResourceConversion defines how one resource type converts to another.
type ResourceConversion struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Rate     float64 `json:"rate"`      // Conversion rate
	MaxBatch float64 `json:"max_batch"` // Maximum per conversion
}

// BufferConfig controls resource buffering during demand spikes.
type BufferConfig struct {
	MaxCapacity float64 `json:"max_capacity"`
	DrainRate   float64 `json:"drain_rate"`
	FillRate    float64 `json:"fill_rate"`
}

// PeroxisomeConfig — Compliance Filter (T4-5).
// Detoxify risky inputs, process regulatory requirements.
type PeroxisomeConfig struct {
	Filters []ComplianceFilter `json:"filters,omitempty"`
}

// ComplianceFilter sanitizes incoming data/requests.
type ComplianceFilter struct {
	FilterID string                 `json:"filter_id"`
	Type     string                 `json:"type"` // "pii_scrub", "injection_guard", "rate_limit", "format_check"
	Config   map[string]interface{} `json:"config,omitempty"`
}

// CentrosomeConfig — Division Orchestrator (T4-6).
// Manage organizational splits and subsidiary creation.
type CentrosomeConfig struct {
	DivisionPlans []DivisionPlan `json:"division_plans,omitempty"`
}

// DivisionPlan specifies how to split an organization.
type DivisionPlan struct {
	PlanID         string             `json:"plan_id"`
	AssetSplit     map[string]float64 `json:"asset_split"`      // Module → fraction allocated to child
	GenomeForkRule string             `json:"genome_fork_rule"` // "full_copy", "selective", "minimal"
}

// EnergyConfig — Mitochondria / Universal Energy System (T4-7).
// ATP-like universal compute budget, API quota management, death trigger.
type EnergyConfig struct {
	MaxATP             float64            `json:"max_atp"`                // Maximum energy budget
	CurrentATP         float64            `json:"current_atp"`            // Current energy level
	RegenerationRate   float64            `json:"regeneration_rate"`      // ATP regeneration per tick
	ApoptosisThreshold float64            `json:"apoptosis_threshold"`    // Below this → trigger death
	QuotaLimits        map[string]float64 `json:"quota_limits,omitempty"` // Per-module energy caps
}

// CytoskeletonConfig — Dynamic Organizational Restructuring (T4-8).
// Motor protein transport chains for internal data routing.
type CytoskeletonConfig struct {
	TransportRoutes []TransportRoute `json:"transport_routes,omitempty"`
	Restructuring   bool             `json:"restructuring"` // Allow dynamic reorg?
}

// TransportRoute defines directional, prioritized internal data routing.
type TransportRoute struct {
	RouteID   string `json:"route_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Direction string `json:"direction"` // "unidirectional", "bidirectional"
	Priority  int    `json:"priority"`
}

// MembraneConfig — Cell Membrane / Organizational Boundary (T4-9).
// Selective permeability, receptor inventory, API gateway.
type MembraneConfig struct {
	Receptors         []Receptor         `json:"receptors,omitempty"`
	PermeabilityRules []PermeabilityRule `json:"permeability_rules,omitempty"`
}

// Receptor defines an external signal receptor on the org boundary.
type Receptor struct {
	ReceptorID string `json:"receptor_id"`
	SignalType string `json:"signal_type"` // Type of external signal this receives
	Handler    string `json:"handler"`     // Module that handles this signal
}

// PermeabilityRule controls what can enter/exit the org boundary.
type PermeabilityRule struct {
	RuleID    string `json:"rule_id"`
	Direction string `json:"direction"` // "inbound", "outbound", "both"
	DataType  string `json:"data_type"` // What data type this controls
	Action    string `json:"action"`    // "allow", "deny", "transform"
}

// VesicularConfig — Vesicular Transport / Inter-Subsystem Messages (T4-10).
// SNARE-like tagging and routing for internal messages.
type VesicularConfig struct {
	TaggingProtocol TaggingProtocol `json:"tagging_protocol"`
}

// TaggingProtocol defines the message tagging standard for internal transport.
type TaggingProtocol struct {
	RequiredTags []string `json:"required_tags"` // Tags every message must carry
	DefaultTTL   int      `json:"default_ttl"`   // Message freshness in seconds
}
