package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/guardian"
)

// ToolExecutionRequest represents a request to execute a tool via MCP.
type ToolExecutionRequest struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	SessionID string                 `json:"session_id"`
}

// ToolExecutionResponse represents the result of a tool execution.
type ToolExecutionResponse struct {
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
	Evaluated bool   `json:"evaluated"` // Whether policy was evaluated
	ReceiptID string `json:"receipt_id,omitempty"`
}

// PolicyEvaluator abstracts the governance decision evaluation.
// This allows the GovernanceFirewall to be tested without a full Guardian.
type PolicyEvaluator interface {
	EvaluateDecision(ctx context.Context, req guardian.DecisionRequest) (*contracts.DecisionRecord, error)
}

// GovernanceFirewall intercepts tool calls and enforces Guardian policies.
type GovernanceFirewall struct {
	evaluator PolicyEvaluator
	catalog   *ToolCatalog
}

// NewGovernanceFirewall creates a new firewall instance.
// The guardian.Guardian satisfies the PolicyEvaluator interface.
func NewGovernanceFirewall(evaluator PolicyEvaluator, catalog *ToolCatalog) *GovernanceFirewall {
	return &GovernanceFirewall{evaluator: evaluator, catalog: catalog}
}

// InterceptToolExecution checks if a tool execution is allowed by the Guardian.
// If allowed, it returns nil. If blocked, it returns an error.
func (f *GovernanceFirewall) InterceptToolExecution(ctx context.Context, req ToolExecutionRequest) error {
	decision, err := f.evaluator.EvaluateDecision(ctx, guardian.DecisionRequest{
		Principal: req.SessionID,
		Action:    "EXECUTE_TOOL",
		Resource:  req.ToolName,
		Context:   req.Arguments,
	})
	if err != nil {
		return fmt.Errorf("governance check failed: %w", err)
	}

	// Enforce Decision — use canonical verdict constants
	if decision.Verdict == string(contracts.VerdictDeny) {
		return fmt.Errorf("governance blocked execution: %s", decision.Reason)
	}

	if decision.Verdict == string(contracts.VerdictEscalate) || decision.Verdict == "PENDING" {
		return fmt.Errorf("governance requires approval: %s", decision.Reason)
	}

	// Allow Proceed
	return nil
}

// WrapToolHandler wraps a standard tool handler with the firewall.
type ToolHandler func(ctx context.Context, req ToolExecutionRequest) (ToolExecutionResponse, error)

func (f *GovernanceFirewall) WrapToolHandler(handler ToolHandler) ToolHandler {
	return func(ctx context.Context, req ToolExecutionRequest) (ToolExecutionResponse, error) {
		// 1. Pre-Execution Check
		if err := f.InterceptToolExecution(ctx, req); err != nil {
			slog.Warn("governance_firewall: tool execution blocked",
				"tool", req.ToolName,
				"session_id", req.SessionID,
				"error", err,
			)
			return ToolExecutionResponse{
				Content:   fmt.Sprintf("Access Denied: %v", err),
				IsError:   true,
				Evaluated: true,
			}, nil // Return error as response content so agent sees it
		}

		// 2. Execute
		resp, err := handler(ctx, req)

		// 3. Post-Execution Audit
		if f.catalog != nil {
			receipt, auditErr := f.catalog.AuditToolCall(req.ToolName, req.Arguments, resp.Content)
			if auditErr != nil {
				slog.Error("governance_firewall: audit logging failed",
					"tool", req.ToolName,
					"error", auditErr,
				)
			} else {
				slog.Info("governance_firewall: tool execution audited",
					"receipt_id", receipt.ID,
					"tool", receipt.ToolName,
				)
				resp.ReceiptID = receipt.ID
			}
		}

		resp.Evaluated = true
		return resp, err
	}
}

// ToolExecutionPlan represents a sequence of tool calls to be executed.
type ToolExecutionPlan struct {
	PlanID string                 `json:"plan_id"`
	Steps  []ToolExecutionRequest `json:"steps"`
}

// PlanDecision represents the governance decision for an entire plan.
type PlanDecision struct {
	PlanID    string                      `json:"plan_id"`
	Decisions []*contracts.DecisionRecord `json:"decisions"`
	Status    string                      `json:"status"` // ALLOW, DENY, ESCALATE
}

// InterceptPlan evaluates a proposed plan against governance policies.
// It returns a PlanDecision indicating which steps are allowed, blocked, or pending approval.
func (f *GovernanceFirewall) InterceptPlan(ctx context.Context, plan ToolExecutionPlan) (*PlanDecision, error) {
	decisions := make([]*contracts.DecisionRecord, 0, len(plan.Steps))
	overallStatus := string(contracts.VerdictAllow)

	for _, step := range plan.Steps {
		// Evaluate each step
		decision, err := f.evaluator.EvaluateDecision(ctx, guardian.DecisionRequest{
			Principal: step.SessionID,
			Action:    "EXECUTE_TOOL",
			Resource:  step.ToolName,
			Context:   step.Arguments,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate step %s: %w", step.ToolName, err)
		}

		// Aggregate Status
		if decision.Verdict == string(contracts.VerdictDeny) {
			overallStatus = string(contracts.VerdictDeny)
		} else if decision.Verdict == string(contracts.VerdictEscalate) && overallStatus != string(contracts.VerdictDeny) {
			overallStatus = string(contracts.VerdictEscalate)
		}

		decisions = append(decisions, decision)
	}

	return &PlanDecision{
		PlanID:    plan.PlanID,
		Decisions: decisions,
		Status:    overallStatus,
	}, nil
}
