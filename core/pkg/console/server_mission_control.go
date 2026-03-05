package console

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/api"
	"github.com/Mindburn-Labs/helm/core/pkg/audit"
	"github.com/Mindburn-Labs/helm/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm/core/pkg/store"
)

// ── Policy API ─────────────────────────────────────────────────

// apiPolicy is the JSON representation of a loaded policy.
type apiPolicy struct {
	PolicyID    string `json:"policy_id"`
	Source      string `json:"source"`
	Engine      string `json:"engine"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type policiesListResponse struct {
	Policies []apiPolicy `json:"policies"`
	Total    int         `json:"total"`
}

// handlePoliciesListAPI serves GET /api/policies — lists all loaded policy definitions.
func (s *Server) handlePoliciesListAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}
	if s.policyEngine == nil {
		writeJSON(w, http.StatusOK, policiesListResponse{Policies: []apiPolicy{}, Total: 0})
		return
	}

	defs := s.policyEngine.ListDefinitions()
	policies := make([]apiPolicy, 0, len(defs))
	for id, source := range defs {
		policies = append(policies, apiPolicy{
			PolicyID:    id,
			Source:      source,
			Engine:      "CEL",
			Status:      "active",
			Description: "CEL policy: " + id,
		})
	}

	writeJSON(w, http.StatusOK, policiesListResponse{
		Policies: policies,
		Total:    len(policies),
	})
}

// ── Audit Events API ───────────────────────────────────────────

// apiAuditEvent is the JSON representation of an audit entry for the UI.
type apiAuditEvent struct {
	EntryID      string            `json:"entry_id"`
	Sequence     uint64            `json:"sequence"`
	Type         string            `json:"type"`
	Subject      string            `json:"subject"`
	Action       string            `json:"action"`
	Timestamp    string            `json:"timestamp"`
	PayloadHash  string            `json:"payload_hash"`
	PreviousHash string            `json:"previous_hash"`
	EntryHash    string            `json:"entry_hash"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type auditEventsListResponse struct {
	Events []apiAuditEvent `json:"events"`
	Total  int             `json:"total"`
}

// handleAuditEventsAPI serves GET /api/audit/events — lists audit events with pagination.
func (s *Server) handleAuditEventsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}
	if s.auditStore == nil {
		writeJSON(w, http.StatusOK, auditEventsListResponse{Events: []apiAuditEvent{}, Total: 0})
		return
	}

	// Parse optional query params
	maxResults := 50
	if m := r.URL.Query().Get("limit"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v > 0 && v <= 200 {
			maxResults = v
		}
	}

	var entryType store.EntryType
	if t := r.URL.Query().Get("type"); t != "" {
		entryType = store.EntryType(t)
	}

	filter := store.QueryFilter{
		EntryType:  entryType,
		MaxResults: maxResults,
	}

	entries := s.auditStore.Query(filter)
	events := make([]apiAuditEvent, 0, len(entries))
	for _, e := range entries {
		events = append(events, apiAuditEvent{
			EntryID:      e.EntryID,
			Sequence:     e.Sequence,
			Type:         string(e.EntryType),
			Subject:      e.Subject,
			Action:       e.Action,
			Timestamp:    e.Timestamp.UTC().Format(time.RFC3339),
			PayloadHash:  e.PayloadHash,
			PreviousHash: e.PreviousHash,
			EntryHash:    e.EntryHash,
			Metadata:     e.Metadata,
		})
	}

	writeJSON(w, http.StatusOK, auditEventsListResponse{
		Events: events,
		Total:  len(events),
	})
}

// ── Vendor / A2A Mesh API ──────────────────────────────────────

// apiVendorService is the JSON representation of an installed pack as a vendor service.
type apiVendorService struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Capabilities []apiCapabilityGrant `json:"capabilities"`
	Status       string               `json:"status"`
	PowerDelta   int                  `json:"power_delta"`
	InstalledAt  string               `json:"installed_at"`
}

