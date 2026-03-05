package types

// EpigeneticState represents context-dependent genome expression control (T2-1).
// Analogous to DNA methylation, histone acetylation, and genomic imprinting.
//
// Epigenetics in biology: modifications that affect gene expression without
// changing the DNA sequence. In OrgDNA: organizational history and context
// that affect which modules are active and how strongly they express.
type EpigeneticState struct {
	// Methylation maps module types to their methylation level (0.0 = fully expressed, 1.0 = fully silenced).
	// Methylation accumulates over organizational history — heavily regulated modules get increasingly methylated.
	Methylation map[string]float64 `json:"methylation,omitempty"`

	// Acetylation maps module types to their acetylation level (0.0 = closed, 1.0 = fully accessible).
	// Acetylation opens up modules for rapid expression — used during growth phases.
	Acetylation map[string]float64 `json:"acetylation,omitempty"`

	// Imprinting records which modules were inherited from a parent genome and
	// whether the parent's expression state should be preserved.
	Imprinting []ImprintRecord `json:"imprinting,omitempty"`

	// History tracks key organizational events that shape epigenetic marks.
	History []EpigeneticEvent `json:"history,omitempty"`
}

// ImprintRecord captures a parent genome expression inheritance.
type ImprintRecord struct {
	ModuleType   string `json:"module_type"`
	ParentSource string `json:"parent_source"` // "parent_a" or "parent_b" (for M&A)
	Locked       bool   `json:"locked"`        // If true, expression state cannot be changed
}

// EpigeneticEvent records an organizational event that created epigenetic marks.
type EpigeneticEvent struct {
	EventType      string  `json:"event_type"` // "regulatory_action", "market_shock", "growth_phase", "contraction"
	Timestamp      string  `json:"timestamp"`
	AffectedModule string  `json:"affected_module,omitempty"`
	MarkType       string  `json:"mark_type"` // "methylation", "acetylation", "imprint"
	Magnitude      float64 `json:"magnitude"` // How much the mark changed
}

// ChromatinState controls genome region accessibility (T2-2).
// Analogous to euchromatin (open, expressed) vs heterochromatin (closed, silenced).
type ChromatinState struct {
	// Regions maps genome paths to their access state.
	Regions map[string]AccessState `json:"regions"`

	// RemodelingRules define conditions under which regions open or close.
	RemodelingRules []ChromatinRemodelingRule `json:"remodeling_rules,omitempty"`
}

// AccessState represents the accessibility of a genome region.
type AccessState string

const (
	// Euchromatin — open, actively expressed.
	Euchromatin AccessState = "euchromatin"
	// Heterochromatin — constitutively closed, never expressed in this context.
	Heterochromatin AccessState = "heterochromatin"
	// FacultativeHeterochromatin — conditionally closed, can be reopened by remodeling.
	FacultativeHeterochromatin AccessState = "facultative"
)

// ChromatinRemodelingRule defines when a genome region's accessibility changes.
type ChromatinRemodelingRule struct {
	RuleID    string      `json:"rule_id"`
	Target    string      `json:"target"`    // Genome path to remodel
	Condition string      `json:"condition"` // CEL expression
	NewState  AccessState `json:"new_state"` // State to transition to
}

// ModulePromoter controls when, how strongly, and under what conditions
// a module activates (T2-3). Replaces binary enabled/disabled with a
// multi-signal activation model.
//
// Biology: Gene promoters are DNA regions upstream of genes that control
// transcription initiation. They have core, proximal, and distal regulatory elements.
type ModulePromoter struct {
	// CoreSignal is the primary activation condition (CEL expression).
	// Must evaluate to true for the module to activate at all.
	CoreSignal string `json:"core_signal"`

	// Strength controls expression intensity (0.0 = minimal, 1.0 = full).
	// Analogous to promoter strength in biology.
	Strength float64 `json:"strength"`

	// ProximalTuners are nearby regulatory elements that fine-tune expression.
	ProximalTuners []ProximalTuner `json:"proximal_tuners,omitempty"`

	// TissueSpecificity limits this module to specific org units.
	// Empty = ubiquitous (expressed in all org units).
	TissueSpecificity []string `json:"tissue_specificity,omitempty"`
}

// ProximalTuner is a fine-tuning regulatory element near the module.
type ProximalTuner struct {
	TunerID   string  `json:"tuner_id"`
	Condition string  `json:"condition"` // CEL expression
	Effect    float64 `json:"effect"`    // Multiplier on strength (-1.0 to +1.0)
}

// Enhancer is a remote activation booster (T2-4).
// One module can remotely boost another's expression from anywhere in the hierarchy.
type Enhancer struct {
	EnhancerID   string  `json:"enhancer_id"`
	SourceModule string  `json:"source_module"` // Module providing the boost
	TargetModule string  `json:"target_module"` // Module being boosted
	BoostFactor  float64 `json:"boost_factor"`  // Multiplier (> 1.0 = boost)
	Condition    string  `json:"condition"`     // CEL expression for when this enhancer is active
	MaxDistance  int     `json:"max_distance"`  // Maximum org hierarchy distance (0 = unlimited)
}

// Silencer is a remote suppression element (T2-5).
// One module can remotely suppress another's expression.
type Silencer struct {
	SilencerID      string  `json:"silencer_id"`
	SourceModule    string  `json:"source_module"`
	TargetModule    string  `json:"target_module"`
	RepressionLevel float64 `json:"repression_level"` // 0.0 = no effect, 1.0 = full silence
	Condition       string  `json:"condition"`
}

// MorphogenScope adds spatial scoping to morphogenesis rules (T2-6).
// Analogous to morphogen gradients in embryonic development.
type MorphogenScope struct {
	// UnitPath is the org unit this scope applies to.
	UnitPath string `json:"unit_path"`

	// GradientFunction defines how the effect diminishes with distance.
	// "linear", "exponential", "step"
	GradientFunction string `json:"gradient_function"`

	// DiffusionCoefficient controls how far the effect reaches.
	// Higher = further reach, lower = more localized.
	DiffusionCoefficient float64 `json:"diffusion_coefficient"`

	// SourcePosition is the origin point of the gradient.
	SourcePosition string `json:"source_position"`
}

// GenomeRegulation extends the genome with all DNA-level regulatory elements.
// This is the unified container for all T2 regulatory mechanisms.
type GenomeRegulation struct {
	Epigenetics     *EpigeneticState `json:"epigenetics,omitempty"`      // T2-1
	Chromatin       *ChromatinState  `json:"chromatin,omitempty"`        // T2-2
	Promoters       []ModulePromoter `json:"promoters,omitempty"`        // T2-3
	Enhancers       []Enhancer       `json:"enhancers,omitempty"`        // T2-4
	Silencers       []Silencer       `json:"silencers,omitempty"`        // T2-5
	MorphogenFields []MorphogenScope `json:"morphogen_fields,omitempty"` // T2-6
}
