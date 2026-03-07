package console

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/audit"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/contracts"
)

// Enterprise API Implementation

// handleAuditExportAPI exports audit logs for the tenant.
func (s *Server) handleAuditExportAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.WriteMethodNotAllowed(w)
		return
	}

	// 1. RBAC Check (Requires 'admin' role)
	ctx := r.Context()
	userID := "user:unknown"
	if p, err := auth.GetPrincipal(ctx); err == nil {
		userID = p.GetID()
	}

	accessReq := contracts.AccessRequest{
		PrincipalID: userID,
		Action:      "export.audit_logs",
		ResourceID:  "tenant:" + auth.MustGetTenantID(ctx),
		Context:     map[string]any{},
	}

	decision, err := s.policyEngine.Evaluate(ctx, "admin_policy", accessReq) // Reuse admin_policy for simplicity
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// Log Policy Decision Receipt?
	// The policy evaluation happens, but we should probably record that this check happened.
	// For now, if denied, return 403.
	if decision.Verdict != "ALLOW" {
		api.WriteForbidden(w, decision.Reason)
		return
	}

	// 2. Prepare Export
	tenantID := auth.MustGetTenantID(ctx) // Safe because Auth middleware ran

	req := audit.ExportRequest{
		TenantID:  tenantID,
		StartTime: time.Now().Add(-24 * time.Hour), // Default 24h
		EndTime:   time.Now(),
	}

	exporter := s.auditExport
	if exporter == nil {
		api.WriteInternal(w, errors.New("fail-closed: audit exporter not configured"))
		return
	}
	zipBytes, checksum, err := exporter.GeneratePack(ctx, req)
	if err != nil {
		api.WriteInternal(w, err)
		return
	}

	// 3. Log Audit Event
	_ = s.auditLogger.Record(ctx, audit.EventAccess, "export_audit_logs", "evidence_pack", map[string]interface{}{
		"checksum": checksum,
		"size":     len(zipBytes),
	})

	// 4. Return Zip
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"audit-%s-%s.zip\"", tenantID, time.Now().Format("20060102")))
	w.Header().Set("X-Audit-Checksum", checksum)
	_, _ = w.Write(zipBytes)
}
