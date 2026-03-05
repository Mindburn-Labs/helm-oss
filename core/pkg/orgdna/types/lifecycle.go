package types

// T5 — Lifecycle, Signaling, Self-Healing, Death
// Maps cell cycle, signaling cascades, DNA repair, and programmed death to organizational lifecycle.

// LifecycleConfig is the unified container for all lifecycle management subsystems.
type LifecycleConfig struct {
	Phase           LifecyclePhase        `json:"phase"`                      // T5-1: Current phase
	Checkpoints     []LifecycleCheckpoint `json:"checkpoints,omitempty"`      // T5-2
	DivisionConfig  *DivisionConfig       `json:"division_config,omitempty"`  // T5-3/T5-4
	SignalCascades  []SignalCascade       `json:"signal_cascades,omitempty"`  // T5-5
	ParacrineRules  []ParacrineRule       `json:"paracrine_rules,omitempty"`  // T5-6
	JuxtacrineLinks []JuxtacrineLink      `json:"juxtacrine_links,omitempty"` // T5-7
	RepairSystem    *RepairSystem         `json:"repair_system,omitempty"`    // T5-8
	DeathConfig     *DeathConfig          `json:"death_config,omitempty"`     // T5-9/T5-10/T5-11
	Autopoietic     *AutopoieticConfig    `json:"autopoietic,omitempty"`      // T5-12
}

// LifecyclePhase represents the current organizational lifecycle phase (T5-1).
// Maps to cell cycle phases: G0 (quiescence), G1 (growth), S (revision), G2 (pre-division), M (division).
type LifecyclePhase string

const (
	PhaseG0 LifecyclePhase = "G0" // Quiescence — operational, not growing
	PhaseG1 LifecyclePhase = "G1" // Growth — expanding capabilities, hiring
	PhaseS  LifecyclePhase = "S"  // Synthesis — genome revision/replication
	PhaseG2 LifecyclePhase = "G2" // Pre-Division — preparing for split
	PhaseM  LifecyclePhase = "M"  // Mitosis — actively dividing
)

// LifecycleCheckpoint is a quality gate between lifecycle phases (T5-2).
type LifecycleCheckpoint struct {
	CheckpointID string         `json:"checkpoint_id"`
	FromPhase    LifecyclePhase `json:"from_phase"`
	ToPhase      LifecyclePhase `json:"to_phase"`
	Condition    string         `json:"condition"` // CEL expression
	Description  string         `json:"description"`
}

// DivisionConfig handles organizational division (T5-3) and M&A preparation (T5-4).
type DivisionConfig struct {
	// Mitosis: create subsidiaries/spin-offs.
	MitosisPlan *MitosisPlan `json:"mitosis_plan,omitempty"`
	// Meiosis: prepare org contribution for merger.
	MeiosisPlan *MeiosisPlan `json:"meiosis_plan,omitempty"`
}

// MitosisPlan specifies how to create a subsidiary (T5-3).
type MitosisPlan struct {
	GenomeForkStrategy string             `json:"genome_fork_strategy"` // "full", "selective", "minimal"
	AssetAllocation    map[string]float64 `json:"asset_allocation"`     // Module → fraction
	LegalEntity        string             `json:"legal_entity"`         // Type of legal entity to create
}

// MeiosisPlan specifies how to prepare a genome subset for merger (T5-4).
type MeiosisPlan struct {
	ContributionModules []string `json:"contribution_modules"` // Modules to contribute
	RetainModules       []string `json:"retain_modules"`       // Modules to keep
	CrossoverPoints     []string `json:"crossover_points"`     // Where genomes can recombine
}

// SignalCascade defines a signal transduction cascade (T5-5).
// External event → receptor → first messenger → second messengers → amplified response.
type SignalCascade struct {
	CascadeID           string   `json:"cascade_id"`
	Receptor            string   `json:"receptor"`          // External signal receptor
	FirstMessenger      string   `json:"first_messenger"`   // External event type
	SecondMessengers    []string `json:"second_messengers"` // Internal amplified signals
	AmplificationFactor float64  `json:"amplification_factor"`
	TargetFactories     []string `json:"target_factories"` // Factories to activate
	DecayRate           float64  `json:"decay_rate"`       // How quickly the signal fades
}

// ParacrineRule defines short-range inter-factory communication (T5-6).
type ParacrineRule struct {
	RuleID     string `json:"rule_id"`
	Source     string `json:"source"`      // Source factory
	Target     string `json:"target"`      // Target factory (must be in same org unit)
	SignalType string `json:"signal_type"` // Type of signal
	Scope      string `json:"scope"`       // "same_unit", "adjacent_unit"
}

