// AUTO-GENERATED from api/openapi/helm.openapi.yaml — DO NOT EDIT
// Regenerate: bash scripts/sdk/gen.sh

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Deterministic reason codes returned by the kernel.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum ReasonCode {
    #[serde(rename = "ALLOW")]
    Allow,
    #[serde(rename = "DENY_TOOL_NOT_FOUND")]
    DenyToolNotFound,
    #[serde(rename = "DENY_SCHEMA_MISMATCH")]
    DenySchemaMismatch,
    #[serde(rename = "DENY_OUTPUT_DRIFT")]
    DenyOutputDrift,
    #[serde(rename = "DENY_BUDGET_EXCEEDED")]
    DenyBudgetExceeded,
    #[serde(rename = "DENY_APPROVAL_REQUIRED")]
    DenyApprovalRequired,
    #[serde(rename = "DENY_APPROVAL_TIMEOUT")]
    DenyApprovalTimeout,
    #[serde(rename = "DENY_SANDBOX_TRAP")]
    DenySandboxTrap,
    #[serde(rename = "DENY_GAS_EXHAUSTION")]
    DenyGasExhaustion,
    #[serde(rename = "DENY_TIME_LIMIT")]
    DenyTimeLimit,
    #[serde(rename = "DENY_MEMORY_LIMIT")]
    DenyMemoryLimit,
    #[serde(rename = "DENY_POLICY_VIOLATION")]
    DenyPolicyViolation,
    #[serde(rename = "DENY_TRUST_KEY_REVOKED")]
    DenyTrustKeyRevoked,
    #[serde(rename = "DENY_IDEMPOTENCY_DUPLICATE")]
    DenyIdempotencyDuplicate,
    #[serde(rename = "ERROR_INTERNAL")]
    ErrorInternal,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HelmErrorDetail {
    pub message: String,
    #[serde(rename = "type")]
    pub error_type: String,
    pub code: String,
    pub reason_code: ReasonCode,
    pub details: Option<HashMap<String, serde_json::Value>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HelmError {
    pub error: HelmErrorDetail,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatMessage {
    pub role: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_call_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolFunction {
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parameters: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tool {
    #[serde(rename = "type")]
    pub tool_type: String,
    pub function: ToolFunction,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatCompletionRequest {
    pub model: String,
    pub messages: Vec<ChatMessage>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tools: Option<Vec<Tool>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub temperature: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_tokens: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stream: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolCallFn {
    pub name: String,
    pub arguments: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolCall {
    pub id: String,
    #[serde(rename = "type")]
    pub call_type: String,
    pub function: ToolCallFn,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChoiceMessage {
    pub role: String,
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_calls: Option<Vec<ToolCall>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Choice {
    pub index: u32,
    pub message: ChoiceMessage,
    pub finish_reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Usage {
    pub prompt_tokens: u32,
    pub completion_tokens: u32,
    pub total_tokens: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatCompletionResponse {
    pub id: String,
    pub object: String,
    pub created: i64,
    pub model: String,
    pub choices: Vec<Choice>,
    pub usage: Option<Usage>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequest {
    pub intent_hash: String,
    pub signature_b64: String,
    pub public_key_b64: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub challenge_response: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Receipt {
    pub receipt_id: String,
    pub decision_id: String,
    pub effect_id: String,
    pub status: String,
    pub reason_code: String,
    pub output_hash: String,
    pub blob_hash: String,
    pub prev_hash: String,
    pub lamport_clock: i64,
    pub signature: String,
    pub timestamp: String,
    pub principal: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Session {
    pub session_id: String,
    pub created_at: String,
    pub receipt_count: u32,
    pub last_lamport_clock: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResult {
    pub verdict: String,
    pub checks: HashMap<String, String>,
    pub errors: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConformanceRequest {
    pub level: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub profile: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConformanceResult {
    pub report_id: String,
    pub level: String,
    pub verdict: String,
    pub gates: u32,
    pub failed: u32,
    pub details: Option<HashMap<String, String>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VersionInfo {
    pub version: String,
    pub commit: String,
    pub build_time: String,
    pub go_version: String,
}
