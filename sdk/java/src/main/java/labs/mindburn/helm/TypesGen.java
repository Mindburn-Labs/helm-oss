// AUTO-GENERATED from api/openapi/helm.openapi.yaml — DO NOT EDIT
// Regenerate: bash scripts/sdk/gen.sh

package labs.mindburn.helm;

import java.util.List;
import java.util.Map;

/** Deterministic reason codes returned by the kernel. */
public final class TypesGen {

    public enum ReasonCode {
        ALLOW,
        DENY_TOOL_NOT_FOUND,
        DENY_SCHEMA_MISMATCH,
        DENY_OUTPUT_DRIFT,
        DENY_BUDGET_EXCEEDED,
        DENY_APPROVAL_REQUIRED,
        DENY_APPROVAL_TIMEOUT,
        DENY_SANDBOX_TRAP,
        DENY_GAS_EXHAUSTION,
        DENY_TIME_LIMIT,
        DENY_MEMORY_LIMIT,
        DENY_POLICY_VIOLATION,
        DENY_TRUST_KEY_REVOKED,
        DENY_IDEMPOTENCY_DUPLICATE,
        ERROR_INTERNAL
    }

    public static class HelmErrorDetail {
        public String message;
        public String type;
        public String code;
        public String reason_code;
        public Map<String, Object> details;
    }

    public static class HelmError {
        public HelmErrorDetail error;
    }

    public static class ChatMessage {
        public String role;
        public String content;
        public String tool_call_id;

        public ChatMessage() {}
        public ChatMessage(String role, String content) {
            this.role = role;
            this.content = content;
        }
    }

    public static class ToolFunction {
        public String name;
        public String description;
        public Map<String, Object> parameters;
    }

    public static class Tool {
        public String type = "function";
        public ToolFunction function;
    }

    public static class ChatCompletionRequest {
        public String model;
        public List<ChatMessage> messages;
        public List<Tool> tools;
        public Double temperature;
        public Integer max_tokens;
        public Boolean stream;
    }

    public static class ToolCallFn {
        public String name;
        public String arguments;
    }

    public static class ToolCall {
        public String id;
        public String type;
        public ToolCallFn function;
    }

    public static class ChoiceMessage {
        public String role;
        public String content;
        public List<ToolCall> tool_calls;
    }

    public static class Choice {
        public int index;
        public ChoiceMessage message;
        public String finish_reason;
    }

    public static class Usage {
        public int prompt_tokens;
        public int completion_tokens;
        public int total_tokens;
    }

    public static class ChatCompletionResponse {
        public String id;
        public String object;
        public long created;
        public String model;
        public List<Choice> choices;
        public Usage usage;
    }

    public static class ApprovalRequest {
        public String intent_hash;
        public String signature_b64;
        public String public_key_b64;
        public String challenge_response;
    }

    public static class Receipt {
        public String receipt_id;
        public String decision_id;
        public String effect_id;
        public String status;
        public String reason_code;
        public String output_hash;
        public String blob_hash;
        public String prev_hash;
        public int lamport_clock;
        public String signature;
        public String timestamp;
        public String principal;
    }

    public static class Session {
        public String session_id;
        public String created_at;
        public int receipt_count;
        public int last_lamport_clock;
    }

    public static class VerificationResult {
        public String verdict;
        public Map<String, String> checks;
        public List<String> errors;
    }

    public static class ConformanceRequest {
        public String level;
        public String profile;

        public ConformanceRequest() {}
        public ConformanceRequest(String level) {
            this.level = level;
            this.profile = "full";
        }
    }

    public static class ConformanceResult {
        public String report_id;
        public String level;
        public String verdict;
        public int gates;
        public int failed;
        public Map<String, String> details;
    }

    public static class VersionInfo {
        public String version;
        public String commit;
        public String build_time;
        public String go_version;
    }

    private TypesGen() {}
}
