package types

import "time"

// OrgRNA represents the intermediate representation (IR) between the OrgGenome
// (DNA) and OrgPhenotype (Protein). Analogous to messenger RNA (mRNA) in biology.
//
// Biology: DNA → (transcription) → mRNA → (translation) → Protein
// OrgDNA: OrgGenome → (Stage 3.5: Transcribe) → OrgRNA → (Stage 4: Morphogenesis) → OrgPhenotype
//
// The OrgRNA captures which parts of the genome are active for a given context,
// supporting alternative splicing (same genome → different phenotypes).
type OrgRNA struct {
	// TranscriptID uniquely identifies this transcript.
	TranscriptID string `json:"transcript_id"`

	// SourceGenomeHash is the CSNF hash of the source genome at transcription time.
	SourceGenomeHash string `json:"source_genome_hash"`

	// SpliceVariant identifies which context-dependent variant was selected.
	// Examples: "startup", "regulated", "crisis", "growth", "default"
	SpliceVariant string `json:"splice_variant"`

	// ExonSet contains the genome sections that ARE included in this transcript.
	// These are the "expressed" parts of the genome for this variant.
	ExonSet []GenomeExon `json:"exon_set"`

	// IntronSet contains the genome sections that are EXCLUDED (spliced out)
	// for this variant. They exist in the genome but are not expressed.
	IntronSet []GenomeIntron `json:"intron_set"`

	// UTR5 (5' Untranslated Region) — metadata and regulatory signals
	// that precede the coding region. Maps to pre-operational governance.
	UTR5 *UTRRegion `json:"utr5,omitempty"`

	// UTR3 (3' Untranslated Region) — stability signals and decay control.
	// Maps to lifecycle and decommissioning instructions.
	UTR3 *UTRRegion `json:"utr3,omitempty"`

	// Stability represents the half-life of this transcript.
	// Higher stability = longer-lived phenotype before recompilation needed.
	// Range: 0.0 (immediate decay) to 1.0 (permanent).
	Stability float64 `json:"stability"`

	// TranscribedAt is the deterministic timestamp of transcription.
	TranscribedAt time.Time `json:"transcribed_at"`

	// Modifications tracks post-transcriptional modifications (PTM analogs).
	Modifications []PostCompilationModification `json:"modifications,omitempty"`
}

// GenomeExon represents an included section of the genome.
// Exons are the "coding" parts — they get translated into operational phenotype.
type GenomeExon struct {
	// ExonID uniquely identifies this exon within the transcript.
	ExonID string `json:"exon_id"`

	// SourcePath identifies which genome element this exon comes from.
	// Format: "{section}.{element}" e.g. "modules[0]", "regulation.essential_variables[2]"
	SourcePath string `json:"source_path"`

	// Content is the resolved genome fragment for this exon.
	Content interface{} `json:"content"`

	// Order determines the position of this exon in the final transcript.
	Order int `json:"order"`
}

// GenomeIntron represents an excluded section of the genome.
// Introns are spliced out for this variant but remain in the genome.
type GenomeIntron struct {
	// IntronID uniquely identifies this intron.
	IntronID string `json:"intron_id"`

	// SourcePath identifies which genome element was excluded.
	SourcePath string `json:"source_path"`

	// Reason explains why this section was excluded for this variant.
	Reason string `json:"reason"`
}

// UTRRegion represents an untranslated region of the transcript.
type UTRRegion struct {
	// Signals are regulatory/control signals in this UTR.
	Signals []UTRSignal `json:"signals"`
}

// UTRSignal is a regulatory signal embedded in the UTR.
type UTRSignal struct {
	// Type categorizes the signal.
	// UTR5 types: "ribosome_binding" (startup priority), "kozak" (translation efficiency)
	// UTR3 types: "poly_a" (stability), "au_rich" (rapid decay), "mirna_target" (regulation)
	Type string `json:"type"`

	// Value is the signal's parameter.
	Value interface{} `json:"value,omitempty"`
}

// SpliceVariantConfig defines how a specific splice variant is selected and applied.
type SpliceVariantConfig struct {
	// VariantID is the unique identifier for this splice variant.
	VariantID string `json:"variant_id"`

	// Condition is a CEL expression evaluated against the EnvironmentProfile
	// to determine if this variant should be used.
	Condition string `json:"condition"`

	// Priority determines which variant wins if multiple conditions match.
	// Higher = higher priority.
	Priority int `json:"priority"`

	// ExcludeModules lists module types to splice OUT for this variant.
	ExcludeModules []string `json:"exclude_modules,omitempty"`

	// ExcludeRegulation lists regulation IDs to splice OUT.
	ExcludeRegulation []string `json:"exclude_regulation,omitempty"`

	// IncludeOnly lists module types to EXCLUSIVELY include (all others excluded).
	// Mutually exclusive with ExcludeModules.
	IncludeOnly []string `json:"include_only,omitempty"`

	// StabilityOverride overrides the default transcript stability.
	StabilityOverride *float64 `json:"stability_override,omitempty"`
}

