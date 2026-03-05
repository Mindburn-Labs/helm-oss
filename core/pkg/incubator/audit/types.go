// Package audit provides the canonical audit record interface and types.
//
// All audit subsystems in HELM (store, guardian, crypto, observability)
// MUST implement the Record interface to enable cross-subsystem correlation,
// unified querying, and fan-out via the audit Bus.
package audit

import "time"

// RecordType categorizes audit records across all subsystems.
type RecordType string

const (
	// RecordAccess represents read operations (queries, exports, views).
	RecordAccess RecordType = "ACCESS"
	// RecordMutation represents write operations (create, update, delete).
	RecordMutation RecordType = "MUTATION"
	// RecordSystem represents system-level events (startup, shutdown, health).
	RecordSystem RecordType = "SYSTEM"
	// RecordPolicy represents policy evaluation events (admit, deny, escalate).
	RecordPolicy RecordType = "POLICY"
	// RecordEvidence represents evidence artifacts (bundles, attestations).
	RecordEvidence RecordType = "EVIDENCE"
	// RecordSecurity represents security events (auth failure, secret rotation).
	RecordSecurity RecordType = "SECURITY"
)

// Record is the canonical audit event interface.
//
// All audit subsystems MUST implement this to enable:
//   - Cross-subsystem correlation via GetID()
//   - Unified timeline queries via GetTimestamp() + GetType()
//   - Fan-out via the audit Bus
//   - Evidence chain verification via GetHash()
type Record interface {
	// GetID returns the unique identifier for this audit record.
	GetID() string
	// GetTimestamp returns when the event occurred.
	GetTimestamp() time.Time
	// GetActor returns the principal that caused the event.
	GetActor() string
	// GetAction returns the action that was performed.
	GetAction() string
	// GetResource returns the resource that was affected.
	GetResource() string
	// GetType returns the category of the audit event.
	GetType() RecordType
	// GetHash returns the content hash. Empty string if not hashed.
	GetHash() string
}

// MissionSpec declares a single audit mission that must be executed.
type MissionSpec struct {
	ID          string `json:"id" yaml:"id"`
	Required    bool   `json:"required" yaml:"required"`
	Description string `json:"description" yaml:"description"`
}

// MissionManifest declares the full set of required audit missions.
type MissionManifest struct {
	Missions []MissionSpec `json:"missions" yaml:"missions"`
}

// MissionEvidence records that a specific mission was executed.
type MissionEvidence struct {
	MissionID     string    `json:"mission_id"`
	Timestamp     time.Time `json:"timestamp"`
	Model         string    `json:"model"`
	FindingCount  int       `json:"finding_count"`
	Severity      string    `json:"severity"`
	CoverageScore float64   `json:"coverage_score"`
	ContentHash   string    `json:"content_hash"`
}

// MergedAuditReport is the unified final report combining mechanical + AI audit layers.
type MergedAuditReport struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	GitSHA    string    `json:"git_sha"`

	Mechanical   MechanicalSummary  `json:"mechanical"`
	AI           AISummary          `json:"ai"`
	Completeness CompletenessResult `json:"completeness"`

	Verdict    string `json:"verdict"` // "COMPLIANT" or "NOT_COMPLIANT"
	ReportHash string `json:"report_hash"`
}

// MechanicalSummary captures the mechanical audit layer results.
type MechanicalSummary struct {
	Sections int               `json:"sections"`
	Pass     int               `json:"pass"`
	Fail     int               `json:"fail"`
	Warn     int               `json:"warn"`
	Skip     int               `json:"skip"`
	Details  []SectionEvidence `json:"details"`
}

// SectionEvidence is the evidence for a single mechanical audit section.
type SectionEvidence struct {
	Section   string `json:"section"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	Timestamp string `json:"timestamp"`
	GitSHA    string `json:"git_sha"`
}

// AISummary captures the AI audit layer results.
type AISummary struct {
	Model          string            `json:"model"`
	Missions       int               `json:"missions"`
	Completed      int               `json:"completed"`
	Findings       []AIFinding       `json:"findings"`
	CoverageScore  float64           `json:"coverage_score"`
	MissionResults []MissionEvidence `json:"mission_results"`
}

// AIFinding is a single finding from the AI audit layer.
type AIFinding struct {
	MissionID      string `json:"mission_id"`
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	File           string `json:"file,omitempty"`
	Line           int    `json:"line,omitempty"`
	Description    string `json:"description"`
	Recommendation string `json:"recommendation,omitempty"`
}

// CompletenessResult captures whether all required missions were executed.
type CompletenessResult struct {
	AllMissionsRan       bool     `json:"all_missions_ran"`
	MissionChainVerified bool     `json:"mission_chain_verified"`
	ChainHead            string   `json:"chain_head"`
	MissingMissions      []string `json:"missing_missions,omitempty"`
	TotalRequired        int      `json:"total_required"`
	TotalCompleted       int      `json:"total_completed"`
}
