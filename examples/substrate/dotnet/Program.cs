/// HELM EffectBoundary Substrate Example — C# (.NET)
///
/// Demonstrates implementing the EffectBoundary contract in .NET
/// using ASP.NET Minimal APIs against the OpenAPI spec.

using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var boundary = new EffectBoundary();

app.MapPost("/v1/effects", (EffectRequest req) =>
{
    var result = boundary.Submit(req);
    return Results.Ok(result);
});

app.MapPost("/v1/effects/{receiptId}/complete", (string receiptId, CompletionRequest req) =>
{
    var result = boundary.Complete(receiptId, req);
    return Results.Ok(result);
});

Console.WriteLine("HELM EffectBoundary substrate running on :4001");
app.Run("http://0.0.0.0:4001");

// --- Types ---

public record EffectRequest(
    [property: JsonPropertyName("effect_type")] string EffectType,
    [property: JsonPropertyName("principal_id")] string PrincipalId,
    [property: JsonPropertyName("params")] Dictionary<string, JsonElement>? Params = null,
    [property: JsonPropertyName("context")] Dictionary<string, JsonElement>? Context = null,
    [property: JsonPropertyName("idempotency_key")] string? IdempotencyKey = null
);

public record CompletionRequest(
    [property: JsonPropertyName("result")] Dictionary<string, JsonElement>? Result = null
);

public record Receipt(
    [property: JsonPropertyName("receipt_id")] string ReceiptId,
    [property: JsonPropertyName("verdict")] string Verdict,
    [property: JsonPropertyName("reason_code")] string ReasonCode,
    [property: JsonPropertyName("reason")] string Reason,
    [property: JsonPropertyName("timestamp")] string Timestamp,
    [property: JsonPropertyName("lamport")] int Lamport,
    [property: JsonPropertyName("principal_id")] string PrincipalId
);

public record SubmitResponse(
    [property: JsonPropertyName("verdict")] string Verdict,
    [property: JsonPropertyName("receipt")] Receipt Receipt,
    [property: JsonPropertyName("intent")] Intent Intent
);

public record Intent(
    [property: JsonPropertyName("effect_type")] string EffectType,
    [property: JsonPropertyName("allowed")] bool Allowed
);

// --- EffectBoundary ---

public class EffectBoundary
{
    private int _lamport;
    private readonly List<Receipt> _receipts = new();

    public SubmitResponse Submit(EffectRequest req)
    {
        _lamport++;
        var (verdict, reasonCode, reason) = Evaluate(req);
        var receipt = CreateReceipt(verdict, reasonCode, reason, req.PrincipalId);
        _receipts.Add(receipt);

        return new SubmitResponse(
            verdict,
            receipt,
            new Intent(req.EffectType, verdict == "ALLOW")
        );
    }

    public Receipt Complete(string receiptId, CompletionRequest req)
    {
        _lamport++;
        var receipt = CreateReceipt("ALLOW", "EFFECT_COMPLETED",
            $"Effect {receiptId} completed", "system");
        _receipts.Add(receipt);
        return receipt;
    }

    private (string Verdict, string ReasonCode, string Reason) Evaluate(EffectRequest req)
    {
        if (req.EffectType == "data_export" &&
            req.Params?.TryGetValue("data_class", out var dc) == true &&
            dc.GetString() == "PII")
        {
            return ("DENY", "POLICY_VIOLATION", "PII export denied by policy");
        }

        if (req.EffectType == "financial_transfer" &&
            req.Params?.TryGetValue("amount_cents", out var ac) == true &&
            ac.GetInt32() > 1_000_000)
        {
            return ("ESCALATE", "TEMPORAL_INTERVENTION",
                "High value transfer requires approval");
        }

        return ("ALLOW", "POLICY_SATISFIED", "Effect allowed");
    }

    private Receipt CreateReceipt(string verdict, string reasonCode,
        string reason, string principalId)
    {
        var ts = DateTime.UtcNow.ToString("o");
        var content = $"{verdict}:{reasonCode}:{ts}:{_lamport}";
        var hash = Convert.ToHexString(
            SHA256.HashData(Encoding.UTF8.GetBytes(content))
        )[..16].ToLowerInvariant();

        return new Receipt(
            $"urn:helm:receipt:{hash}", verdict, reasonCode,
            reason, ts, _lamport, principalId
        );
    }
}
