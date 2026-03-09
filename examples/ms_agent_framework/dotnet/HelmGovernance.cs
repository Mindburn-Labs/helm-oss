// HELM × Microsoft Agent Framework — Minimal .NET Example
//
// NuGet: dotnet add package Mindburn.Helm.Governance
// Or use the HTTP API directly as shown below.
//
// This is a minimal example for the MS Agent Framework RC.
// For production use, see the Python adapter (sdk/python/microsoft_agents/).

using System;
using System.Net.Http;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;

namespace Mindburn.Helm.Example;

/// <summary>
/// Minimal HELM governance client for MS Agent Framework .NET.
/// Routes tool execution through HELM PEP boundary.
/// </summary>
public class HelmGovernance
{
    private readonly HttpClient _client;
    private readonly string _baseUrl;

    public HelmGovernance(string helmUrl = "http://localhost:8080")
    {
        _baseUrl = helmUrl.TrimEnd('/');
        _client = new HttpClient { Timeout = TimeSpan.FromSeconds(10) };
    }

    /// <summary>
    /// Evaluate a tool execution against HELM governance policies.
    /// </summary>
    public async Task<GovernanceResult> EvaluateAsync(string toolName, object arguments)
    {
        var payload = JsonSerializer.Serialize(new
        {
            tool_name = toolName,
            arguments = arguments,
            principal = "ms-agent-dotnet"
        });

        try
        {
            var response = await _client.PostAsync(
                $"{_baseUrl}/v1/tools/evaluate",
                new StringContent(payload, Encoding.UTF8, "application/json")
            );

            var body = await response.Content.ReadAsStringAsync();
            var result = JsonSerializer.Deserialize<GovernanceResult>(body);
            return result ?? new GovernanceResult { Verdict = "ALLOW", ReasonCode = "POLICY_PASS" };
        }
        catch (Exception)
        {
            // Fail-closed: deny on HELM unreachable
            return new GovernanceResult { Verdict = "DENY", ReasonCode = "HELM_UNREACHABLE" };
        }
    }
}

public class GovernanceResult
{
    public string Verdict { get; set; } = "ALLOW";
    public string ReasonCode { get; set; } = "";
    public string ReceiptId { get; set; } = "";
}

// Usage with MS Agent Framework RC:
//
// var helm = new HelmGovernance("http://localhost:8080");
// var result = await helm.EvaluateAsync("deploy_service", new { env = "production" });
//
// if (result.Verdict == "DENY")
//     throw new InvalidOperationException($"HELM denied: {result.ReasonCode}");
//
// // Proceed with tool execution
