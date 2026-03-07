package agent

import (
	"context"
	"fmt"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
)

func (k *KernelBridge) searchObligations(ctx context.Context, params map[string]any) (any, error) {
	// MVP: Just list all for now, assuming SQL filters come later
	// ledger.List() needed
	// For now, return stub
	return []map[string]string{
		{"id": "obl-scale-123", "intent": "Scale Service A", "status": "PENDING"},
	}, nil
}

func (k *KernelBridge) proposePlan(ctx context.Context, params map[string]any) (any, error) {
	return nil, fmt.Errorf("planning subsystem deprecated in OSS kernel")
}

func (k *KernelBridge) requestDecision(ctx context.Context, params map[string]any) (any, error) {
	// dryRun, _ := params["dry_run"].(bool)
	planID, _ := params["plan_attempt_id"].(string)

	if planID == "" {
		return nil, fmt.Errorf("missing plan_attempt_id")
	}

	// 1. Construct Decision Record (Draft)
	// For demo, we create a fresh decision. In real system, this might link to a Proposal.
	decision := &contracts.DecisionRecord{
		ID:         "dec-" + planID + "-1", // Simplified
		ProposalID: planID,
		// Verdict will be set by Guardian
	}

	// 2. Extract Action/Effect details (Mocking extraction from params/plan)
	// In strict mode, we'd lookup the step from the Plan.
	// We assume params has tool info:
	toolName, _ := params["tool_name"].(string)
	if toolName == "" {
		// Fallback to generic action if checking plan as a whole (not implemented in PRG yet)
		toolName = "generic_action"
	}

	effect := &contracts.Effect{
		EffectID: "eff-" + decision.ID,
		Params:   map[string]any{"tool_name": toolName},
	}

	// 3. Extract Evidence Hashes provided by Agent
	evidenceRaw, _ := params["evidence_hashes"].([]any)
	var evidenceHashes []string
	for _, e := range evidenceRaw {
		if s, ok := e.(string); ok {
			evidenceHashes = append(evidenceHashes, s)
		}
	}

	// 4. Call Guardian to Sign (MANDATORY - Fail Closed)
	if k.guardian == nil {
		return nil, fmt.Errorf("governance violation: Guardian not configured (fail-closed)")
	}
	if err := k.guardian.SignDecision(ctx, decision, effect, evidenceHashes, nil); err != nil {
		return map[string]string{
			"verdict": "FAIL",
			"reason":  fmt.Sprintf("Guardian Rejected: %v", err),
		}, nil
	}

	// 5. Encode Token
	token, err := contracts.EncodeDecisionRecord(decision)
	if err != nil {
		return nil, fmt.Errorf("failed to encode decision: %w", err)
	}

	return map[string]any{
		"decision_id": token,
		"verdict":     decision.Verdict,
		"reason":      decision.Reason,
	}, nil
}

func (k *KernelBridge) submitModuleBundle(ctx context.Context, params map[string]any) (any, error) {
	return nil, fmt.Errorf("orgvm subsystem deprecated in OSS kernel")
}

func (k *KernelBridge) requestModuleActivation(ctx context.Context, params map[string]any) (any, error) {
	bundleID, _ := params["bundle_id"].(string)
	// strategy, _ := params["strategy"].(string) // Unused for now

	if bundleID == "" {
		return nil, fmt.Errorf("missing bundle_id")
	}

	// 1. Create a Governed Action (ActionActivateModule)
	// 2. Create an Obligation to execute this action

	// obl := ledger.Obligation{
	// 	ID: "obl-act-" + bundleID,
	// 	Intent: fmt.Sprintf("Activate Module %s (Strategy: %s)", bundleID, strategy),
	// 	State: ledger.StatePending,
	// }
	// err := k.ledger.Create(ctx, obl)

	// For now, return stub
	return map[string]string{
		"obligation_id": "obl-act-" + bundleID,
		"status":        "QUEUED",
	}, nil
}