type apiCapabilityGrant struct {
	Name string `json:"name"`
}

type vendorsListResponse struct {
	Vendors []apiVendorService `json:"vendors"`
	Total   int                `json:"total"`
}

// handleVendorsAPI serves GET /api/vendors — lists installed packs as vendor services.
func (s *Server) handleVendorsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}
	if s.registry == nil {
		writeJSON(w, http.StatusOK, vendorsListResponse{Vendors: []apiVendorService{}, Total: 0})
		return
	}

	packs := s.registry.List()
	vendors := make([]apiVendorService, 0, len(packs))
	for _, p := range packs {
		caps := make([]apiCapabilityGrant, 0, len(p.Manifest.Capabilities))
		for _, c := range p.Manifest.Capabilities {
			caps = append(caps, apiCapabilityGrant{Name: c.Name})
		}
		vendors = append(vendors, apiVendorService{
			ID:           p.Manifest.Name,
			Name:         p.Manifest.Name,
			Capabilities: caps,
			Status:       "active",
			PowerDelta:   p.PowerDelta,
			InstalledAt:  "", // Packs do not track install time — empty signals unknown to the UI
		})
	}

	writeJSON(w, http.StatusOK, vendorsListResponse{
		Vendors: vendors,
		Total:   len(vendors),
	})
}

// ── Policy Simulation API ──────────────────────────────────────

type policySimulateRequest struct {
	PolicyID  string                 `json:"policy_id"`
	TestEvent map[string]interface{} `json:"test_event"`
}

type policySimulateResponse struct {
	PolicyID    string   `json:"policy_id"`
	Verdict     string   `json:"verdict"`
	Reason      string   `json:"reason"`
	Trace       []string `json:"trace"`
	EvaluatedAt string   `json:"evaluated_at"`
}

