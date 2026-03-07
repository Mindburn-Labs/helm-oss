package agent

import (
	"context"
	"fmt"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/crypto"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/executor"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/guardian"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/kernel"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/manifest"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/mcp"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store/ledger"
)

// KernelBridge exposes the OS to the LLM.
type KernelBridge struct {
	ledger   ledger.Ledger
	executor executor.Executor
	catalog  mcp.Catalog
	guardian *guardian.Guardian
	verifier crypto.Verifier
	limiter  kernel.LimiterStore
}

func NewKernelBridge(l ledger.Ledger, e executor.Executor, c mcp.Catalog, g *guardian.Guardian, verifier crypto.Verifier, lim kernel.LimiterStore) *KernelBridge {
	return &KernelBridge{
		ledger:   l,
		executor: e,
		catalog:  c,
		guardian: g,
		verifier: verifier,
		limiter:  lim,
	}
}

// Dispatch handles a tool call from the LLM.
func (k *KernelBridge) Dispatch(ctx context.Context, toolName string, params map[string]any) (any, error) {
	// 1. Identify Actor (Context-Awareness)
	// In a real system, we extract the claims from the context (ignoring `params` for identity).
	actorID := "anonymous"
	if v := ctx.Value("actor_id"); v != nil {
		actorID = v.(string)
	}

	// 2. Rate Limit Check
	if k.limiter != nil {
		// Use real ActorID
		policy := kernel.BackpressurePolicy{RPM: 600, Burst: 50} // 10 RPS
		if err := kernel.EvaluateBackpressure(ctx, k.limiter, actorID, policy); err != nil {
			return nil, fmt.Errorf("rate limit exceeded for %s: %w", actorID, err)
		}
	}

	switch KernelToolName(toolName) {
	case ToolCreateObligation:
		return k.createObligation(ctx, params)
	case ToolCallMCPTool:
		return k.callMCPTool(ctx, params)
	case ToolMCPToolSearch:
		return k.searchTools(ctx, params)
	case ToolSearchObligations:
		return k.searchObligations(ctx, params)
	case ToolProposePlan:
		return k.proposePlan(ctx, params)
	case ToolRequestDecision:
		return k.requestDecision(ctx, params)
	case ToolSubmitModule:
		return k.submitModuleBundle(ctx, params)
	case ToolRequestActivation:
		return k.requestModuleActivation(ctx, params)
	default:
		// Dynamic dispatch for MCP tools found in Catalog
		if k.catalog != nil {
			tools, searchErr := k.catalog.Search(ctx, toolName)
			if searchErr != nil {
				return nil, fmt.Errorf("catalog search failed for %q: %w", toolName, searchErr)
			}
			for _, t := range tools {
				if t.Name == toolName {
					return k.dispatchDirectMCP(ctx, toolName, params)
				}
			}
		}
		return nil, fmt.Errorf("unknown kernel tool: %s", toolName)
	}
}

func (k *KernelBridge) createObligation(ctx context.Context, params map[string]any) (string, error) {
	intent, _ := params["intent"].(string)
	idem, _ := params["idempotency_key"].(string)

	if intent == "" {
		return "", fmt.Errorf("missing intent")
	}

	obl := ledger.Obligation{
		ID:             "obl-" + idem, // Simplification
		IdempotencyKey: idem,
		Intent:         intent,
		State:          ledger.StatePending,
	}

	if err := k.ledger.Create(ctx, obl); err != nil {
		return "", fmt.Errorf("failed to create obligation: %w", err)
	}

	return obl.ID, nil
}

