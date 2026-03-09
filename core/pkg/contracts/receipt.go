package contracts

import "time"

// Receipt represents a proof of effect execution, linked to a decision.
type Receipt struct {
	ReceiptID           string         `json:"receipt_id"`
	DecisionID          string         `json:"decision_id"`
	EffectID            string         `json:"effect_id"`
	ExternalReferenceID string         `json:"external_reference_id"`
	Status              string         `json:"status"`
	BlobHash            string         `json:"blob_hash,omitempty"`   // Link to Input Snapshot CAS
	OutputHash          string         `json:"output_hash,omitempty"` // Link to Tool Output CAS
	Timestamp           time.Time      `json:"timestamp"`
	ExecutorID          string         `json:"executor_id,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
	Signature           string         `json:"signature,omitempty"` // Cryptographic proof of execution
	// V2: Tamper-Evidence
	MerkleRoot        string             `json:"merkle_root,omitempty"`
	WitnessSignatures []WitnessSignature `json:"witness_signatures,omitempty"`

	// V3: Causal chain – ProofGraph DAG
	PrevHash     string `json:"prev_hash"`           // SHA-256 of the previous receipt's signature (causal link)
	LamportClock uint64 `json:"lamport_clock"`       // Monotonic logical clock per session
	ArgsHash     string `json:"args_hash,omitempty"` // Phase 2: SHA-256 of JCS-canonicalized tool args (PEP boundary)

	// Receipt-as-First-Class Artifact Extensions
	ReplayScript     *ReplayScriptRef   `json:"replay_script,omitempty"`     // Link to deterministic replay script
	Provenance       *ReceiptProvenance `json:"provenance,omitempty"`        // Chain of custody
	BundledArtifacts []ParsedArtifact   `json:"bundled_artifacts,omitempty"` // Hashable bundles of related artifacts
}

// ReplayScriptRef points to the script that can reproduce this receipt's effect.
type ReplayScriptRef struct {
	ScriptID   string `json:"script_id"`
	ScriptHash string `json:"script_hash"`
	Engine     string `json:"engine"` // e.g., "governance-v1", "frontier-adapter-v1"
	Entrypoint string `json:"entrypoint"`
}

// ReceiptProvenance tracks the origin and chain of custody for the receipt.
type ReceiptProvenance struct {
	GeneratedBy string    `json:"generated_by"` // Agent/Component ID
	GeneratedAt time.Time `json:"generated_at"`
	Context     string    `json:"context"`           // e.g., "production", "simulation"
	Parents     []string  `json:"parents,omitempty"` // Parent Receipt IDs used as input
}

// ParsedArtifact represents a hashable bundle of data produced or used.
type ParsedArtifact struct {
	ArtifactID   string `json:"artifact_id"`
	Type         string `json:"type"` // e.g., "file", "db_record", "api_response"
	Hash         string `json:"hash"`
	URIRef       string `json:"uri_ref,omitempty"`       // Where to find it
	Inlinedigest string `json:"inline_digest,omitempty"` // Small data can be inlined
}

type WitnessSignature struct {
	WitnessID string `json:"witness_id"`
	Signature string `json:"signature"`
}