// JuxtacrineLink defines direct module-to-module coupling (T5-7).
type JuxtacrineLink struct {
	LinkID   string `json:"link_id"`
	ModuleA  string `json:"module_a"`
	ModuleB  string `json:"module_b"`
	Protocol string `json:"protocol"` // "state_share", "direct_call", "mutex_lock"
}

// RepairSystem implements organizational self-healing (T5-8).
// Maps DNA repair mechanisms: BER (minor), NER (structural), NHEJ (emergency), HR (precise).
type RepairSystem struct {
	BER  *BERConfig  `json:"ber,omitempty"`  // Base Excision Repair — minor hot-patches
	NER  *NERConfig  `json:"ner,omitempty"`  // Nucleotide Excision Repair — structural
	NHEJ *NHEJConfig `json:"nhej,omitempty"` // Non-Homologous End Joining — emergency lossy
	HR   *HRConfig   `json:"hr,omitempty"`   // Homologous Recombination — precise from backup
}

// BERConfig — Minor hot-patch repair.
type BERConfig struct {
	AutoPatchEnabled bool     `json:"auto_patch_enabled"`
	PatchableModules []string `json:"patchable_modules,omitempty"` // Module types that can be auto-patched
}

// NERConfig — Structural repair for larger issues.
type NERConfig struct {
	ScanInterval string `json:"scan_interval"` // How often to scan for structural damage
	RepairPolicy string `json:"repair_policy"` // "automatic", "manual_approval", "hybrid"
}

// NHEJConfig — Emergency lossy repair (may lose data but restores operation).
type NHEJConfig struct {
	MaxDataLoss  float64 `json:"max_data_loss"` // Maximum acceptable data loss (0.0-1.0)
	FallbackMode string  `json:"fallback_mode"` // "degraded", "minimal", "readonly"
}

// HRConfig — Precise repair using backup genome as template.
type HRConfig struct {
	BackupGenomeRef   string `json:"backup_genome_ref"`   // Reference to backup genome
	VerifyAfterRepair bool   `json:"verify_after_repair"` // Re-compile and verify after repair
}

// DeathConfig handles organizational shutdown and legacy (T5-9, T5-10, T5-11).
type DeathConfig struct {
	// Extrinsic Apoptosis (T5-9): external authority-triggered shutdown.
	DeathReceptors []DeathReceptor `json:"death_receptors,omitempty"`

	// Autophagy (T5-10): crisis-mode resource cannibalization.
	AutophagyConfig *AutophagyConfig `json:"autophagy_config,omitempty"`

	// Senescence (T5-11): legacy mode — stop growing, continue operating.
	SenescenceConfig *SenescenceConfig `json:"senescence_config,omitempty"`
}

// DeathReceptor listens for external kill signals (T5-9).
type DeathReceptor struct {
	ReceptorID     string `json:"receptor_id"`
	AuthorityType  string `json:"authority_type"`  // "regulator", "court", "parent_company"
	ShutdownPolicy string `json:"shutdown_policy"` // "graceful", "immediate"
}

// AutophagyConfig defines crisis-mode resource cannibalization (T5-10).
type AutophagyConfig struct {
	StarvationThreshold float64  `json:"starvation_threshold"` // Energy level that triggers autophagy
	SacrificeOrder      []string `json:"sacrifice_order"`      // Module types to consume, in order
	RecyclingEfficiency float64  `json:"recycling_efficiency"` // How much energy is recovered (0.0-1.0)
}

// SenescenceConfig defines legacy mode behavior (T5-11).
type SenescenceConfig struct {
	StopAcceptingNew   bool   `json:"stop_accepting_new"`   // Stop accepting new work
	EmitTakeoverSignal bool   `json:"emit_takeover_signal"` // Signal neighbors to take over
	MaxLifespan        string `json:"max_lifespan"`         // Maximum time to remain senescent
}

// AutopoieticConfig — Self-production and self-maintenance (T5-12).
// Based on Free Energy Principle: the system actively maintains its own organization.
type AutopoieticConfig struct {
	RequiredComponents []string         `json:"required_components"` // Module types that must be present
	ProductionRules    []ProductionRule `json:"production_rules,omitempty"`
	IntegrityInterval  string           `json:"integrity_interval"` // How often to check integrity
}

// ProductionRule defines how the system self-repairs crashed factories.
type ProductionRule struct {
	RuleID    string `json:"rule_id"`
	Component string `json:"component"` // Module type to produce/repair
	Trigger   string `json:"trigger"`   // CEL condition for when to activate
	Source    string `json:"source"`    // Where to get the blueprint ("phenotype", "genome", "backup")
}
