package types

// RNARegulation contains the fine-grained regulatory layer between genome and
// phenotype (T3). Maps every major RNA type to organizational control mechanisms.
type RNARegulation struct {
	MicroRules          []PolicyMicroRule    `json:"micro_rules,omitempty"`          // T3-1: miRNA analog
	KillSwitches        []KillSwitch         `json:"kill_switches,omitempty"`        // T3-2: siRNA analog
	GovernanceScaffolds []GovernanceScaffold `json:"governance_scaffolds,omitempty"` // T3-3: lncRNA analog
	CapabilityAdapters  []CapabilityAdapter  `json:"capability_adapters,omitempty"`  // T3-4: tRNA deepening
	CommitmentRules     []CommitmentRule     `json:"commitment_rules,omitempty"`     // T3-5: Waddington lifecycle
}

// PolicyMicroRule is a short targeted rule that attenuates specific module behaviors (T3-1).
// Analogous to microRNA (miRNA) — small regulatory elements that bind to and suppress targets.
type PolicyMicroRule struct {
	RuleID      string  `json:"rule_id"`
	Target      string  `json:"target"`       // Target module type
	BindingSite string  `json:"binding_site"` // Specific aspect of the module to affect
	Effect      string  `json:"effect"`       // "suppress", "attenuate", "redirect"
	Strength    float64 `json:"strength"`     // 0.0-1.0
	Condition   string  `json:"condition"`    // CEL expression for when this rule is active
}

// KillSwitch provides rapid targeted module shutdown (T3-2).
// Analogous to siRNA / RNA interference — transcript destruction for immediate silencing.
type KillSwitch struct {
	SwitchID     string `json:"switch_id"`
	TargetModule string `json:"target_module"` // Module to shut down
	TriggerCEL   string `json:"trigger_cel"`   // CEL condition that activates the kill switch
	Reversible   bool   `json:"reversible"`    // Can the module be reactivated?
	AuditReason  string `json:"audit_reason"`  // Why this kill switch exists
}

// GovernanceScaffold is a non-operational structure that organizes regulatory interactions (T3-3).
// Analogous to long non-coding RNA (lncRNA) — doesn't produce operational output but shapes regulation.
type GovernanceScaffold struct {
	ScaffoldID string   `json:"scaffold_id"`
	Type       string   `json:"type"`                // "scaffold", "guide", "decoy"
	Organizes  []string `json:"organizes,omitempty"` // Module types this scaffold organizes
	Purpose    string   `json:"purpose"`
}

// CapabilityAdapter translates abstract capabilities to concrete implementations (T3-4).
// Deepening of tRNA: capability → adapter → implementation chain.
type CapabilityAdapter struct {
	AdapterID        string  `json:"adapter_id"`
	Capability       string  `json:"capability"`        // Abstract capability name
	Implementation   string  `json:"implementation"`    // Concrete implementation
	Specificity      float64 `json:"specificity"`       // 0.0-1.0, how well this adapter matches
	WobbleCompatible bool    `json:"wobble_compatible"` // Can handle approximate matches
}

// CommitmentRule controls Waddington module commitment lifecycle (T3-5).
// Modules progress: pluripotent → determined → differentiated → senescent.
type CommitmentRule struct {
	RuleID       string `json:"rule_id"`
	ModuleType   string `json:"module_type"`
	FromState    string `json:"from_state"`   // Current commitment state
	ToState      string `json:"to_state"`     // Target commitment state
	Condition    string `json:"condition"`    // CEL condition for transition
	Irreversible bool   `json:"irreversible"` // Once committed, can it go back?
}

// CommitmentState tracks a module's position in the Waddington landscape.
type CommitmentState struct {
	State       string `json:"state"`       // "pluripotent", "determined", "differentiated", "senescent"
	Transitions int    `json:"transitions"` // How many state changes have occurred
}