func (k *KernelBridge) callMCPTool(ctx context.Context, params map[string]any) (any, error) {
	toolName, _ := params["tool_name"].(string)
	decisionToken, _ := params["decision_id"].(string) // In MVP, this is the JSON blob
	toolParams, _ := params["params"].(map[string]any)

	if toolName == "" {
		return "", fmt.Errorf("missing tool_name")
	}
	if decisionToken == "" {
		return "", fmt.Errorf("missing decision_id (token)")
	}

	// Phase 2 PEP Boundary: Validate and canonicalize tool args.
	// This stops inbound hallucinations and binds a stable argsHash.
	var argsHash string
	if toolParams != nil {
		// Lookup schema from catalog if available
		var schema *manifest.ToolArgSchema
		if k.catalog != nil {
			if tools, err := k.catalog.Search(ctx, toolName); err == nil {
				for _, t := range tools {
					if t.Name == toolName && t.Schema != nil {
						schema = catalogSchemaToArgSchema(t.Schema)
						break
					}
				}
			}
		}
		result, err := manifest.ValidateAndCanonicalizeToolArgs(schema, toolParams)
		if err != nil {
			return "", fmt.Errorf("PEP boundary: %w", err)
		}
		argsHash = result.ArgsHash
	}

	// 1. Decode generic decision token
	decision, err := contracts.DecodeDecisionRecord(decisionToken)
	if err != nil {
		return "", fmt.Errorf("invalid decision token format: %w", err)
	}

	// 2. Construct Effect with argsHash binding
	effect := &contracts.Effect{
		EffectID:   "eff-" + decision.ID, // Link effect to decision
		Params:     toolParams,
		EffectType: contracts.EffectTypeCallTool,
		ArgsHash:   argsHash, // Phase 2: bound canonical hash of validated args
	}
	if effect.Params == nil {
		effect.Params = make(map[string]any)
	}
	effect.Params["tool_name"] = toolName

	// Sequence 8: Authorize Effect (Issue Intent)
	if k.guardian == nil {
		return "", fmt.Errorf("execution blocked: guardian not configured")
	}
	intent, err := k.guardian.IssueExecutionIntent(ctx, decision, effect)
	if err != nil {
		return "", fmt.Errorf("intent issuance failed: %w", err)
	}

	// Returns receipt, result, error
	_, result, err := k.executor.Execute(ctx, effect, decision, intent)
	if err != nil {
		return "", fmt.Errorf("execution rejected: %w", err)
	}

	// Return the actual tool result (or truncated version)
	// We marshal if needed, but 'any' is fine for JSON-RPC layer
	return result, nil
}

func (k *KernelBridge) searchTools(ctx context.Context, params map[string]any) (any, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("missing query")
	}

	if k.catalog == nil {
		return nil, fmt.Errorf("catalog not available")
	}

	return k.catalog.Search(ctx, query)
}

// dispatchDirectMCP executes an MCP tool directly, bypassing the "call_mcp_tool" wrapper.
// This allows the Agent to call tools naturally as defined in the catalog.
func (k *KernelBridge) dispatchDirectMCP(ctx context.Context, toolName string, params map[string]any) (any, error) {
	// 1. Authorize via Guardian (Enforce Policy)
	// We synthesize a request, but the Guardian must validate it.
	// In strict mode, we shouldn't even be synthesizing the decision here.
	// We should be asking the Guardian for a decision first.
	// For now, to close the backdoor, we reuse callMCPTool's flow or require a decision token.
	// BUT, dispatchDirectMCP is for internal agent calls.
	// The Pivot: The Agent *is* a principal. It should request a decision.
	// Since we don't have a decision token here, we fail-closed.
	// Direct execution without a decision token is not permitted.
	return nil, fmt.Errorf("direct execution blocked: agent must request decision token via 'request_decision' tool first")
}

// catalogSchemaToArgSchema converts a catalog's generic schema (map[string]any)
// into a typed manifest.ToolArgSchema for PEP boundary validation.
func catalogSchemaToArgSchema(raw any) *manifest.ToolArgSchema {
	schemaMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	props, _ := schemaMap["properties"].(map[string]any)
	if props == nil {
		return nil
	}
	requiredList, _ := schemaMap["required"].([]any)
	requiredSet := make(map[string]bool)
	for _, r := range requiredList {
		if s, ok := r.(string); ok {
			requiredSet[s] = true
		}
	}
	fields := make(map[string]manifest.FieldSpec)
	for name, propRaw := range props {
		prop, _ := propRaw.(map[string]any)
		t, _ := prop["type"].(string)
		if t == "" {
			t = "any"
		}
		fields[name] = manifest.FieldSpec{
			Type:     t,
			Required: requiredSet[name],
		}
	}
	// Reject unknown fields by default (strict validation)
	return &manifest.ToolArgSchema{
		Fields:     fields,
		AllowExtra: false,
	}
}
