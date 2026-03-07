package console

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
)

// ---------------------------------------------------------------------------
// Operator types — canonical structs for the Intent → Plan → Approve → Run loop
// ---------------------------------------------------------------------------

// IntentStatus tracks an intent through its lifecycle.
type IntentStatus string

const (
	IntentDraft     IntentStatus = "draft"
	IntentPlanned   IntentStatus = "planned"
	IntentSubmitted IntentStatus = "submitted"
	IntentApproved  IntentStatus = "approved"
	IntentRejected  IntentStatus = "rejected"
	IntentExecuting IntentStatus = "executing"
	IntentCompleted IntentStatus = "completed"
)

// operatorIntent represents an operator's declared intention.
type operatorIntent struct {
	IntentID    string                 `json:"intent_id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Status      IntentStatus           `json:"status"`
	PlanHash    string                 `json:"plan_hash,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// operatorPlanStep is a single step in a plan.
type operatorPlanStep struct {
	StepID      string `json:"step_id"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	Description string `json:"description"`
	Estimated   string `json:"estimated_duration,omitempty"`
}

// operatorPlan is the output of a dry-run plan preview.
type operatorPlan struct {
	IntentID  string             `json:"intent_id"`
	Steps     []operatorPlanStep `json:"steps"`
	Hash      string             `json:"hash"`
	DryRunOK  bool               `json:"dry_run_ok"`
	CreatedAt string             `json:"created_at"`
	RiskLevel string             `json:"risk_level"`
	ReceiptID string             `json:"receipt_id"`
}

// ApprovalDecision is the operator's verdict.
type ApprovalDecision string

const (
	DecisionApprove ApprovalDecision = "approve"
	DecisionReject  ApprovalDecision = "reject"
)

// operatorApproval records an approval decision.
type operatorApproval struct {
	ApprovalID string           `json:"approval_id"`
	IntentID   string           `json:"intent_id"`
	Decision   ApprovalDecision `json:"decision"`
	Reason     string           `json:"reason"`
	DecidedBy  string           `json:"decided_by"`
	DecidedAt  string           `json:"decided_at"`
	ReceiptID  string           `json:"receipt_id"`
}

// RunControlAction defines what the operator wants to do to a run.
type RunControlAction string

const (
	RunActionPause  RunControlAction = "pause"
	RunActionCancel RunControlAction = "cancel"
	RunActionRetry  RunControlAction = "retry"
	RunActionResume RunControlAction = "resume"
)

// RunState tracks an operator-initiated run.
type RunState string

const (
	RunStatePending   RunState = "pending"
	RunStateRunning   RunState = "running"
	RunStatePaused    RunState = "paused"
	RunStateCancelled RunState = "canceled"
	RunStateCompleted RunState = "completed"
	RunStateFailed    RunState = "failed"
)

// operatorRunState tracks an operator-created run.
type operatorRunState struct {
	RunID     string   `json:"run_id"`
	IntentID  string   `json:"intent_id"`
	State     RunState `json:"state"`
	PlanHash  string   `json:"plan_hash"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	ReceiptID string   `json:"receipt_id"`
}

// replayDiff shows difference between a planned and executed step.
type replayDiff struct {
	StepID   string `json:"step_id"`
	Planned  string `json:"planned"`
	Executed string `json:"executed"`
	Match    bool   `json:"match"`
}

// replayResult is the output of a replay comparison.
type replayResult struct {
	RunID           string       `json:"run_id"`
	PlannedSteps    int          `json:"planned_steps"`
	ExecutedSteps   int          `json:"executed_steps"`
	Diffs           []replayDiff `json:"diffs"`
	MatchPercentage float64      `json:"match_percentage"`
	ReceiptID       string       `json:"receipt_id"`
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

type createIntentRequest struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

type approvalRequest struct {
	Decision  ApprovalDecision `json:"decision"`
	Reason    string           `json:"reason"`
	DecidedBy string           `json:"decided_by"`
}

type createRunRequest struct {
	IntentID string `json:"intent_id"`
}

type runControlRequest struct {
	Action RunControlAction `json:"action"`
	Reason string           `json:"reason,omitempty"`
}

// ---------------------------------------------------------------------------
// List response wrappers
// ---------------------------------------------------------------------------

type intentsListResponse struct {
	Intents []operatorIntent `json:"intents"`
	Total   int              `json:"total"`
}

type approvalsListResponse struct {
	Approvals []operatorApproval `json:"approvals"`
	Total     int                `json:"total"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func generateID(prefix string) string {
	ts := time.Now().UnixNano()
	raw := fmt.Sprintf("%s-%d", prefix, ts)
	h := sha256.Sum256([]byte(raw))
	return prefix + "-" + hex.EncodeToString(h[:8])
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// storeOperatorReceipt creates and persists a receipt for an operator action.
func (s *Server) storeOperatorReceipt(tool string, meta map[string]any) string {
	id := generateID("rcpt")
	receipt := &contracts.Receipt{
		ReceiptID:  id,
		DecisionID: generateID("dec"),
		EffectID:   generateID("eff"),
		Status:     "SUCCESS",
		Timestamp:  time.Now(),
		ExecutorID: "helm-operator",
		Metadata:   meta,
	}
	if receipt.Metadata == nil {
		receipt.Metadata = make(map[string]any)
	}
	receipt.Metadata["tool"] = tool
	_ = s.receiptStore.Store(context.Background(), receipt) //nolint:errcheck
	return id
}

// ---------------------------------------------------------------------------
// Intent Handlers
// ---------------------------------------------------------------------------

// handleCreateIntent handles POST /api/intents.
func (s *Server) handleCreateIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req createIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Type == "" || req.Description == "" {
		writeError(w, http.StatusBadRequest, "type and description are required")
		return
	}

	now := nowRFC3339()
	intent := &operatorIntent{
		IntentID:    generateID("int"),
		Type:        req.Type,
		Description: req.Description,
		Params:      req.Params,
		Status:      IntentDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.operatorMu.Lock()
	s.intents[intent.IntentID] = intent
	s.operatorMu.Unlock()

	receiptID := s.storeOperatorReceipt("operator_create_intent", map[string]any{
		"intent_id":   intent.IntentID,
		"intent_type": intent.Type,
	})
	_ = receiptID

	writeJSON(w, http.StatusCreated, intent)
}

// handleListIntents handles GET /api/intents.
func (s *Server) handleListIntents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	statusFilter := r.URL.Query().Get("status")

	s.operatorMu.RLock()
	intents := make([]operatorIntent, 0, len(s.intents))
	for _, intent := range s.intents {
		if statusFilter != "" && string(intent.Status) != statusFilter {
			continue
		}
		intents = append(intents, *intent)
	}
	s.operatorMu.RUnlock()

	writeJSON(w, http.StatusOK, intentsListResponse{
		Intents: intents,
		Total:   len(intents),
	})
}

// handleGetIntent handles GET /api/intents/{id}.
func (s *Server) handleGetIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	intentID := extractPathSegment(r.URL.Path, "intents")
	if intentID == "" {
		writeError(w, http.StatusBadRequest, "Missing intent ID")
		return
	}

	// Check for sub-resource paths like /api/intents/{id}/plan
	if strings.Contains(r.URL.Path, "/plan") || strings.Contains(r.URL.Path, "/submit") {
		writeError(w, http.StatusBadRequest, "Use POST for plan/submit actions")
		return
	}

	s.operatorMu.RLock()
	intent, ok := s.intents[intentID]
	s.operatorMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "Intent not found")
		return
	}

	writeJSON(w, http.StatusOK, intent)
}

// handlePlanIntent handles POST /api/intents/{id}/plan — dry-run plan preview.
func (s *Server) handlePlanIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	intentID := extractIntentIDFromSubpath(r.URL.Path)
	if intentID == "" {
		writeError(w, http.StatusBadRequest, "Missing intent ID")
		return
	}

	s.operatorMu.Lock()
	defer s.operatorMu.Unlock()

	intent, ok := s.intents[intentID]
	if !ok {
		writeError(w, http.StatusNotFound, "Intent not found")
		return
	}

	if intent.Status != IntentDraft && intent.Status != IntentPlanned {
		writeError(w, http.StatusConflict, fmt.Sprintf("Cannot plan intent in status %q", intent.Status))
		return
	}

	// Generate plan steps based on intent type
	steps := generatePlanSteps(intent)

	// Hash the plan
	planBytes, _ := json.Marshal(steps) //nolint:errcheck
	planHash := sha256.Sum256(planBytes)
	hashStr := "sha256:" + hex.EncodeToString(planHash[:])

	// Determine risk level
	riskLevel := "low"
	if len(steps) > 3 {
		riskLevel = "medium"
	}
	if intent.Type == "deploy" || intent.Type == "delete" {
		riskLevel = "high"
	}

	// Update intent status
	intent.Status = IntentPlanned
	intent.PlanHash = hashStr
	intent.UpdatedAt = nowRFC3339()

	receiptID := s.storeOperatorReceipt("operator_plan_intent", map[string]any{
		"intent_id": intentID,
		"plan_hash": hashStr,
		"steps":     len(steps),
	})

	plan := operatorPlan{
		IntentID:  intentID,
		Steps:     steps,
		Hash:      hashStr,
		DryRunOK:  true,
		CreatedAt: nowRFC3339(),
		RiskLevel: riskLevel,
		ReceiptID: receiptID,
	}

	writeJSON(w, http.StatusOK, plan)
}

// handleSubmitIntent handles POST /api/intents/{id}/submit — moves to approval queue.
func (s *Server) handleSubmitIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	intentID := extractIntentIDFromSubpath(r.URL.Path)
	if intentID == "" {
		writeError(w, http.StatusBadRequest, "Missing intent ID")
		return
	}

	s.operatorMu.Lock()
	defer s.operatorMu.Unlock()

	intent, ok := s.intents[intentID]
	if !ok {
		writeError(w, http.StatusNotFound, "Intent not found")
		return
	}

	if intent.Status != IntentPlanned {
		writeError(w, http.StatusConflict, fmt.Sprintf("Cannot submit intent in status %q; must be planned first", intent.Status))
		return
	}

	intent.Status = IntentSubmitted
	intent.UpdatedAt = nowRFC3339()

	s.storeOperatorReceipt("operator_submit_intent", map[string]any{
		"intent_id": intentID,
	})

	writeJSON(w, http.StatusOK, intent)
}

// ---------------------------------------------------------------------------
// Approval Handlers
// ---------------------------------------------------------------------------

// handleListApprovals handles GET /api/approvals.
func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.operatorMu.RLock()
	approvals := make([]operatorApproval, 0, len(s.approvals))
	for _, a := range s.approvals {
		approvals = append(approvals, *a)
	}
	s.operatorMu.RUnlock()

	writeJSON(w, http.StatusOK, approvalsListResponse{
		Approvals: approvals,
		Total:     len(approvals),
	})
}

// handleApproveIntent handles POST /api/approvals/{id}/approve.
func (s *Server) handleApproveIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	intentID := extractIntentIDFromSubpath(r.URL.Path)
	if intentID == "" {
		writeError(w, http.StatusBadRequest, "Missing intent ID")
		return
	}

	var req approvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}
	if req.DecidedBy == "" {
		req.DecidedBy = "operator"
	}

	s.operatorMu.Lock()
	defer s.operatorMu.Unlock()

	intent, ok := s.intents[intentID]
	if !ok {
		writeError(w, http.StatusNotFound, "Intent not found")
		return
	}

	if intent.Status != IntentSubmitted {
		writeError(w, http.StatusConflict, fmt.Sprintf("Cannot approve intent in status %q; must be submitted", intent.Status))
		return
	}

	// Record decision
	if req.Decision == DecisionApprove {
		intent.Status = IntentApproved
	} else {
		intent.Status = IntentRejected
	}
	intent.UpdatedAt = nowRFC3339()

	approval := &operatorApproval{
		ApprovalID: generateID("apr"),
		IntentID:   intentID,
		Decision:   req.Decision,
		Reason:     req.Reason,
		DecidedBy:  req.DecidedBy,
		DecidedAt:  nowRFC3339(),
	}

	receiptID := s.storeOperatorReceipt("operator_approve_intent", map[string]any{
		"intent_id":   intentID,
		"decision":    string(req.Decision),
		"decided_by":  req.DecidedBy,
		"approval_id": approval.ApprovalID,
	})
	approval.ReceiptID = receiptID

	s.approvals[approval.ApprovalID] = approval

	writeJSON(w, http.StatusOK, approval)
}

// ---------------------------------------------------------------------------
// Run Handlers (operator-initiated)
// ---------------------------------------------------------------------------

// handleCreateRun handles POST /api/runs — creates a run from an approved intent.
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req createRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.IntentID == "" {
		writeError(w, http.StatusBadRequest, "intent_id is required")
		return
	}

	s.operatorMu.Lock()
	defer s.operatorMu.Unlock()

	intent, ok := s.intents[req.IntentID]
	if !ok {
		writeError(w, http.StatusNotFound, "Intent not found")
		return
	}

	if intent.Status != IntentApproved {
		writeError(w, http.StatusConflict, fmt.Sprintf("Cannot start run for intent in status %q; must be approved", intent.Status))
		return
	}

	now := nowRFC3339()
	run := &operatorRunState{
		RunID:     generateID("run"),
		IntentID:  req.IntentID,
		State:     RunStateRunning,
		PlanHash:  intent.PlanHash,
		CreatedAt: now,
		UpdatedAt: now,
	}

	intent.Status = IntentExecuting
	intent.UpdatedAt = now

	receiptID := s.storeOperatorReceipt("operator_create_run", map[string]any{
		"run_id":    run.RunID,
		"intent_id": req.IntentID,
		"plan_hash": intent.PlanHash,
	})
	run.ReceiptID = receiptID

	s.operatorRuns[run.RunID] = run

	writeJSON(w, http.StatusCreated, run)
}

// handleRunControl handles POST /api/runs/{id}/control — pause/cancel/retry.
func (s *Server) handleRunControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	runID := extractIntentIDFromSubpath(r.URL.Path)
	if runID == "" {
		writeError(w, http.StatusBadRequest, "Missing run ID")
		return
	}

	var req runControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	s.operatorMu.Lock()
	defer s.operatorMu.Unlock()

	run, ok := s.operatorRuns[runID]
	if !ok {
		writeError(w, http.StatusNotFound, "Run not found")
		return
	}

	var newState RunState
	switch req.Action {
	case RunActionPause:
		if run.State != RunStateRunning {
			writeError(w, http.StatusConflict, "Can only pause a running run")
			return
		}
		newState = RunStatePaused
	case RunActionCancel:
		if run.State != RunStateRunning && run.State != RunStatePaused {
			writeError(w, http.StatusConflict, "Can only cancel a running or paused run")
			return
		}
		newState = RunStateCancelled
	case RunActionResume:
		if run.State != RunStatePaused {
			writeError(w, http.StatusConflict, "Can only resume a paused run")
			return
		}
		newState = RunStateRunning
	case RunActionRetry:
		if run.State != RunStateFailed && run.State != RunStateCancelled {
			writeError(w, http.StatusConflict, "Can only retry a failed or canceled run")
			return
		}
		newState = RunStateRunning
	default:
		writeError(w, http.StatusBadRequest, "Invalid action; must be pause, cancel, resume, or retry")
		return
	}

	run.State = newState
	run.UpdatedAt = nowRFC3339()

	s.storeOperatorReceipt("operator_run_control", map[string]any{
		"run_id": runID,
		"action": string(req.Action),
		"reason": req.Reason,
	})

	writeJSON(w, http.StatusOK, run)
}

// handleRunReceipts handles GET /api/runs/{id}/receipts — receipt timeline for a run.
func (s *Server) handleRunReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	runID := extractIntentIDFromSubpath(r.URL.Path)
	if runID == "" {
		writeError(w, http.StatusBadRequest, "Missing run ID")
		return
	}

	s.operatorMu.RLock()
	run, ok := s.operatorRuns[runID]
	s.operatorMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "Run not found")
		return
	}

	// Find all receipts that reference this run or its intent
	ctx := r.Context()
	allReceipts, err := s.receiptStore.List(ctx, 1000)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch receipts")
		return
	}

	matchingReceipts := make([]*contracts.Receipt, 0)
	for _, rcpt := range allReceipts {
		if meta, ok := rcpt.Metadata["run_id"]; ok && fmt.Sprintf("%v", meta) == runID {
			matchingReceipts = append(matchingReceipts, rcpt)
			continue
		}
		if meta, ok := rcpt.Metadata["intent_id"]; ok && fmt.Sprintf("%v", meta) == run.IntentID {
			matchingReceipts = append(matchingReceipts, rcpt)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"run_id":   runID,
		"receipts": matchingReceipts,
		"total":    len(matchingReceipts),
	})
}

// handleRunReplay handles POST /api/runs/{id}/replay — compare planned vs executed.
func (s *Server) handleRunReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	runID := extractIntentIDFromSubpath(r.URL.Path)
	if runID == "" {
		writeError(w, http.StatusBadRequest, "Missing run ID")
		return
	}

	s.operatorMu.RLock()
	run, ok := s.operatorRuns[runID]
	if !ok {
		s.operatorMu.RUnlock()
		writeError(w, http.StatusNotFound, "Run not found")
		return
	}

	intent, intentOK := s.intents[run.IntentID]
	s.operatorMu.RUnlock()

	if !intentOK {
		writeError(w, http.StatusNotFound, "Intent not found for run")
		return
	}

	// Generate planned steps (deterministic from intent)
	plannedSteps := generatePlanSteps(intent)

	// Simulate executed steps (in real system, would come from execution log)
	executedSteps := make([]operatorPlanStep, len(plannedSteps))
	copy(executedSteps, plannedSteps)

	// Build diffs
	diffs := make([]replayDiff, 0)
	matchCount := 0
	for i, ps := range plannedSteps {
		executed := ""
		match := false
		if i < len(executedSteps) {
			executed = executedSteps[i].Action + ": " + executedSteps[i].Target
			match = ps.Action == executedSteps[i].Action && ps.Target == executedSteps[i].Target
		}
		if match {
			matchCount++
		}
		diffs = append(diffs, replayDiff{
			StepID:   ps.StepID,
			Planned:  ps.Action + ": " + ps.Target,
			Executed: executed,
			Match:    match,
		})
	}

	matchPct := 0.0
	if len(plannedSteps) > 0 {
		matchPct = float64(matchCount) / float64(len(plannedSteps)) * 100
	}

	receiptID := s.storeOperatorReceipt("operator_replay", map[string]any{
		"run_id":           runID,
		"match_percentage": matchPct,
	})

	result := replayResult{
		RunID:           runID,
		PlannedSteps:    len(plannedSteps),
		ExecutedSteps:   len(executedSteps),
		Diffs:           diffs,
		MatchPercentage: matchPct,
		ReceiptID:       receiptID,
	}

	writeJSON(w, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Intent routing helper
// ---------------------------------------------------------------------------

// handleIntentsRouter routes requests to /api/intents/* to the correct handler.
func (s *Server) handleIntentsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/intents")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "" || path == "/":
		// /api/intents — list or create
		if r.Method == http.MethodPost {
			s.handleCreateIntent(w, r)
		} else {
			s.handleListIntents(w, r)
		}
	case strings.HasSuffix(path, "/plan"):
		s.handlePlanIntent(w, r)
	case strings.HasSuffix(path, "/submit"):
		s.handleSubmitIntent(w, r)
	default:
		// /api/intents/{id}
		s.handleGetIntent(w, r)
	}
}

// handleApprovalsRouter routes requests to /api/approvals/*.
func (s *Server) handleApprovalsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/approvals")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "" || path == "/":
		s.handleListApprovals(w, r)
	case strings.HasSuffix(path, "/approve"):
		s.handleApproveIntent(w, r)
	default:
		writeError(w, http.StatusNotFound, "Unknown approvals endpoint")
	}
}

// handleOperatorRunsRouter routes operator-specific run requests.
// This handles POST /api/runs (create) and /api/runs/{id}/control, /api/runs/{id}/receipts, /api/runs/{id}/replay.
func (s *Server) handleOperatorRunsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runs")
	path = strings.TrimSuffix(path, "/")

	switch {
	case (path == "" || path == "/") && r.Method == http.MethodPost:
		s.handleCreateRun(w, r)
	case strings.HasSuffix(path, "/control"):
		s.handleRunControl(w, r)
	case strings.HasSuffix(path, "/receipts"):
		s.handleRunReceipts(w, r)
	case strings.HasSuffix(path, "/replay"):
		s.handleRunReplay(w, r)
	default:
		// Not an operator-run route, let existing handlers deal with it
		if r.Method == http.MethodGet && path == "" {
			s.handleRunsListAPI(w, r)
		} else if r.Method == http.MethodGet {
			s.handleRunDetailAPI(w, r)
		} else {
			writeError(w, http.StatusNotFound, "Unknown runs endpoint")
		}
	}
}

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// extractPathSegment extracts the segment after /{resource}/ from a URL path.
func extractPathSegment(path string, resource string) string {
	prefix := "/api/" + resource + "/"
	trimmed := strings.TrimPrefix(path, prefix)
	if trimmed == path {
		return ""
	}
	// Remove any further sub-paths
	if idx := strings.Index(trimmed, "/"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	return trimmed
}

// extractIntentIDFromSubpath extracts the ID from paths like /api/intents/{id}/plan.
func extractIntentIDFromSubpath(path string) string {
	// Works for /api/intents/{id}/plan, /api/approvals/{id}/approve, /api/runs/{id}/control etc.
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// parts: [api, intents, {id}, plan] or [api, intents, {id}]
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// ---------------------------------------------------------------------------
// Plan generation (deterministic from intent)
// ---------------------------------------------------------------------------

func generatePlanSteps(intent *operatorIntent) []operatorPlanStep {
	steps := []operatorPlanStep{
		{
			StepID:      "step-1",
			Action:      "validate",
			Target:      intent.Type,
			Description: "Validate " + intent.Type + " parameters",
			Estimated:   "2s",
		},
		{
			StepID:      "step-2",
			Action:      "prepare",
			Target:      intent.Type,
			Description: "Prepare execution environment for " + intent.Description,
			Estimated:   "5s",
		},
		{
			StepID:      "step-3",
			Action:      "execute",
			Target:      intent.Type,
			Description: "Execute " + intent.Description,
			Estimated:   "10s",
		},
		{
			StepID:      "step-4",
			Action:      "verify",
			Target:      intent.Type,
			Description: "Verify execution results",
			Estimated:   "3s",
		},
	}

	// Add step for high-risk types
	if intent.Type == "deploy" || intent.Type == "delete" {
		steps = append(steps, operatorPlanStep{
			StepID:      "step-5",
			Action:      "rollback_plan",
			Target:      intent.Type,
			Description: "Prepare rollback plan for " + intent.Description,
			Estimated:   "5s",
		})
	}

	return steps
}
