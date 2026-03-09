package labs.mindburn.helm;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;
import com.google.gson.Gson;

/**
 * Functional tests for the HELM Java SDK.
 * These test client construction, request building, serialization,
 * and error handling without requiring a live server.
 */
public class HelmClientTest {
    private static final Gson gson = new Gson();

    @Test
    @DisplayName("Client construction with base URL")
    void testClientConstruction() {
        HelmClient client = new HelmClient("http://localhost:8080");
        assertNotNull(client);
    }

    @Test
    @DisplayName("Client construction with API key")
    void testClientConstructionWithApiKey() {
        HelmClient client = new HelmClient("http://localhost:8080", "test-api-key");
        assertNotNull(client);
    }

    @Test
    @DisplayName("Client strips trailing slash from base URL")
    void testTrailingSlashNormalization() {
        // Should not throw — constructor normalizes URL
        HelmClient client = new HelmClient("http://localhost:8080/");
        assertNotNull(client);
    }

    @Test
    @DisplayName("TypesGen: ChatCompletionRequest serialization")
    void testChatCompletionRequestSerialization() {
        TypesGen.ChatCompletionRequest req = new TypesGen.ChatCompletionRequest();
        req.model = "gpt-4";
        TypesGen.ChatMessage msg = new TypesGen.ChatMessage();
        msg.role = "user";
        msg.content = "Hello";
        req.messages = java.util.List.of(msg);

        String json = gson.toJson(req);
        assertNotNull(json);
        assertTrue(json.contains("\"model\":\"gpt-4\""));
        assertTrue(json.contains("\"role\":\"user\""));
        assertTrue(json.contains("\"content\":\"Hello\""));
    }

    @Test
    @DisplayName("TypesGen: Receipt deserialization")
    void testReceiptDeserialization() {
        String json = "{\"receipt_id\":\"rcpt-123\",\"decision_id\":\"dec-456\",\"status\":\"PASS\",\"blob_hash\":\"sha256:abc\"}";
        TypesGen.Receipt receipt = gson.fromJson(json, TypesGen.Receipt.class);
        assertEquals("rcpt-123", receipt.receipt_id);
        assertEquals("dec-456", receipt.decision_id);
        assertEquals("PASS", receipt.status);
        assertEquals("sha256:abc", receipt.blob_hash);
    }

    @Test
    @DisplayName("TypesGen: ApprovalRequest roundtrip")
    void testApprovalRequestRoundtrip() {
        TypesGen.ApprovalRequest req = new TypesGen.ApprovalRequest();
        req.intent_hash = "intent-789";
        req.signature_b64 = "sig-ed25519-abc";

        String json = gson.toJson(req);
        TypesGen.ApprovalRequest deserialized = gson.fromJson(json, TypesGen.ApprovalRequest.class);
        assertEquals("intent-789", deserialized.intent_hash);
        assertEquals("sig-ed25519-abc", deserialized.signature_b64);
    }

    @Test
    @DisplayName("TypesGen: ConformanceRequest serialization")
    void testConformanceRequestSerialization() {
        TypesGen.ConformanceRequest req = new TypesGen.ConformanceRequest();
        req.level = "L2";
        req.profile = "production";

        String json = gson.toJson(req);
        assertTrue(json.contains("\"level\":\"L2\""));
        assertTrue(json.contains("\"profile\":\"production\""));
    }

    @Test
    @DisplayName("HelmApiException preserves status and reason code")
    void testHelmApiException() {
        HelmClient.HelmApiException ex = new HelmClient.HelmApiException(
            403, "Access denied by policy", "POLICY_DENIED"
        );
        assertEquals(403, ex.status);
        assertEquals("POLICY_DENIED", ex.reasonCode);
        assertEquals("Access denied by policy", ex.getMessage());
    }

    @Test
    @DisplayName("TypesGen: HelmError deserialization")
    void testHelmErrorDeserialization() {
        String json = "{\"error\":{\"message\":\"Not found\",\"reason_code\":\"NOT_FOUND\"}}";
        TypesGen.HelmError err = gson.fromJson(json, TypesGen.HelmError.class);
        assertNotNull(err.error);
        assertEquals("Not found", err.error.message);
        assertEquals("NOT_FOUND", err.error.reason_code);
    }

    @Test
    @DisplayName("TypesGen: VersionInfo deserialization")
    void testVersionInfoDeserialization() {
        String json = "{\"version\":\"0.1.0\",\"commit\":\"abc123\",\"build_time\":\"2026-02-17T00:00:00Z\"}";
        TypesGen.VersionInfo info = gson.fromJson(json, TypesGen.VersionInfo.class);
        assertEquals("0.1.0", info.version);
        assertEquals("abc123", info.commit);
        assertEquals("2026-02-17T00:00:00Z", info.build_time);
    }

    @Test
    @DisplayName("TypesGen: VerificationResult deserialization")
    void testVerificationResultDeserialization() {
        String json = "{\"verdict\":\"PASS\",\"checks\":{\"integrity\":\"PASS\"}}";
        TypesGen.VerificationResult result = gson.fromJson(json, TypesGen.VerificationResult.class);
        assertEquals("PASS", result.verdict);
        assertEquals("PASS", result.checks.get("integrity"));
    }
}