// PostCompilationModification represents a runtime modification to the phenotype
// without triggering a full recompile. Analogous to post-translational modifications
// (PTM) in biology: phosphorylation, glycosylation, ubiquitination.
type PostCompilationModification struct {
	// ModificationID uniquely identifies this modification.
	ModificationID string `json:"modification_id"`

	// Type categorizes the modification.
	// "phosphorylation" — activate/deactivate a module (reversible)
	// "glycosylation"   — tag a module for external visibility/routing
	// "ubiquitination"  — mark a module for deprecation/recycling
	// "sumoylation"     — modify module's regulatory behavior
	// "acetylation"     — change module's accessibility/permissions
	Type string `json:"type"`

	// TargetModule identifies which module is being modified.
	TargetModule string `json:"target_module"`

	// Effect describes the specific effect of this modification.
	Effect PTMEffect `json:"effect"`

	// AppliedAt is the deterministic timestamp of application.
	AppliedAt time.Time `json:"applied_at"`

	// AppliedBy identifies who/what applied this modification.
	AppliedBy string `json:"applied_by"`

	// Reversible indicates if this modification can be undone.
	Reversible bool `json:"reversible"`

	// ReceiptID links to the governance receipt for this modification.
	ReceiptID string `json:"receipt_id,omitempty"`
}

// PTMEffect describes the specific effect of a post-compilation modification.
type PTMEffect struct {
	// Action is the effect type: "activate", "deactivate", "tag", "deprecate",
	// "redirect", "throttle", "boost"
	Action string `json:"action"`

	// Parameters are action-specific parameters.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// PhenotypeDiff captures the differences between two phenotype compilations.
// Generated on every genome evolution to provide audit trail and change documentation.
type PhenotypeDiff struct {
	// DiffID uniquely identifies this diff.
	DiffID string `json:"diff_id"`

	// OldGenomeHash is the CSNF hash of the genome before evolution.
	OldGenomeHash string `json:"old_genome_hash"`

	// NewGenomeHash is the CSNF hash of the genome after evolution.
	NewGenomeHash string `json:"new_genome_hash"`

	// ModulesAdded lists modules present in new but not old.
	ModulesAdded []string `json:"modules_added,omitempty"`

	// ModulesRemoved lists modules present in old but not new.
	ModulesRemoved []string `json:"modules_removed,omitempty"`

	// ModulesModified lists modules that changed between versions.
	ModulesModified []ModuleModification `json:"modules_modified,omitempty"`

	// CapabilitiesGained lists capabilities added.
	CapabilitiesGained []string `json:"capabilities_gained,omitempty"`

	// CapabilitiesLost lists capabilities removed.
	CapabilitiesLost []string `json:"capabilities_lost,omitempty"`

	// RegulationChanged indicates if the regulation config changed.
	RegulationChanged bool `json:"regulation_changed"`

	// RiskScore is a computed risk score for this change (0.0 = safe, 1.0 = critical).
	RiskScore float64 `json:"risk_score"`

	// RequiresApproval indicates if manual approval is needed based on risk.
	RequiresApproval bool `json:"requires_approval"`

	// ComputedAt is the deterministic timestamp.
	ComputedAt time.Time `json:"computed_at"`
}

// ModuleModification describes a specific change to a module.
type ModuleModification struct {
	ModuleType string `json:"module_type"`
	Field      string `json:"field"`
	OldValue   string `json:"old_value"`
	NewValue   string `json:"new_value"`
}

// ConfluenceProof is the artifact produced by the confluence verifier.
// Proves that morphogenesis rule application order does not affect the final result.
type ConfluenceProof struct {
	// ProofID uniquely identifies this proof.
	ProofID string `json:"proof_id"`

	// GenomeHash is the genome this proof applies to.
	GenomeHash string `json:"genome_hash"`

	// PermutationsTested is how many rule orderings were tested.
	PermutationsTested int `json:"permutations_tested"`

	// ResultHash is the phenotype hash that all permutations produced.
	// If nil, confluence failed.
	ResultHash string `json:"result_hash,omitempty"`

	// IsConfluent indicates whether all permutations produced the same result.
	IsConfluent bool `json:"is_confluent"`

	// Violations lists any permutations that produced different results.
	Violations []ConfluenceViolation `json:"violations,omitempty"`

	// ComputedAt is the deterministic timestamp.
	ComputedAt time.Time `json:"computed_at"`
}

// ConfluenceViolation records a specific ordering that produced a different result.
type ConfluenceViolation struct {
	// RuleOrder is the order of rules that caused the violation.
	RuleOrder []string `json:"rule_order"`

	// ResultHash is the phenotype hash produced by this ordering.
	ResultHash string `json:"result_hash"`

	// Diff describes what differs from the canonical result.
	Diff string `json:"diff"`
}
