package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
)

// Run types matching the control-room UI RunSchema (Zod-validated).

type runStageData struct {
	Stage     string `json:"stage"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
}

type runProposal struct {
	ProposalID string `json:"proposal_id"`
	Kind       string `json:"kind"`
	Intent     string `json:"intent"`
	Scope      string `json:"scope"`
	CreatedAt  string `json:"created_at"`
}

type runDecision struct {
	DecisionID string `json:"decision_id"`
	ProposalID string `json:"proposal_id"`
	Verdict    string `json:"verdict"`
	SignedAt   string `json:"signed_at"`
	Signer     string `json:"signer"`
}

type runReceipt struct {
	ReceiptID  string `json:"receipt_id"`
	DecisionID string `json:"decision_id"`
	EffectID   string `json:"effect_id"`
	Status     string `json:"status"`
	Timestamp  string `json:"timestamp"`
	BlobHash   string `json:"blob_hash,omitempty"`
	ExecutorID string `json:"executor_id"`
}

type runEffect struct {
	EffectID string `json:"effect_id"`
	ToolName string `json:"tool_name"`
	Status   string `json:"status"`
}

type apiRun struct {
	RunID        string         `json:"run_id"`
	Status       string         `json:"status"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	CurrentStage string         `json:"current_stage"`
	Stages       []runStageData `json:"stages"`
	Proposal     *runProposal   `json:"proposal,omitempty"`
	Decision     *runDecision   `json:"decision,omitempty"`
	Effects      []runEffect    `json:"effects"`
	Receipt      *runReceipt    `json:"receipt,omitempty"`
}

type runsListResponse struct {
	Runs     []apiRun `json:"runs"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// mapReceiptStatus maps receipt status to run status.
func mapReceiptStatus(status string) string {
	switch status {
	case "SUCCESS":
		return "complete"
	case "FAILURE":
		return "failed"
	default:
		return "active"
	}
}

// mapReceiptStage maps receipt status to the corresponding run stage.
func mapReceiptStage(status string) string {
	switch status {
	case "SUCCESS":
		return "run_complete"
	case "FAILURE":
		return "run_failed"
	default:
		return "execution_started"
	}
}

// handleRunsListAPI serves GET /api/runs with optional pagination.
func (s *Server) handleRunsListAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	// Parse pagination
	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	// Fetch receipts from store (use a generous limit for total count)
	ctx := r.Context()
	receipts, err := s.receiptStore.List(ctx, 1000)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	total := len(receipts)

	// Apply pagination
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	pageReceipts := receipts[start:end]

	runs := make([]apiRun, 0, len(pageReceipts))
	for _, rcpt := range pageReceipts {
		ts := rcpt.Timestamp.UTC().Format(time.RFC3339)
		runStatus := mapReceiptStatus(rcpt.Status)
		currentStage := mapReceiptStage(rcpt.Status)

		toolName := "unknown"
		if meta, ok := rcpt.Metadata["tool"]; ok {
			toolName = fmt.Sprintf("%v", meta)
		}

		run := apiRun{
			RunID:        rcpt.ReceiptID,
			Status:       runStatus,
			CreatedAt:    ts,
			UpdatedAt:    ts,
			CurrentStage: currentStage,
			Stages: []runStageData{
				{Stage: "proposal_created", Status: "complete", Timestamp: ts},
				{Stage: "decision_signed", Status: "complete", Timestamp: ts},
				{Stage: "execution_started", Status: "complete", Timestamp: ts},
				{Stage: "receipt_committed", Status: "complete", Timestamp: ts},
				{Stage: currentStage, Status: "complete", Timestamp: ts},
			},
			Proposal: &runProposal{
				ProposalID: "prop-" + rcpt.ReceiptID,
				Kind:       "tool_execution",
				Intent:     toolName,
				Scope:      "tenant",
				CreatedAt:  ts,
			},
			Decision: &runDecision{
				DecisionID: rcpt.DecisionID,
				ProposalID: "prop-" + rcpt.ReceiptID,
				Verdict:    "PERMIT",
				SignedAt:   ts,
				Signer:     "helm-kernel",
			},
			Effects: []runEffect{
				{
					EffectID: rcpt.EffectID,
					ToolName: toolName,
					Status:   "executed",
				},
			},
			Receipt: &runReceipt{
				ReceiptID:  rcpt.ReceiptID,
				DecisionID: rcpt.DecisionID,
				EffectID:   rcpt.EffectID,
				Status:     rcpt.Status,
				Timestamp:  ts,
				BlobHash:   rcpt.BlobHash,
				ExecutorID: rcpt.ExecutorID,
			},
		}

		runs = append(runs, run)
	}

	resp := runsListResponse{
		Runs:     runs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// handleRunDetailAPI serves GET /api/runs/{id}.
func (s *Server) handleRunDetailAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	// Extract run ID from path: /api/runs/{id}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 || pathParts[2] == "" {
		api.WriteBadRequest(w, "Missing run ID")
		return
	}
	runID := pathParts[2]

	ctx := r.Context()
	rcpt, err := s.receiptStore.GetByReceiptID(ctx, runID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.WriteNotFound(w, "Run not found")
			return
		}
		api.WriteInternal(w, err)
		return
	}

	ts := rcpt.Timestamp.UTC().Format(time.RFC3339)
	runStatus := mapReceiptStatus(rcpt.Status)
	currentStage := mapReceiptStage(rcpt.Status)

	toolName := "unknown"
	if meta, ok := rcpt.Metadata["tool"]; ok {
		toolName = fmt.Sprintf("%v", meta)
	}

	run := apiRun{
		RunID:        rcpt.ReceiptID,
		Status:       runStatus,
		CreatedAt:    ts,
		UpdatedAt:    ts,
		CurrentStage: currentStage,
		Stages: []runStageData{
			{Stage: "proposal_created", Status: "complete", Timestamp: ts},
			{Stage: "decision_signed", Status: "complete", Timestamp: ts},
			{Stage: "execution_started", Status: "complete", Timestamp: ts},
			{Stage: "receipt_committed", Status: "complete", Timestamp: ts},
			{Stage: currentStage, Status: "complete", Timestamp: ts},
		},
		Proposal: &runProposal{
			ProposalID: "prop-" + rcpt.ReceiptID,
			Kind:       "tool_execution",
			Intent:     toolName,
			Scope:      "tenant",
			CreatedAt:  ts,
		},
		Decision: &runDecision{
			DecisionID: rcpt.DecisionID,
			ProposalID: "prop-" + rcpt.ReceiptID,
			Verdict:    "PERMIT",
			SignedAt:   ts,
			Signer:     "helm-kernel",
		},
		Effects: []runEffect{
			{
				EffectID: rcpt.EffectID,
				ToolName: toolName,
				Status:   "executed",
			},
		},
		Receipt: &runReceipt{
			ReceiptID:  rcpt.ReceiptID,
			DecisionID: rcpt.DecisionID,
			EffectID:   rcpt.EffectID,
			Status:     rcpt.Status,
			Timestamp:  ts,
			BlobHash:   rcpt.BlobHash,
			ExecutorID: rcpt.ExecutorID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(run) //nolint:errcheck
}
