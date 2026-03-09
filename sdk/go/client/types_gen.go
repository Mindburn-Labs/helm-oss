// AUTO-GENERATED from api/openapi/helm.openapi.yaml — DO NOT EDIT
// Regenerate: bash scripts/sdk/gen.sh

package client

// ReasonCode is a deterministic reason code returned by the kernel.
type ReasonCode string

const (
	ReasonAllow                ReasonCode = "ALLOW"
	ReasonDenyToolNotFound     ReasonCode = "DENY_TOOL_NOT_FOUND"
	ReasonDenySchemaMismatch   ReasonCode = "DENY_SCHEMA_MISMATCH"
	ReasonDenyOutputDrift      ReasonCode = "DENY_OUTPUT_DRIFT"
	ReasonDenyBudgetExceeded   ReasonCode = "DENY_BUDGET_EXCEEDED"
	ReasonDenyApprovalRequired ReasonCode = "DENY_APPROVAL_REQUIRED"
	ReasonDenyApprovalTimeout  ReasonCode = "DENY_APPROVAL_TIMEOUT"
	ReasonDenySandboxTrap      ReasonCode = "DENY_SANDBOX_TRAP"
	ReasonDenyGasExhaustion    ReasonCode = "DENY_GAS_EXHAUSTION"
	ReasonDenyTimeLimit        ReasonCode = "DENY_TIME_LIMIT"
	ReasonDenyMemoryLimit      ReasonCode = "DENY_MEMORY_LIMIT"
	ReasonDenyPolicyViolation  ReasonCode = "DENY_POLICY_VIOLATION"
	ReasonDenyTrustKeyRevoked  ReasonCode = "DENY_TRUST_KEY_REVOKED"
	ReasonDenyIdempotencyDup   ReasonCode = "DENY_IDEMPOTENCY_DUPLICATE"
	ReasonErrorInternal        ReasonCode = "ERROR_INTERNAL"
)

// HelmError is the standard error envelope.
type HelmError struct {
	Error struct {
		Message    string         `json:"message"`
		Type       string         `json:"type"`
		Code       string         `json:"code"`
		ReasonCode ReasonCode     `json:"reason_code"`
		Details    map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

type ChatMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      ChoiceMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type ChoiceMessage struct {
	Role      string     `json:"role"`
	Content   *string    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Function ToolCallFn `json:"function"`
}

type ToolCallFn struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ApprovalRequest struct {
	IntentHash        string `json:"intent_hash"`
	SignatureB64      string `json:"signature_b64"`
	PublicKeyB64      string `json:"public_key_b64"`
	ChallengeResponse string `json:"challenge_response,omitempty"`
}

type Receipt struct {
	ReceiptID    string `json:"receipt_id"`
	DecisionID   string `json:"decision_id"`
	EffectID     string `json:"effect_id"`
	Status       string `json:"status"`
	ReasonCode   string `json:"reason_code"`
	OutputHash   string `json:"output_hash"`
	BlobHash     string `json:"blob_hash"`
	PrevHash     string `json:"prev_hash"`
	LamportClock int    `json:"lamport_clock"`
	Signature    string `json:"signature"`
	Timestamp    string `json:"timestamp"`
	Principal    string `json:"principal"`
}

type Session struct {
	SessionID        string `json:"session_id"`
	CreatedAt        string `json:"created_at"`
	ReceiptCount     int    `json:"receipt_count"`
	LastLamportClock int    `json:"last_lamport_clock"`
}

type ExportRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Format    string `json:"format,omitempty"`
}

type VerificationResult struct {
	Verdict string            `json:"verdict"`
	Checks  map[string]string `json:"checks"`
	Errors  []string          `json:"errors"`
}

type ConformanceRequest struct {
	Level   string `json:"level"`
	Profile string `json:"profile,omitempty"`
}

type ConformanceResult struct {
	ReportID string            `json:"report_id"`
	Level    string            `json:"level"`
	Verdict  string            `json:"verdict"`
	Gates    int               `json:"gates"`
	Failed   int               `json:"failed"`
	Details  map[string]string `json:"details,omitempty"`
}

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}