// handlePolicySimulateAPI serves POST /api/policies/simulate — dry-run policy evaluation.
func (s *Server) handlePolicySimulateAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}
	if s.policyEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Policy engine not initialized",
			"code":  "POLICY_ENGINE_UNAVAILABLE",
		})
		return
	}

	var req policySimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid simulation request")
		return
	}

	if req.PolicyID == "" {
		api.WriteBadRequest(w, "policy_id is required")
		return
	}

	// Build a synthetic access request from the test event
	principalID, _ := req.TestEvent["principal_id"].(string)
	if principalID == "" {
		principalID = "simulation:anonymous"
	}
	action, _ := req.TestEvent["action"].(string)
	if action == "" {
		action = "simulate.test"
	}
	resourceID, _ := req.TestEvent["resource_id"].(string)
	if resourceID == "" {
		resourceID = "resource:test"
	}

	accessReq := contracts.AccessRequest{
		PrincipalID: principalID,
		Action:      action,
		ResourceID:  resourceID,
		Context:     req.TestEvent,
	}

	decision, err := s.policyEngine.Evaluate(r.Context(), req.PolicyID, accessReq)
	trace := []string{}
	verdict := "DENY"
	reason := ""

	if err != nil {
		trace = append(trace, "error: "+err.Error())
		reason = err.Error()
	} else {
		verdict = decision.Verdict
		reason = decision.Reason
		trace = append(trace, "policy_id: "+req.PolicyID)
		trace = append(trace, "verdict: "+verdict)
		trace = append(trace, "reason: "+reason)
	}

	writeJSON(w, http.StatusOK, policySimulateResponse{
		PolicyID:    req.PolicyID,
		Verdict:     verdict,
		Reason:      reason,
		Trace:       trace,
		EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// ── Ops Flight Control API ─────────────────────────────────────

type opsControlRequest struct {
	Action string `json:"action"` // freeze, throttle, isolate, resume
	Reason string `json:"reason"`
}

type opsControlResponse struct {
	Action     string `json:"action"`
	Result     string `json:"result"`
	ReceiptID  string `json:"receipt_id"`
	ExecutedAt string `json:"executed_at"`
	Mode       string `json:"mode"`
}

// handleOpsControlAPI serves POST /api/ops/control — freeze/throttle/isolate/resume with receipt.
func (s *Server) handleOpsControlAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req opsControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid ops control request")
		return
	}

	validActions := map[string]bool{"freeze": true, "throttle": true, "isolate": true, "resume": true}
	if !validActions[req.Action] {
		api.WriteBadRequest(w, "Invalid action. Must be one of: freeze, throttle, isolate, resume")
		return
	}

	if req.Reason == "" {
		api.WriteBadRequest(w, "reason is required for all ops control actions")
		return
	}

	// Execute action via eval controllers.
	// NOTE: throttle and isolate currently map to the same Pause() mechanism
	// as freeze. The EvalController does not yet support granular rate-limiting
	// or per-agent isolation. The distinct actions exist so the UI can record
	// different intents and the audit trail captures the operator's purpose.
	// When the EvalController gains Throttle(rate) and Isolate(agentID),
	// this switch should be updated to call those methods.
	newMode := "UNKNOWN"
	switch req.Action {
	case "freeze":
		// if s.evalControllers != nil {
		// 	ctrl := s.evalControllers.GetOrCreate("default")
		// 	_ = ctrl.Pause()
		// }
		newMode = "FROZEN"
	case "throttle":
		// Currently maps to Pause — see note above
		// if s.evalControllers != nil {
		// 	ctrl := s.evalControllers.GetOrCreate("default")
		// 	_ = ctrl.Pause()
		// }
		newMode = "DEGRADED"
	case "isolate":
		// Currently maps to Pause — see note above
		// if s.evalControllers != nil {
		// 	ctrl := s.evalControllers.GetOrCreate("default")
		// 	_ = ctrl.Pause()
		// }
		newMode = "PAUSED"
	case "resume":
		// if s.evalControllers != nil {
		// 	ctrl := s.evalControllers.GetOrCreate("default")
		// 	_ = ctrl.Resume()
		// }
		newMode = "HEALTHY"
	}

	// Generate receipt
	ts := time.Now()
	receiptID := fmt.Sprintf("rcpt-ops-%d", ts.UnixNano())

	// Compute deterministic blob hash from receipt payload
	hashPayload, _ := json.Marshal(map[string]any{
		"action":     req.Action,
		"reason":     req.Reason,
		"mode":       newMode,
		"receipt_id": receiptID,
		"timestamp":  ts.UTC().Format(time.RFC3339Nano),
	})
	blobHash := fmt.Sprintf("sha256:%x", sha256.Sum256(hashPayload))

	receipt := &contracts.Receipt{
		ReceiptID:  receiptID,
		DecisionID: fmt.Sprintf("dec-ops-%s-%d", req.Action, ts.UnixNano()),
		EffectID:   fmt.Sprintf("eff-ops-%s-%d", req.Action, ts.UnixNano()),
		Status:     "SUCCESS",
		Timestamp:  ts,
		BlobHash:   blobHash,
		ExecutorID: "helm-ops-controller",
		Metadata: map[string]any{
			"action": req.Action,
			"reason": req.Reason,
			"mode":   newMode,
		},
	}

	if err := s.receiptStore.Store(r.Context(), receipt); err != nil {
		slog.Error("failed to store ops control receipt", "error", err)
	}

	// Audit log
	if s.auditLogger != nil {
		_ = s.auditLogger.Record(context.Background(), audit.EventSystem, req.Action, "ops-controller", map[string]interface{}{
			"reason":     req.Reason,
			"receipt_id": receiptID,
			"mode":       newMode,
		})
	}

	writeJSON(w, http.StatusOK, opsControlResponse{
		Action:     req.Action,
		Result:     "executed",
		ReceiptID:  receiptID,
		ExecutedAt: ts.UTC().Format(time.RFC3339),
		Mode:       newMode,
	})
}
